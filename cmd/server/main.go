// Package main provides the nts-gater server entry point.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	logging "github.com/yannick2025-tech/gwc-logging"
	"github.com/yannick2025-tech/nts-gater/internal/api"
	"github.com/yannick2025-tech/nts-gater/internal/config"
	"github.com/yannick2025-tech/nts-gater/internal/database"
	"github.com/yannick2025-tech/nts-gater/internal/dispatcher"
	"github.com/yannick2025-tech/nts-gater/internal/handlers"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	"github.com/yannick2025-tech/nts-gater/internal/recorder"
	"github.com/yannick2025-tech/nts-gater/internal/report"
	"github.com/yannick2025-tech/nts-gater/internal/scenario"
	"github.com/yannick2025-tech/nts-gater/internal/server"
	"github.com/yannick2025-tech/nts-gater/internal/session"
	"github.com/yannick2025-tech/nts-gater/internal/validator"
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
	logger.Infof("config loaded: server=%s:%d, http=%d", cfg.Server.Host, cfg.Server.Port, cfg.Server.HTTPPort)

	// 初始化数据库
	if err := database.Init(cfg.Database); err != nil {
		logger.Fatalf("failed to init database: %v", err)
	}
	defer database.Close()
	logger.Info("database initialized")

	// 填充测试种子数据（开发阶段：模拟已结束的会话，供前端验证）
	if err := report.SeedTestData(); err != nil {
		logger.Warnf("seed test data: %v", err)
	} else {
		logger.Info("seed test data ready")
	}

	proto := standard.New()
	logger.Infof("protocol: name=%s, version=0x%02X, registered_messages=%d",
		proto.Name(), proto.Version(), len(proto.Registry().AllSpecs()))

	sessMgr := session.NewManager(proto, cfg.Server.HeartbeatTimeout)
	frameValidator := validator.New(proto)
	dp := dispatcher.New(proto, sessMgr, logger)
	handlers.Register(dp, sessMgr, logger)

	// TCP服务器
	srv := server.New(cfg, proto, logger)

	// 场景引擎（需在TCP回调之前创建，因为回调中引用它）
	scenarioEngine := scenario.NewEngine(sessMgr, srv, proto, logger)

	srv.OnMessage(func(conn *server.Connection, header types.MessageHeader, data []byte) {
		onMessage(conn, header, data, proto, sessMgr, frameValidator, dp, scenarioEngine, logger)
	})
	srv.OnDisconnect(func(conn *server.Connection, postNo uint32) {
		onDisconnect(conn, postNo, sessMgr, proto, scenarioEngine, logger)
	})

	// HTTP服务器
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()
	engine.Use(corsMiddleware())
	router := api.NewRouter(sessMgr, scenarioEngine)
	router.Setup(engine)

	// API文档：YAML文件 + Swagger UI
	engine.Static("/docs", "./docs/api")
	engine.GET("/swagger", serveSwaggerUI)

	// 前端静态文件（生产模式：服务 web/dist 目录）
	engine.Static("/assets", "./web/dist/assets")
	engine.NoRoute(func(c *gin.Context) {
		// SPA fallback: 非API请求返回 index.html
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
			return
		}
		c.File("./web/dist/index.html")
	})
	httpSrv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HTTPPort),
		Handler: engine,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		logger.Fatalf("failed to start TCP server: %v", err)
	}

	go func() {
		logger.Infof("HTTP server listening on %s", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("HTTP server error: %v", err)
		}
	}()

	// 心跳超时检查：定期扫描超时会话并断开连接
	go heartbeatCheckLoop(ctx, sessMgr, srv, proto, scenarioEngine, logger)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Infof("received signal %v, shutting down...", sig)

	srv.Stop()
	httpSrv.Close()
}

