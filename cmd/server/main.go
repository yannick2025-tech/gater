// Package main provides the nts-gater server entry point.
package main

import (
	"context"
	"encoding/json"
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

	srv.OnMessage(func(conn *server.Connection, header types.MessageHeader, data []byte, rawFrame []byte) {
		onMessage(conn, header, data, rawFrame, proto, sessMgr, frameValidator, dp, scenarioEngine, logger)
	})
	srv.OnDisconnect(func(conn *server.Connection, postNo uint32) {
		onDisconnect(conn, postNo, sessMgr, proto, scenarioEngine, logger)
	})

	// HTTP服务器
	gin.SetMode(gin.ReleaseMode)
	router := api.NewRouter(sessMgr, scenarioEngine, logger)

	// Server 1: API 接口（内网端口，仅 /api/* 路由）
	apiEngine := gin.Default()
	router.SetupAPI(apiEngine)
	// API端口的404：返回带样式的错误页面（与Web端口风格一致）
	apiEngine.NoRoute(func(c *gin.Context) {
		c.Data(http.StatusNotFound, "text/html; charset=utf-8", []byte(api.NotFoundHTML))
	})
	apiSrv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HTTPPort),
		Handler: apiEngine,
	}

	// Server 2: Web 静态页面（对外端口，SPA + 静态资源，不含 API）
	webEngine := gin.Default()
	router.SetupWeb(webEngine)
	webSrv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.WebPort),
		Handler: webEngine,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		logger.Fatalf("failed to start TCP server: %v", err)
	}

	go func() {
		logger.Infof("API server listening on %s", apiSrv.Addr)
		if err := apiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("API server error: %v", err)
		}
	}()

	go func() {
		logger.Infof("Web server listening on %s", webSrv.Addr)
		if err := webSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Web server error: %v", err)
		}
	}()

	// 心跳超时检查：定期扫描超时会话并断开连接
	go heartbeatCheckLoop(ctx, sessMgr, srv, proto, scenarioEngine, logger)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Infof("received signal %v, shutting down...", sig)

	srv.Stop()
	apiSrv.Close()
	webSrv.Close()
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

