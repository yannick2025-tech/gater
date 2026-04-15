package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/yannick2025-tech/nts-gater/internal/config"
	"github.com/yannick2025-tech/nts-gater/internal/dispatcher"
	"github.com/yannick2025-tech/nts-gater/internal/handlers"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	"github.com/yannick2025-tech/nts-gater/internal/server"
	"github.com/yannick2025-tech/nts-gater/internal/session"
	"github.com/yannick2025-tech/nts-gater/internal/validator"
	logging "github.com/yannick2025-tech/gwc-logging"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := initLogger(cfg.Log)
	defer logger.Close()

	logger.Info("nts-gater starting...")
	logger.Infof("config loaded: server=%s:%d", cfg.Server.Host, cfg.Server.Port)

	proto := standard.New()
	logger.Infof("protocol: name=%s, version=0x%02X, registered_messages=%d",
		proto.Name(), proto.Version(), len(proto.Registry().AllSpecs()))

	sessMgr := session.NewManager(proto, cfg.Server.HeartbeatTimeout)
	frameValidator := validator.New(proto)
	dp := dispatcher.New(proto, sessMgr, logger)
	handlers.Register(dp, sessMgr, logger)

	srv := server.New(cfg, proto, logger)
	srv.OnMessage(func(conn *server.Connection, header types.MessageHeader, data []byte) {
		onMessage(conn, header, data, proto, sessMgr, frameValidator, dp, logger)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		logger.Fatalf("failed to start server: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Infof("received signal %v, shutting down...", sig)
	srv.Stop()
}

func onMessage(conn *server.Connection, header types.MessageHeader, data []byte,
	proto types.Protocol, sessMgr *session.SessionManager,
	validator *validator.FrameValidator, dp *dispatcher.Dispatcher,
	logger logging.Logger,
) {
	// 1. 帧头校验
	if verr := validator.ValidateHeader(header); verr != nil {
		logger.Warnf("[%s] frame validation failed: %s", conn.ID, verr.Message)
		return
	}

	// 2. 获取或创建会话
	sess, ok := sessMgr.GetByPostNo(header.PostNo)
	if !ok {
		sess, err := sessMgr.Create(header.PostNo, conn.ID)
		if err != nil {
			logger.Errorf("[%s] create session failed: %v", conn.ID, err)
			return
		}
		logger.Infof("[%s] new session: %s, postNo=%d", conn.ID, sess.ID, header.PostNo)
	}
	sess.UpdateActive()

	// 3. 解密数据域（如需加密且已认证，或使用固定密钥的功能码）
	decryptFn := sess.GetDecryptFn()
	if proto.IsFixedKeyFuncCode(header.FuncCode) || sess.GetAuthState() == session.Authenticated {
		if header.EncryptFlag == 0x01 && decryptFn != nil {
			decrypted, err := decryptFn(data)
			if err != nil {
				logger.Warnf("[%s] decrypt func=0x%02X failed: %v", conn.ID, header.FuncCode, err)
				return
			}
			data = decrypted
		}
	}

	// 4. 消息解码
	dir := directionOf(header)
	msg, ok := proto.Registry().Create(header.FuncCode, dir)
	if !ok {
		logger.Warnf("[%s] unregistered func=0x%02X dir=%v", conn.ID, header.FuncCode, dir)
		return
	}
	if len(data) > 0 {
		if err := msg.Decode(data); err != nil {
			logger.Warnf("[%s] decode func=0x%02X failed: %v", conn.ID, header.FuncCode, err)
			return
		}
	}

	// 5. 字段校验
	if errs := validator.ValidateMessage(msg); len(errs) > 0 {
		for _, e := range errs {
			logger.Warnf("[%s] validation: field=%s code=%s msg=%s", conn.ID, e.Field, e.Code, e.Message)
		}
	}

	// 6. 日志
	logger.Infof("[%s] recv func=0x%02X name=%s postNo=%d charger=%d json=%+v",
		conn.ID, header.FuncCode, msg.Spec().Name, header.PostNo, header.Charger, msg.ToJSONMap())

	// 7. 分发
	replyFn := func(replyHeader types.MessageHeader, replyData []byte) error {
		// 自动加密
		encryptFn := sess.GetEncryptFn()
		replyHeader.PostNo = header.PostNo
		replyHeader.Charger = header.Charger
		replyHeader.Version = proto.Version()
		replyHeader.StartByte = proto.FrameConfig().StartByte
		return conn.Send(replyHeader, replyData, encryptFn)
	}

	ctx := &dispatcher.Context{
		PostNo:   header.PostNo,
		Charger:  header.Charger,
		FuncCode: header.FuncCode,
		Dir:      dir,
		Data:     data,
		Message:  msg,
		Session:  sess,
		Logger:   logger,
		Reply:    replyFn,
		Proto:    proto,
	}

	if dp.HasHandler(header.FuncCode, dir) {
		if err := dp.Dispatch(ctx); err != nil {
			logger.Errorf("[%s] handle func=0x%02X error: %v", conn.ID, header.FuncCode, err)
		}
	}
}

func directionOf(header types.MessageHeader) types.Direction {
	return types.DirectionUpload
}

func initLogger(cfg config.LogConfig) logging.Logger {
	logCfg := &logging.Config{
		LevelStr:      cfg.Level,
		Format:        logging.TextFormat,
		Target:        logging.StdoutTarget,
		Filename:      cfg.Filename,
		MaxSize:       cfg.MaxSize,
		MaxBackups:    cfg.MaxBackups,
		MaxAge:        cfg.MaxAge,
		Compress:      cfg.Compress,
		LocalTime:     true,
		FlushInterval: 5,
	}

	logger, err := logging.NewZapLogger(logCfg)
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}
	return logger
}
