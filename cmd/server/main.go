package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/yannick2025-tech/nts-gater/internal/config"
	"github.com/yannick2025-tech/nts-gater/internal/dispatcher"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	"github.com/yannick2025-tech/nts-gater/internal/server"
	"github.com/yannick2025-tech/nts-gater/internal/session"
	"github.com/yannick2025-tech/nts-gater/internal/validator"
	logging "github.com/yannick2025-tech/gwc-logging"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger := initLogger(cfg.Log)
	defer logger.Close()

	logger.Info("nts-gater starting...")
	logger.Infof("config loaded: server=%s:%d", cfg.Server.Host, cfg.Server.Port)

	// 初始化协议
	proto := standard.New()
	logger.Infof("protocol: name=%s, version=0x%02X, registered_messages=%d",
		proto.Name(), proto.Version(), len(proto.Registry().AllSpecs()))

	// 初始化会话管理器
	sessMgr := session.NewManager(proto, cfg.Server.HeartbeatTimeout)

	// 初始化验证器
	frameValidator := validator.New(proto)

	// 初始化分发器
	dp := dispatcher.New(proto, sessMgr, logger)
	registerHandlers(dp)

	// 创建TCP服务器
	srv := server.New(cfg, proto, logger)
	srv.OnMessage(func(conn *server.Connection, header types.MessageHeader, data []byte) {
		onMessage(conn, header, data, proto, sessMgr, frameValidator, dp, logger)
	})

	// 启动
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		logger.Fatalf("failed to start server: %v", err)
	}

	// 等待退出信号
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

	// 2. 确定消息方向
	dir := directionOf(header)

	// 3. 获取或创建会话
	sess, ok := sessMgr.GetByPostNo(header.PostNo)
	if !ok {
		sess, err := sessMgr.Create(header.PostNo, conn.ID)
		if err != nil {
			logger.Errorf("[%s] create session failed: %v", conn.ID, err)
			return
		}
		logger.Infof("[%s] new session created: %s, postNo=%d", conn.ID, sess.ID, header.PostNo)
	}

	sess.UpdateActive()

	// 4. 检查认证状态（仅对需要认证的功能码）
	// 0x0A/0x0B/0x21 使用固定密钥，不需要已认证状态
	if !proto.IsFixedKeyFuncCode(header.FuncCode) && sess.GetAuthState() != session.Authenticated {
		logger.Warnf("[%s] session not authenticated, func=0x%02X", conn.ID, header.FuncCode)
		return
	}

	// 5. 创建消息实例并解码
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

	// 6. 字段校验
	if errs := validator.ValidateMessage(msg); len(errs) > 0 {
		for _, e := range errs {
			logger.Warnf("[%s] validation: field=%s code=%s msg=%s", conn.ID, e.Field, e.Code, e.Message)
		}
	}

	// 7. 打印日志
	logger.Infof("[%s] recv func=0x%02X name=%s postNo=%d charger=%d json=%+v",
		conn.ID, header.FuncCode, msg.Spec().Name, header.PostNo, header.Charger, msg.ToJSONMap())

	// 8. 分发到业务处理器
	ctx := &dispatcher.Context{
		PostNo:   header.PostNo,
		Charger:  header.Charger,
		FuncCode: header.FuncCode,
		Dir:      dir,
		Data:     data,
		Message:  msg,
		Session:  sess,
	}

	if dp.HasHandler(header.FuncCode, dir) {
		if err := dp.Dispatch(ctx); err != nil {
			logger.Errorf("[%s] handle func=0x%02X error: %v", conn.ID, header.FuncCode, err)
		}
	} else {
		logger.Debugf("[%s] no handler for func=0x%02X, skipping dispatch", conn.ID, header.FuncCode)
	}
}

func directionOf(header types.MessageHeader) types.Direction {
	return types.DirectionUpload
}

// registerHandlers 注册业务处理器
func registerHandlers(dp *dispatcher.Dispatcher) {
	// TODO: 在 M3 中注册具体的业务处理器
	// 示例:
	// dp.RegisterFunc(types.FuncHeartbeat, types.DirectionUpload, handleHeartbeat)
	// dp.RegisterFunc(types.FuncAuthRandom, types.DirectionUpload, handleAuthRandom)
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