func onMessage(conn *server.Connection, header types.MessageHeader, data []byte, rawFrame []byte,
	proto types.Protocol, sessMgr *session.SessionManager,
	frameValidator *validator.FrameValidator, dp *dispatcher.Dispatcher,
	scenarioEngine *scenario.Engine, logger logging.Logger,
) {
	// 1. 帧头校验
	if verr := frameValidator.ValidateHeader(header); verr != nil {
		logger.Warnf("[%s] frame validation failed: %s", conn.ID, verr.Message)
		return
	}

	// 1.1 充电桩编号校验：协议规定为8位数字（10000000~99999999）
	if postNoErr := frameValidator.ValidatePostNo(header.PostNo); postNoErr != nil {
		logger.Errorf("[%s] %s", conn.ID, postNoErr.Message)
		// 记录到会话统计（会话可能还未创建，尝试获取已有的）
		if sess, ok := sessMgr.GetByPostNo(header.PostNo); ok && sess.Recorder != nil {
			sess.Recorder.RecordRecv(header.FuncCode, recorder.StatusInvalidField,
				fmt.Sprintf("% X", data), "", postNoErr.Message)
		}
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
	if header.EncryptFlag == 0x01 {
		decrypted, err := decryptMessage(sess, proto, header.FuncCode, data)
		if err != nil {
			logger.Warnf("[%s] decrypt func=0x%02X failed: %v", sess.ID, header.FuncCode, err)
			sess.Recorder.RecordRecv(header.FuncCode, recorder.StatusDecodeFail,
				fmt.Sprintf("% X", data), "", err.Error())
			return
		}
		data = decrypted
	}

	// 4. 消息解码
	dir := directionOf(header)
	msg, ok := proto.Registry().Create(header.FuncCode, dir)
	msgStatus := recorder.StatusSuccess
	var decodeErr string
	if !ok {
		logger.Warnf("[%s] unregistered func=0x%02X dir=%v", sess.ID, header.FuncCode, dir)
		decodeErr = fmt.Sprintf("unregistered func=0x%02X", header.FuncCode)
		msgStatus = recorder.StatusDecodeFail
	} else if len(data) > 0 {
		if err := msg.Decode(data); err != nil {
			logger.Warnf("[%s] decode func=0x%02X failed: %v", sess.ID, header.FuncCode, err)
			decodeErr = err.Error()
			msgStatus = recorder.StatusDecodeFail
		}
	}

	// 5. 字段校验
	if msgStatus == recorder.StatusSuccess && msg != nil {
		if errs := frameValidator.ValidateMessage(msg); len(errs) > 0 {
			msgStatus = recorder.StatusInvalidField
			for _, e := range errs {
				logger.Warnf("[%s] validation: field=%s code=%s msg=%s", sess.ID, e.Field, e.Code, e.Message)
			}
		}
	}

	// 6. 记录消息到 recorder
	hexData := fmt.Sprintf("% X", data)
	jsonData := ""
	if msg != nil {
		jsonBytes, _ := json.Marshal(msg.ToJSONMap())
		jsonData = string(jsonBytes)
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

	// 7. 日志（完整帧hex + 完整消息json，带sessionID便于定位完整会话周期）
	frameHex := fmt.Sprintf("% X", rawFrame)
	// 构建完整消息JSON：帧头字段 + 消息体字段
	msgJSON := buildMessageJSON(header, msg)
	if msg != nil {
		logger.Infof("[%s] [Post→GATER] [0x%02X] %s postNo=%d charger=%d status=%s",
			sess.ID, header.FuncCode, msg.Spec().Name, header.PostNo, header.Charger, msgStatus)
		logger.Infof("[%s] [Post→GATER] [0x%02X] HEX: %s", sess.ID, header.FuncCode, frameHex)
		logger.Infof("[%s] [Post→GATER] [0x%02X] JSON: %s", sess.ID, header.FuncCode, msgJSON)
	} else {
		logger.Warnf("[%s] [Post→GATER] [0x%02X] decode failed postNo=%d status=%s",
			sess.ID, header.FuncCode, header.PostNo, msgStatus)
		logger.Infof("[%s] [Post→GATER] [0x%02X] HEX: %s", sess.ID, header.FuncCode, frameHex)
	}

	// 8. 分发
	replyFn := func(replyHeader types.MessageHeader, replyData []byte) error {
		encryptFn := sess.GetEncryptFn()
		replyHeader.PostNo = header.PostNo
		replyHeader.Charger = header.Charger
		replyHeader.Version = proto.Version()
		replyHeader.StartByte = proto.FrameConfig().StartByte

		// 编码完整帧（用于日志打印完整hex）
		replyFrame, encErr := conn.Encoder.Encode(replyHeader, replyData, encryptFn)
		if encErr != nil {
			logger.Errorf("[%s] [GATER→Post] [0x%02X] encode failed: %v", sess.ID, replyHeader.FuncCode, encErr)
			return encErr
		}

		// 更新replyHeader中的Checksum和DataLength为编码后的实际值（Encode是值拷贝不会更新原header）
		if len(replyFrame) >= 12 {
			replyHeader.Checksum = replyFrame[8]
			replyHeader.DataLength = uint16(replyFrame[10]) | uint16(replyFrame[11])<<8
		}

		replyHex := fmt.Sprintf("% X", replyData)
		replyFrameHex := fmt.Sprintf("% X", replyFrame)
		sess.Recorder.RecordReply(replyHeader.FuncCode, recorder.StatusSuccess, replyHex, "", "")

		// 日志：发送的回复消息，带sessionID，打印完整帧hex + 完整消息json
		// 解码回复消息体以构建JSON
		var replyMsg types.Message
		if replyRegMsg, ok := proto.Registry().Create(replyHeader.FuncCode, types.DirectionReply); ok && len(replyData) > 0 {
			_ = replyRegMsg.Decode(replyData)
			replyMsg = replyRegMsg
		}
		replyJSON := buildMessageJSON(replyHeader, replyMsg)
		logger.Infof("[%s] [GATER→Post] [0x%02X] postNo=%d charger=%d dataLen=%d",
			sess.ID, replyHeader.FuncCode, replyHeader.PostNo, replyHeader.Charger, len(replyData))
		logger.Infof("[%s] [GATER→Post] [0x%02X] HEX: %s", sess.ID, replyHeader.FuncCode, replyFrameHex)
		logger.Infof("[%s] [GATER→Post] [0x%02X] JSON: %s", sess.ID, replyHeader.FuncCode, replyJSON)

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

		return conn.SendFrame(replyFrame)
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
		SendDownload: func(dlMsg types.Message) error {
			// 主动下发消息到充电桩（如0x21密钥更新）
			dlData, encErr := dlMsg.Encode()
			if encErr != nil {
				return fmt.Errorf("encode download message failed: %w", encErr)
			}

			spec := dlMsg.Spec()
			dlHeader := types.MessageHeader{
				StartByte:   proto.FrameConfig().StartByte,
				Version:     proto.Version(),
				FuncCode:    spec.FuncCode,
				PostNo:      header.PostNo,
				Charger:     header.Charger,
				EncryptFlag: 0x00,
			}
			if spec.Encrypt {
				dlHeader.EncryptFlag = 0x01
			}

			// 加密：0x0A/0x0B/0x21始终用固定密钥，其他用当前会话密钥
			var encryptFn func([]byte) ([]byte, error)
			if proto.IsFixedKeyFuncCode(spec.FuncCode) {
				encryptFn = sess.FixedCipher.Encrypt
			} else {
				encryptFn = sess.GetEncryptFn()
			}
			dlFrame, encErr := conn.Encoder.Encode(dlHeader, dlData, encryptFn)
			if encErr != nil {
				return fmt.Errorf("encode download frame failed: %w", encErr)
			}

			// 更新header中的Checksum和DataLength
			if len(dlFrame) >= 12 {
				dlHeader.Checksum = dlFrame[8]
				dlHeader.DataLength = uint16(dlFrame[10]) | uint16(dlFrame[11])<<8
			}

			// 日志
			dlHex := fmt.Sprintf("% X", dlData)
			dlFrameHex := fmt.Sprintf("% X", dlFrame)
			sess.Recorder.RecordReply(dlHeader.FuncCode, recorder.StatusSuccess, dlHex, "", "")

			var dlReplyMsg types.Message
			if dlRegMsg, ok := proto.Registry().Create(dlHeader.FuncCode, types.DirectionDownload); ok && len(dlData) > 0 {
				_ = dlRegMsg.Decode(dlData)
				dlReplyMsg = dlRegMsg
			}
			dlJSON := buildMessageJSON(dlHeader, dlReplyMsg)
			logger.Infof("[%s] [GATER→Post] [0x%02X] postNo=%d charger=%d dataLen=%d",
				sess.ID, dlHeader.FuncCode, dlHeader.PostNo, dlHeader.Charger, len(dlData))
			logger.Infof("[%s] [GATER→Post] [0x%02X] HEX: %s", sess.ID, dlHeader.FuncCode, dlFrameHex)
			logger.Infof("[%s] [GATER→Post] [0x%02X] JSON: %s", sess.ID, dlHeader.FuncCode, dlJSON)

			return conn.SendFrame(dlFrame)
		},
	}

	if dp.HasHandler(header.FuncCode, dir) {
		if err := dp.Dispatch(ctx); err != nil {
			logger.Errorf("[%s] handle func=0x%02X error: %v", sess.ID, header.FuncCode, err)
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

// buildMessageJSON 构建完整消息JSON（帧头字段 + 消息体字段）
// 输出格式：{"cmd":6,"postNo":96048851,"charger":1,"encryptFlag":1,"checkSum":103,"bodyT":{...}}
func buildMessageJSON(header types.MessageHeader, msg types.Message) string {
	m := map[string]interface{}{
		"cmd":         header.FuncCode,
		"postNo":      header.PostNo,
		"charger":     header.Charger,
		"encryptFlag": header.EncryptFlag,
		"checkSum":    header.Checksum,
	}
	if msg != nil {
		m["bodyT"] = msg.ToJSONMap()
	}
	jsonBytes, _ := json.Marshal(m)
	return string(jsonBytes)
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
		authState := sess.GetAuthState().String()
		if err := report.SaveReport(summary, proto.Name(), fmt.Sprintf("v0x%02X", proto.Version()), authState); err != nil {
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
	// 使用 gwc-logging 的 InitFromConfig 加载独立日志配置文件
	// 配置文件：configs/logging.yaml（支持 target/filename/format/flush_interval 等完整字段）
	return logging.InitFromConfig("configs/logging.yaml")
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

// decryptMessage 解密充电桩消息
// 规则：
//   - 0x0A/0x0B 始终用固定密钥解密（认证阶段尚未交换密钥）
//   - 0x21回复和其他消息：gater在下发0x21时已切换到新密钥，桩回复时也已用新密钥加密
func decryptMessage(sess *session.Session, proto types.Protocol, funcCode byte, data []byte) ([]byte, error) {
	// 0x0A/0x0B 始终用固定密钥
	if funcCode == types.FuncAuthRandom || funcCode == types.FuncAuthEncrypted {
		return sess.FixedCipher.Decrypt(data)
	}

	// 0x21回复和其他消息：使用当前会话密钥
	// gater在下发0x21时已通过SetSessionKey切换到新密钥
	decryptFn := sess.GetDecryptFn()
	return decryptFn(data)
}