// corsMiddleware 跨域中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func onMessage(conn *server.Connection, header types.MessageHeader, data []byte,
	proto types.Protocol, sessMgr *session.SessionManager,
	frameValidator *validator.FrameValidator, dp *dispatcher.Dispatcher,
	scenarioEngine *scenario.Engine, logger logging.Logger,
) {
	// 1. 帧头校验
	if verr := frameValidator.ValidateHeader(header); verr != nil {
		logger.Warnf("[%s] frame validation failed: %s", conn.ID, verr.Message)
		return
	}

	// 2. 获取或创建会话（同一桩号拒绝重复连接）
	sess, ok := sessMgr.GetByPostNo(header.PostNo)
	if !ok {
		var createErr error
		sess, createErr = sessMgr.Create(header.PostNo, conn.ID)
		if createErr != nil {
			logger.Errorf("[%s] create session failed: %v", conn.ID, createErr)
			conn.Close()
			return
		}
		logger.Infof("[%s] new session: %s, postNo=%d", conn.ID, sess.ID, header.PostNo)
	}
	sess.UpdateActive()

	// 3. 解密数据域
	decryptFn := sess.GetDecryptFn()
	if proto.IsFixedKeyFuncCode(header.FuncCode) || sess.GetAuthState() == session.Authenticated {
		if header.EncryptFlag == 0x01 && decryptFn != nil {
			decrypted, err := decryptFn(data)
			if err != nil {
				logger.Warnf("[%s] decrypt func=0x%02X failed: %v", conn.ID, header.FuncCode, err)
				sess.Recorder.RecordRecv(header.FuncCode, recorder.StatusDecodeFail,
					fmt.Sprintf("% X", data), "", err.Error())
				return
			}
			data = decrypted
		}
	}

	// 4. 消息解码
	dir := directionOf(header)
	msg, ok := proto.Registry().Create(header.FuncCode, dir)
	msgStatus := recorder.StatusSuccess
	var decodeErr string
	if !ok {
		logger.Warnf("[%s] unregistered func=0x%02X dir=%v", conn.ID, header.FuncCode, dir)
		decodeErr = fmt.Sprintf("unregistered func=0x%02X", header.FuncCode)
		msgStatus = recorder.StatusDecodeFail
	} else if len(data) > 0 {
		if err := msg.Decode(data); err != nil {
			logger.Warnf("[%s] decode func=0x%02X failed: %v", conn.ID, header.FuncCode, err)
			decodeErr = err.Error()
			msgStatus = recorder.StatusDecodeFail
		}
	}

	// 5. 字段校验
	if msgStatus == recorder.StatusSuccess && msg != nil {
		if errs := frameValidator.ValidateMessage(msg); len(errs) > 0 {
			msgStatus = recorder.StatusInvalidField
			for _, e := range errs {
				logger.Warnf("[%s] validation: field=%s code=%s msg=%s", conn.ID, e.Field, e.Code, e.Message)
			}
		}
	}

	// 6. 记录消息到 recorder
	hexData := fmt.Sprintf("% X", data)
	jsonData := ""
	if msg != nil {
		jsonData = fmt.Sprintf("%+v", msg.ToJSONMap())
	}
	sess.Recorder.RecordRecv(header.FuncCode, msgStatus, hexData, jsonData, decodeErr)

	// 6.1 异步存档消息到数据库（不影响主流程）
	go func() {
		if msg != nil && msgStatus == recorder.StatusSuccess {
			_ = report.SaveMessageArchive(sess.ID, recorder.MessageRecord{
				Timestamp: time.Now(),
				FuncCode:  header.FuncCode,
				Direction: dir,
				Status:    msgStatus,
				HexData:   hexData,
				JSONData:  jsonData,
				ErrorMsg:  decodeErr,
			})
		}
	}()

	// 7. 日志
	if msg != nil {
		logger.Infof("[%s] recv func=0x%02X name=%s postNo=%d charger=%d status=%s",
			conn.ID, header.FuncCode, msg.Spec().Name, header.PostNo, header.Charger, msgStatus)
	}

	// 8. 分发
	replyFn := func(replyHeader types.MessageHeader, replyData []byte) error {
		encryptFn := sess.GetEncryptFn()
		replyHeader.PostNo = header.PostNo
		replyHeader.Charger = header.Charger
		replyHeader.Version = proto.Version()
		replyHeader.StartByte = proto.FrameConfig().StartByte

		replyHex := fmt.Sprintf("% X", replyData)
		sess.Recorder.RecordReply(replyHeader.FuncCode, recorder.StatusSuccess, replyHex, "", "")

		// 异步存档回复消息
		go func() {
			_ = report.SaveMessageArchive(sess.ID, recorder.MessageRecord{
				Timestamp: time.Now(),
				FuncCode:  replyHeader.FuncCode,
				Direction: types.DirectionReply,
				Status:    recorder.StatusSuccess,
				HexData:   replyHex,
				JSONData:  "",
				ErrorMsg:  "",
			})
		}()

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

	// 通知场景引擎
	if msg != nil && msgStatus == recorder.StatusSuccess {
		scenarioEngine.OnMessage(sess.ID, header.FuncCode, dir, msg)
	}
}

// directionOf 根据消息来源判断方向。
// 当前作为平台侧网关，接收的消息全部来自充电桩，因此固定返回 DirectionUpload。
// header 参数暂未使用，保留用于未来桩侧网关复用时根据上下文判断方向。
func directionOf(header types.MessageHeader) types.Direction {
	return types.DirectionUpload
}

// onDisconnect 连接断开回调：关闭记录器、保存报告到数据库、移除会话
func onDisconnect(conn *server.Connection, postNo uint32, sessMgr *session.SessionManager,
	proto types.Protocol, scenarioEngine *scenario.Engine, logger logging.Logger,
) {
	sess, ok := sessMgr.GetByPostNo(postNo)
	if !ok {
		return
	}

	// 停止场景
	scenarioEngine.RemoveScenario(sess.ID)

	// 关闭记录器
	if sess.Recorder != nil {
		sess.Recorder.Close()
		summary := sess.Recorder.Summary()
		logger.Infof("[sess:%s] session ended: postNo=%d duration=%s total=%d success=%d fail=%d rate=%.1f%% pass=%v",
			sess.ID, postNo, summary.Duration, summary.TotalMessages,
			summary.SuccessTotal, summary.FailTotal, summary.SuccessRate, summary.IsPass)

		// 保存报告到数据库
		if err := report.SaveReport(summary, proto.Name(), fmt.Sprintf("v0x%02X", proto.Version())); err != nil {
			logger.Errorf("[sess:%s] save report failed: %v", sess.ID, err)
		} else {
			logger.Infof("[sess:%s] report saved to database", sess.ID)
		}
	}

	// 移除会话
	sessMgr.Remove(sess.ID)
	logger.Infof("[sess:%s] session removed for postNo=%d", sess.ID, postNo)
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

// heartbeatCheckLoop 定期检查心跳超时的会话，断开连接并生成报告
func heartbeatCheckLoop(ctx context.Context, sessMgr *session.SessionManager,
	srv *server.Server, proto types.Protocol, scenarioEngine *scenario.Engine, logger logging.Logger,
) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expired := sessMgr.FindHeartbeatTimeout()
			for _, sess := range expired {
				logger.Warnf("[sess:%s] heartbeat timeout, postNo=%d, closing connection", sess.ID, sess.PostNo)

				// 关闭TCP连接
				if conn, ok := srv.FindConnectionByPostNo(sess.PostNo); ok {
					conn.Close()
				}

				// 触发断开回调（生成报告、保存数据库）
				onDisconnect(nil, sess.PostNo, sessMgr, proto, scenarioEngine, logger)
			}
		}
	}
}

// serveSwaggerUI 提供嵌入式 Swagger UI 页面，加载本地 openapi.yaml
func serveSwaggerUI(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>NTS-Gater API Documentation</title>
    <meta charset="utf-8"/>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" >
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
SwaggerUIBundle({
    url: "/docs/openapi.yaml",
    dom_id: '#swagger-ui',
    presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset
    ],
    layout: "BaseLayout"
})
</script>
</body>
</html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
