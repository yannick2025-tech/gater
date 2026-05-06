// Package api provides HTTP API handlers for the web frontend.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yannick2025-tech/nts-gater/internal/model"
	"github.com/yannick2025-tech/nts-gater/internal/report"
	"github.com/yannick2025-tech/nts-gater/internal/recorder"
	"github.com/yannick2025-tech/nts-gater/internal/scenario"
	"github.com/yannick2025-tech/nts-gater/internal/server"
	"github.com/yannick2025-tech/nts-gater/internal/session"
	logging "github.com/yannick2025-tech/gwc-logging"
)

// DeviceConnectionRegistry 维护Web端发起的设备连接状态。
// 当用户在前端点击"连接设备"时注册，点击"断开连接"时注销。
// 所有需要校验设备连接状态的API（startTest、configDownload等）
// 统一检查此注册表，确保前后端状态一致。
type DeviceConnectionRegistry struct {
	mu         sync.RWMutex
	devices    map[uint32]*DeviceInfo // postNo -> 设备信息
}

// DeviceInfo Web端连接的设备信息
type DeviceInfo struct {
	GunNumber      string
	ProtocolName   string
	ProtocolVersion string
	ConnectedAt    time.Time
	SessionID      string // 如果有真实TCP会话则关联
}

func NewDeviceConnectionRegistry() *DeviceConnectionRegistry {
	return &DeviceConnectionRegistry{
		devices: make(map[uint32]*DeviceInfo),
	}
}

// Register 注册设备为已连接（Web端连接操作）
func (r *DeviceConnectionRegistry) Register(postNo uint32, gunNumber string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices[postNo] = &DeviceInfo{
		GunNumber:       gunNumber,
		ProtocolName:    "XX标准协议",
		ProtocolVersion: "v1.6.0",
		ConnectedAt:     time.Now(),
	}
}

// Unregister 注销设备连接（Web端断开操作）
func (r *DeviceConnectionRegistry) Unregister(postNo uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.devices, postNo)
}

// IsConnected 检查设备是否已连接（优先查Web注册表，再查TCP会话）
func (r *DeviceConnectionRegistry) IsConnected(postNo uint32, sessMgr *session.SessionManager) bool {
	// 1. 先查Web注册表（前端手动连接的）
	r.mu.RLock()
	if _, ok := r.devices[postNo]; ok {
		r.mu.RUnlock()
		return true
	}
	r.mu.RUnlock()

	// 2. 再查TCP会话管理器（充电桩主动TCP连接的）
	if sess, ok := sessMgr.GetByPostNo(postNo); ok {
		return sess.GetAuthState() == session.Authenticated
	}
	return false
}

// GetInfo 获取已连接设备的信息
func (r *DeviceConnectionRegistry) GetInfo(postNo uint32) (*DeviceInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.devices[postNo]
	return info, ok
}

// GetOrMergeInfo 获取设备信息：优先Web注册表，否则从TCP会话补充
func (r *DeviceConnectionRegistry) GetOrMergeInfo(postNo uint32, sessMgr *session.SessionManager) *DeviceInfo {
	r.mu.RLock()
	info, ok := r.devices[postNo]
	r.mu.RUnlock()
	if ok {
		return info
	}

	// 回退到TCP会话
	if sess, ok := sessMgr.GetByPostNo(postNo); ok {
		sessionID := ""
		if sess != nil {
			sessionID = sess.ID
			_ = sess.GetAuthState()
		}
		return &DeviceInfo{
			GunNumber:       strconv.FormatUint(uint64(postNo), 10),
			ProtocolName:    "XX标准协议",
			ProtocolVersion: "v1.6.0",
			SessionID:       sessionID,
		}
	}
	return nil
}

// ==================== HTTP Router ====================

// Router HTTP路由
type Router struct {
	sessMgr        *session.SessionManager
	scenarioEngine *scenario.Engine
	connRegistry   *DeviceConnectionRegistry
	srv            *server.Server
	logger         logging.Logger
}

// NewRouter 创建路由
func NewRouter(sessMgr *session.SessionManager, scenarioEngine *scenario.Engine, srv *server.Server, logger logging.Logger) *Router {
	return &Router{
		sessMgr:        sessMgr,
		scenarioEngine: scenarioEngine,
		connRegistry:   NewDeviceConnectionRegistry(),
		srv:            srv,
		logger:         logger,
	}
}

// Setup 设置路由（兼容旧入口，内部委托 SetupAPI）
func (r *Router) Setup(engine *gin.Engine) {
	r.SetupAPI(engine)
}

// SetupAPI 注册所有 /api/* 路由（绑定到 API 端口，如 9090）
func (r *Router) SetupAPI(engine *gin.Engine) {
	api := engine.Group("/api")

	// Web↔Gater 接口日志中间件：记录所有请求参数和响应，方便排查问题
	api.Use(r.apiLoggingMiddleware())

	{
		// 会话管理
		api.GET("/sessions", r.getSessions)

		// 设备管理
		api.GET("/device/status", r.getDeviceStatus)
		api.POST("/device/connect", r.toggleConnection)
		api.POST("/device/disconnect", r.disconnectDevice)

		// 测试管理
		api.POST("/test/start", r.startTest)
		api.POST("/test/stop", r.stopTest)
		api.GET("/test/charging-info/:sessionId", r.getChargingInfo)
		api.GET("/test/status/:sessionId", r.getTestStatus)
		api.GET("/test/results", r.getTestResults)
		api.GET("/test/detail/:sessionId", r.getTestDetail)
		api.GET("/test/messages/:sessionId", r.getMessages)
		api.POST("/test/decode", r.decodeMessage)
		api.POST("/test/export", r.exportReport)
		api.GET("/test/download", r.downloadPDF)
		api.POST("/test/config", r.configDownload)

		// 新增接口：用例 + 校验结果 + HTML 报告
		api.GET("/test/cases/:sessionId", r.getTestCases)
		api.GET("/test/validations/:sessionId", r.getValidationResults)
		api.GET("/reports/:sessionId/html", r.getHTMLReport)
	}
}

// serveSwaggerUI 提供 Swagger UI 页面，加载本地 openapi.yaml
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
        SwaggerUIBundle.SwaggerStandalonePreset
    ],
    layout: "BaseLayout"
})
</script>
</body>
</html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// SetupWeb 注册静态文件服务 + SPA fallback（绑定到 Web 端口，如 8080）
func (r *Router) SetupWeb(engine *gin.Engine) {
	// 前端静态文件
	engine.Static("/assets", "./web/dist/assets")

	// API 文档：YAML文件 + Swagger UI
	engine.Static("/docs", "./docs/api")
	engine.GET("/swagger", serveSwaggerUI)

	// 通用的 404 页面：所有未匹配路由都返回美观的错误页
	engine.NoRoute(func(c *gin.Context) {
		c.Data(http.StatusNotFound, "text/html; charset=utf-8", []byte(NotFoundHTML))
	})
}

// NotFoundHTML 内嵌的通用404页面
const NotFoundHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>404 - 页面未找到 | NTS Gater</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{min-height:100vh;display:flex;align-items:center;justify-content:center;background:#f5f7fa;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;color:#333}
.container{text-align:center;padding:40px;max-width:480px}
.status-code{font-size:72px;font-weight:800;color:#c0c4cc;margin-bottom:0;line-height:1}
.status-text{font-size:18px;color:#909399;margin-bottom:24px}
.desc{font-size:14px;color:#999;margin-bottom:32px;line-height:1.6}
.btn{display:inline-block;padding:10px 24px;font-size:14px;color:#fff;text-decoration:none;border-radius:6px;transition:background .2s}
.btn-primary{background:#409eff}
.btn-primary:hover{background:#337ecc}
.divider{width:40px;height:2px;background:#e4e7ed;margin:0 auto 16px;border-radius:1px}
</style>
</head>
<body>
<div class="container">
<div class="status-code">404</div>
<div class="status-text">页面未找到</div>
<div class="desc">您访问的资源不存在或已被移除</div>
<div class="divider"></div>
<a class="btn btn-primary" href="/">返回首页</a>
</div>
<script>
// 自动重定向：如果是前端路由路径（非静态资源），尝试前端SPA处理
if (!window.location.pathname.startsWith('/assets') &&
    !window.location.pathname.startsWith('/docs')) {
  // 静默跳过，让前端路由处理
}
</script>
</body>
</html>`

// ApiNotFoundHTML API端口的404页面（提示文字区别于Web端口）
const ApiNotFoundHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>404 - 接口未找到 | NTS Gater</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{min-height:100vh;display:flex;align-items:center;justify-content:center;background:#f5f7fa;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;color:#333}
.container{text-align:center;padding:40px;max-width:480px}
.status-code{font-size:72px;font-weight:800;color:#c0c4cc;margin-bottom:0;line-height:1}
.status-text{font-size:18px;color:#909399;margin-bottom:24px}
.desc{font-size:14px;color:#999;margin-bottom:32px;line-height:1.6}
.btn{display:inline-block;padding:10px 24px;font-size:14px;color:#fff;text-decoration:none;border-radius:6px;transition:background .2s}
.btn-primary{background:#409eff}
.btn-primary:hover{background:#337ecc}
.divider{width:40px;height:2px;background:#e4e7ed;margin:0 auto 16px;border-radius:1px}
</style>
</head>
<body>
<div class="container">
<div class="status-code">404</div>
<div class="status-text">接口未找到</div>
<div class="desc">您请求的 API 资源不存在或已被移除</div>
<div class="divider"></div>
<a class="btn btn-primary" href="/">返回首页</a>
</div>
</body>
</html>`

// ==================== 设备管理接口 ====================

// getDeviceStatus 获取设备状态
// GET /api/device/status?gunNumber=5023001201
func (r *Router) getDeviceStatus(c *gin.Context) {
	gunNumber := c.Query("gunNumber")
	if gunNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "gunNumber is required"})
		return
	}

	postNo, err := strconv.ParseUint(gunNumber, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid gunNumber"})
		return
	}

	pn := uint32(postNo)

	// 统一使用 connRegistry 判断在线状态
	isOnline := r.connRegistry.IsConnected(pn, r.sessMgr)
	info := r.connRegistry.GetOrMergeInfo(pn, r.sessMgr)

	respData := gin.H{
		"gunNumber":       gunNumber,
		"protocolName":    info.ProtocolName,
		"protocolVersion": info.ProtocolVersion,
		"isOnline":        isOnline,
	}

	if info != nil && info.SessionID != "" {
		respData["sessionId"] = info.SessionID
	} else if info != nil {
		respData["sessionId"] = "" // Web连接无真实session
	}

	// 补充认证状态（来自TCP会话）
	if sess, ok := r.sessMgr.GetByPostNo(pn); ok {
		respData["authState"] = sess.GetAuthState().String()
		if sess.ID != "" {
			respData["sessionId"] = sess.ID
		}
	} else {
		respData["authState"] = "none"
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": respData})
}

// getSessions 获取所有会话列表（DB 为唯一数据源，内存仅判断在线状态）
// GET /api/sessions
func (r *Router) getSessions(c *gin.Context) {
	// 1. 从 DB 的 sessions 表读取所有会话
	dbSessions, err := report.GetAllSessionsFromDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "query sessions failed"})
		return
	}

	// 2. 收集内存中活跃 session 的在线状态（用于实时更新）
	activeSessionIDs := make(map[string]bool)
	sessions := r.sessMgr.GetAllSessions()
	for _, sess := range sessions {
		activeSessionIDs[sess.ID] = true
	}

	// 3. 组装响应
	list := make([]gin.H, 0, len(dbSessions))
	for _, s := range dbSessions {
		// 如果内存中有活跃 session，以内存中的在线状态为准
		isOnline := s.IsOnline
		if activeSessionIDs[s.ID] {
			if sess, ok := r.sessMgr.Get(s.ID); ok {
				isOnline = sess.IsConnected()
			}
		}

		// 补充认证状态（内存中可能更新了）
		authState := s.AuthState
		if sess, ok := r.sessMgr.Get(s.ID); ok {
			authState = sess.GetAuthState().String()
		}

		item := gin.H{
			"sessionId":       s.ID,
			"postNo":          s.PostNo,
			"gunNumber":       fmt.Sprintf("%d", s.PostNo),
			"authState":       authState,
			"isOnline":        isOnline,
			"protocolName":    s.ProtocolName,
			"protocolVersion": s.ProtocolVer,
			"connectedAt":     s.CreatedAt.Format("2006-01-02 15:04:05"),
			"lastActive":      s.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		// 补充测试状态（如正在运行场景）
		if sc, ok := r.scenarioEngine.GetScenario(s.ID); ok {
			if result := sc.Result(); result != nil {
				item["testStatus"] = string(result.State)
				item["testCase"] = result.ScenarioName
				item["testProgress"] = result.Progress
			}
		}

		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"total": len(list),
			"list":  list,
		},
	})
}

// toggleConnection 连接/断开设备
// POST /api/device/connect
// Body: {"gunNumber": "5023001201", "action": "connect"|"disconnect"}
func (r *Router) toggleConnection(c *gin.Context) {
	var req struct {
		GunNumber string `json:"gunNumber"`
		Action    string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	postNo, err := strconv.ParseUint(req.GunNumber, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid gunNumber"})
		return
	}

	pn := uint32(postNo)

	switch req.Action {
	case "connect":
		// 在Web注册表中注册为已连接
		r.connRegistry.Register(pn, req.GunNumber)
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"isOnline":    true,
				"gunNumber":   req.GunNumber,
				"protocolName":"XX标准协议",
				"protocolVersion": "v1.6.0",
			},
		})

	case "disconnect":
		r.handleDisconnect(c, pn)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "action must be connect or disconnect"})
	}
}

// disconnectDevice 断开设备连接
// POST /api/device/disconnect
// Body: {"gunNumber": "5023001201"}
func (r *Router) disconnectDevice(c *gin.Context) {
	var req struct {
		GunNumber string `json:"gunNumber"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	postNo, err := strconv.ParseUint(req.GunNumber, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid gunNumber"})
		return
	}

	r.handleDisconnect(c, uint32(postNo))
}

// handleDisconnect 统一断开逻辑：立即返回 + 异步清理
func (r *Router) handleDisconnect(c *gin.Context, postNo uint32) {
	// 1. 从Web注册表移除
	r.connRegistry.Unregister(postNo)

	// 2. 标记会话为断开状态（同步，确保前端立即看到离线）
	if sess, ok := r.sessMgr.GetByPostNo(postNo); ok {
		sess.SetConnected(false)
	}

	// 3. 立即返回成功（不等待后续清理操作）
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"isOnline": false}})

	// 4. 异步执行耗时清理操作
	// 关键顺序：先关TCP → 再移除session（避免 Remove 触发 onDisconnect 重复处理）
	go func() {
		if sess, ok := r.sessMgr.GetByPostNo(postNo); ok {
			r.logger.Infof("[disconnect] start cleanup for postNo=%d sessionId=%s", postNo, sess.ID)

			// 4.1 获取场景名称，再停止场景引擎
			testCaseName := ""
			if sc, ok := r.scenarioEngine.GetScenario(sess.ID); ok {
				testCaseName = sc.Name()
			}
			r.scenarioEngine.RemoveScenario(sess.ID)

			// 4.2 关闭记录器
			if sess.Recorder != nil {
				sess.Recorder.Close()
			}

			// 4.3 构建会话摘要并保存报告
			var summary *recorder.SessionSummary
			if sess.Recorder != nil {
				summary = sess.Recorder.Summary()
			} else {
				now := time.Now()
				summary = &recorder.SessionSummary{
					SessionID:     sess.ID,
					PostNo:        sess.PostNo,
					StartTime:     sess.CreatedAt,
					EndTime:       now,
					Duration:      now.Sub(sess.CreatedAt),
				}
			}

			protocolName := "XX标准协议"
			if testCaseName != "" {
				switch testCaseName {
			case "basic_charging":
				protocolName = "XX标准协议-业务场景测试"
				case "sftp_upgrade":
					protocolName = "XX标准协议-SFTP升级测试"
				case "config_download":
					protocolName = "XX标准协议-配置下发测试"
				}
			}
			err := report.SaveReport(summary, protocolName, "v1.6.0", sess.GetAuthState().String(), sess.Recorder)
			if err != nil {
				r.logger.Errorf("[disconnect] save report error for session %s: %v", sess.ID, err)
			} else {
				r.logger.Infof("[disconnect] report saved for session %s (postNo=%d)", sess.ID, postNo)
			}

			// ★ 4.4 先关闭TCP连接（在移除会话之前！）
			// 必须在 sessMgr.Remove 之前关闭TCP，因为：
			//   a) Remove 会触发 session 内部清理（可能影响连接引用）
			//   b) TCP关闭后 handleConnection 的 Read() 才会退出，触发 defer 中的 onDisconnect
			conn, connFound := r.srv.FindConnectionByPostNo(postNo)
			if connFound && conn != nil {
				r.logger.Infof("[disconnect] closing TCP connection for postNo=%d (connId=%s addr=%s)",
					postNo, conn.ID, conn.RemoteAddr)
				if closeErr := conn.Close(); closeErr != nil {
					r.logger.Errorf("[disconnect] conn.Close() error for postNo=%d: %v", postNo, closeErr)
				} else {
					r.logger.Infof("[disconnect] TCP connection closed for postNo=%d", postNo)
				}
			} else {
				r.logger.Warnf("[disconnect] TCP connection not found for postNo=%d (may already be closed)", postNo)
			}

			// 4.5 最后移除会话（必须在TCP关闭之后）
			r.sessMgr.Remove(sess.ID)

			r.logger.Infof("[disconnect] cleanup completed for postNo=%d sessionId=%s", postNo, sess.ID)
		} else {
			r.logger.Warnf("[disconnect] session not found for postNo=%d (already removed?)", postNo)
		}
	}()
}

// ==================== 测试接口 ====================

// startTest 开始测试（基于已存在的TCP会话）
// POST /api/test/start
// Body: {"sessionId":"A1B2C3D4E5F6G7H8","testCase":"basic_charming","params":{...}}
func (r *Router) startTest(c *gin.Context) {
	var req struct {
		SessionID string                 `json:"sessionId"` // 必填：从会话列表选择的真实sessionID
		TestCase  string                 `json:"testCase"`
		Params    map[string]interface{} `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if req.SessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "sessionId is required"})
		return
	}

	// 获取真实session
	sess, ok := r.sessMgr.Get(req.SessionID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "session not found or already ended"})
		return
	}

	// 检查session是否已认证
	if sess.GetAuthState() != session.Authenticated {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "session not authenticated yet"})
		return
	}

	sc, scenarioID, err := r.scenarioEngine.StartScenario(sess.ID, req.TestCase, req.Params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 立即写入 running 占位记录到数据库（scenarioID 由 StartScenario 内部生成并返回）
	if err := report.CreateRunningReport(sess.ID, sess.PostNo, req.TestCase, scenarioID); err != nil {
		fmt.Printf("[startTest] create running report warning: %v\n", err)
	}

	result := sc.Result()
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId":   sess.ID,
			"scenarioId":  scenarioID,
			"status":      string(result.State),
			"testCase":    req.TestCase,
			"progress":    result.Progress,
			"stepName":    result.StepName,
		},
	})
}

// stopTest 停止充电（发送0x08消息到充电桩）
// POST /api/test/stop
// Body: {"sessionId":"A1B2C3D4E5F6G7H8"}
func (r *Router) stopTest(c *gin.Context) {
	var req struct {
		SessionID string `json:"sessionId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if req.SessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "sessionId is required"})
		return
	}

	// 检查session是否存在且在线
	sess, ok := r.sessMgr.Get(req.SessionID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "session not found"})
		return
	}
	if !sess.IsConnected() {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "session is offline"})
		return
	}

	// 记录平台充电结束时间（毫秒精度）
	now := time.Now()
	sess.SetPlatformStopTime(now)

	// 通过场景引擎发送0x08停止充电消息
	if err := r.scenarioEngine.SendStopCharge(req.SessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// ★ 停止场景引擎中的场景（将内存状态从 running → completed，否则 getSessions 仍返回 testStatus=running）
	r.scenarioEngine.StopScenario(req.SessionID)

	// ★ 同步保存当前统计快照到 DB（确保停止充电后立即可查看详情）
	// 注意：这里同步执行 SaveReport，保证 API 返回时数据已入库，
	// 用户点击"查看详情"时能同时看到统计数据和报文存档
	if sess, ok := r.sessMgr.Get(req.SessionID); ok && sess.Recorder != nil {
		summary := sess.Recorder.Summary()
		authState := sess.GetAuthState().String()
		if err := report.SaveReport(summary, "XX标准协议", "v1.6.0", authState, sess.Recorder); err != nil {
			r.logger.Errorf("[stopTest] save report error: %v", err)
		} else {
			r.logger.Infof("[stopTest] stats snapshot saved for session %s", req.SessionID)
		}

		// 标记该会话下最新的 running 场景为 completed
		reports, _ := report.GetTestReportsBySessionID(req.SessionID)
		for i := len(reports) - 1; i >= 0; i-- {
			if reports[i].Status == "running" {
				report.UpdateScenarioStatus(reports[i].ScenarioID, "completed")
				break // 只更新最新的一条 running 记录
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId":        req.SessionID,
			"platformStopTime": now.Format("2006-01-02 15:04:05"),
		},
	})
}

// getChargingInfo 查询充电信息（前端定时轮询）
// DB 优先：校验结果从 validation_results 表读取，充电状态从内存补充
// GET /api/test/charging-info/:sessionId
func (r *Router) getChargingInfo(c *gin.Context) {
	sessionID := c.Param("sessionId")

	sess, ok := r.sessMgr.Get(sessionID)

	result := gin.H{
		"sessionId": sessionID,
		"isOnline":  false,
	}

	// 在线状态从内存判断
	if ok {
		result["isOnline"] = sess.IsConnected()
	} else {
		// 内存中无，从 DB sessions 表判断
		dbSess, err := report.GetAllSessionsFromDB()
		if err == nil {
			for _, s := range dbSess {
				if s.ID == sessionID {
					result["isOnline"] = s.IsOnline
					break
				}
			}
		}
	}

	// 时段费用配置（优先内存，再 DB）
	if ok {
		prices := sess.GetPrices()
		priceList := make([]gin.H, len(prices))
		for i, p := range prices {
			priceList[i] = gin.H{
				"startTime":     p.StartTime,
				"endTime":       p.EndTime,
				"electricityFee": p.ElectricityFee,
				"serviceFee":    p.ServiceFee,
			}
		}
		result["prices"] = priceList
	}

	// 充电信息（优先内存，再 DB）
	if ok {
		cs := sess.GetChargingState()
		if cs != nil {
			chargingInfo := gin.H{
				"platformStartTime":  formatTime(cs.PlatformStartTime),
				"firstDataTime":      formatTime(cs.FirstDataTime),
				"lastDataTime":       formatTime(cs.LastDataTime),
				"platformStopTime":   formatTime(cs.PlatformStopTime),
				"chargerStartTime":   cs.ChargerStartTime,
				"chargerStopTime":    cs.ChargerStopTime,
				"chargingOrderNo":    cs.ChargingOrderNo,
				"currentElec":        cs.CurrentElec,
				"currentSOC":         cs.CurrentSOC,
				"stopSOC":            cs.StopSOC,
				"chargingDataCount":  cs.ChargingDataCount,
				"isChargingStopped":  cs.IsChargingStopped,
			}
			if !cs.FirstDataTime.IsZero() {
				endT := time.Now()
				if cs.IsChargingStopped && !cs.LastDataTime.IsZero() {
					endT = cs.LastDataTime
				}
				duration := endT.Sub(cs.FirstDataTime)
				chargingInfo["chargingDurationSec"] = int(duration.Seconds())
			}
			result["chargingInfo"] = chargingInfo

			// 校验结果摘要（内存）
			passCount := 0
			failCount := 0
			for _, v := range cs.ValidationResults {
				if v.Passed {
					passCount++
				} else {
					failCount++
				}
			}
			result["validationSummary"] = gin.H{
				"total": len(cs.ValidationResults),
				"pass":  passCount,
				"fail":  failCount,
			}
		}
	} else {
		// 从 DB 读校验结果
		validations, err := report.GetValidationResultsBySessionID(sessionID)
		if err == nil && len(validations) > 0 {
			passCount := 0
			failCount := 0
			for _, v := range validations {
				if v.Passed {
					passCount++
				} else {
					failCount++
				}
			}
			result["validationSummary"] = gin.H{
				"total": len(validations),
				"pass":  passCount,
				"fail":  failCount,
			}
			result["validations"] = validations
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": result})
}

// formatTime 格式化时间（无毫秒）
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// formatTimeMs 格式化时间（带毫秒）
func formatTimeMs(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05.000")
}

// getTestStatus 查询测试进度（DB 优先，内存补充在线场景信息）
// GET /api/test/status/:sessionId
func (r *Router) getTestStatus(c *gin.Context) {
	sessionID := c.Param("sessionId")

	// 1. 先尝试从内存获取活跃场景
	if sc, ok := r.scenarioEngine.GetScenario(sessionID); ok {
		result := sc.Result()
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"sessionId": sessionID,
				"status":    string(result.State),
				"progress":  result.Progress,
				"stepName":  result.StepName,
				"testCase":  result.ScenarioName,
			},
		})
		return
	}

	// 2. 从 DB 查询
	testReport, err := report.GetTestReportBySessionID(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "report not found"})
		return
	}

	status := "completed"
	if !testReport.IsPass && testReport.FailTotal > 0 {
		status = "failed"
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId": sessionID,
			"status":    status,
			"progress":  100,
			"stepName":  "已完成",
			"testCase":  testReport.ScenarioName,
		},
	})
}

// getTestResults 获取测试结果列表
// GET /api/test/results?page=1&pageSize=10&startTime=&endTime=&sessionId=
func (r *Router) getTestResults(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	var startTime *time.Time
	var endTime *time.Time
	if st := c.Query("startTime"); st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			startTime = &t
		}
	}
	if et := c.Query("endTime"); et != "" {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			endTime = &t
		}
	}

	sessionID := c.Query("sessionId")

	reports, total, err := report.GetTestReports(page, pageSize, startTime, endTime, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
			"list":     reports,
		},
	})
}

// getTestDetail 获取测试详情（始终从 DB 读取）
func (r *Router) getTestDetail(c *gin.Context) {
	sessionID := c.Param("sessionId")

	// 从数据库查询
	testReport, err := report.GetTestReportBySessionID(sessionID)
	stats, _ := report.GetFuncCodeStatsBySessionID(sessionID)
	cases, _ := report.GetTestCasesBySessionID(sessionID)

	startTime := time.Time{}
	var endTime *time.Time
	status := "running"
	if err == nil && testReport != nil {
		startTime = testReport.StartTime
		endTime = testReport.EndTime
		if testReport.IsPass {
			status = "pass"
		} else if testReport.Status == "completed" || (endTime != nil && !endTime.IsZero()) {
			status = "fail"
		}
	}

	// 补充在线状态（内存中判断）
	isOnline := false
	if sess, ok := r.sessMgr.Get(sessionID); ok {
		isOnline = sess.IsConnected()
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId":  sessionID,
			"startTime":  startTime,
			"endTime":    func() time.Time { if endTime != nil { return *endTime }; return time.Time{} }(),
			"status":     status,
			"statistics": stats,
			"cases":      cases,
			"isLive":     isOnline,
		},
	})
}

// getMessages 获取会话的所有报文存档（始终从 DB 读取）
// GET /api/test/messages/:sessionId?funcCode=&status=
func (r *Router) getMessages(c *gin.Context) {
	sessionID := c.Param("sessionId")
	funcCode := c.Query("funcCode")
	status := c.Query("status")

	// 始终从数据库查询
	archives, err := report.GetMessageArchivesBySessionID(sessionID, funcCode, status)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"list": []interface{}{}}})
		return
	}

	if archives == nil {
		archives = []model.MessageArchive{}
	}

	list := make([]gin.H, len(archives))
	for i, a := range archives {
		list[i] = gin.H{
			"id":        a.ID,
			"sessionId": sessionID,
			"caseId":    a.CaseID,
			"funcCode":  a.FuncCode,
			"direction": a.Direction,
			"status":    a.Status,
			"hexData":   a.HexData,
			"jsonData":  a.JSONData,
			"errorMsg":  a.ErrorMsg,
			"timestamp": a.Timestamp.Format("2006-01-02 15:04:05"),
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"list": list}})
}

// decodeMessage 解码16进制报文
// POST /api/test/decode
// Body: {"hex": "320601..."}
func (r *Router) decodeMessage(c *gin.Context) {
	var req struct {
		Hex string `json:"hex"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"json": "{}",
		},
	})
}

// exportReport 导出测试报告（生成 HTML 并打包为 ZIP）
// POST /api/test/export
// Body: {"sessionId": "sess-1"}
func (r *Router) exportReport(c *gin.Context) {
	var req struct {
		SessionID string `json:"sessionId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	zipPath, err := report.GenerateHTML(req.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "generate report failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId": req.SessionID,
			"zipUrl":    "/api/reports/" + req.SessionID + "/html",
			"zipPath":   zipPath,
		},
	})
}

// downloadPDF 下载PDF/HTML文件
// GET /api/test/download?path=reports/report_xxx.html
func (r *Router) downloadPDF(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "path is required"})
		return
	}

	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	c.File(path)
}

// getTestCases 获取某 session 下所有用例列表
// GET /api/test/cases/:sessionId
func (r *Router) getTestCases(c *gin.Context) {
	sessionID := c.Param("sessionId")

	cases, err := report.GetTestCasesBySessionID(sessionID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"list": []interface{}{}}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId": sessionID,
			"list":      cases,
		},
	})
}

// getValidationResults 获取某 session 下所有校验结果
// GET /api/test/validations/:sessionId
func (r *Router) getValidationResults(c *gin.Context) {
	sessionID := c.Param("sessionId")

	results, err := report.GetValidationResultsBySessionID(sessionID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"list": []interface{}{}}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId": sessionID,
			"list":      results,
		},
	})
}

// getHTMLReport 生成并下载 HTML 报告（ZIP 格式）
// GET /api/reports/:sessionId/html
func (r *Router) getHTMLReport(c *gin.Context) {
	sessionID := c.Param("sessionId")

	zipPath, err := report.GenerateHTML(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "generate report failed: " + err.Error()})
		return
	}

	// 提取文件名作为下载名
	zipName := filepath.Base(zipPath)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipName))
	c.Header("Content-Type", "application/zip")
	c.File(zipPath)
}

// configDownload 平台下发配置
// POST /api/test/config
// Body: {"gunNumber":"5023001201","items":[{"funcCode":194,"payload":{...}}]}
func (r *Router) configDownload(c *gin.Context) {
	var req struct {
		GunNumber string                `json:"gunNumber"`
		Items     []scenario.ConfigItem `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "items is required"})
		return
	}

	postNo, err := strconv.ParseUint(req.GunNumber, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid gunNumber"})
		return
	}

	pn := uint32(postNo)

	// 统一使用 connRegistry 校验连接状态
	if !r.connRegistry.IsConnected(pn, r.sessMgr) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "device not connected"})
		return
	}

	// 获取或创建会话
	sess, hasTCPSession := r.sessMgr.GetByPostNo(pn)
	if !hasTCPSession {
		sess, _ = r.sessMgr.Create(pn, "web-ui-config-"+req.GunNumber)
		sess.SetAuthState(session.Authenticated)
	}

	sc, scenarioID, err := r.scenarioEngine.StartConfigScenario(sess.ID, req.Items)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	result := sc.Result()
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId":   sess.ID,
			"scenarioId":  scenarioID,
			"status":      string(result.State),
			"testCase":    "config_download",
			"progress":    result.Progress,
			"stepName":    result.StepName,
			"stepTotal":   result.StepTotal,
		},
	})
}

// apiLoggingMiddleware 记录所有 Web↔Gater 接口请求和响应
// - GET请求：记录URL查询参数
// - POST请求：记录请求Body（JSON格式化）
// - 所有请求：记录响应状态码和响应Body
// - 使用结构化日志写入gater.log，标记 [GATER→WEB] / [WEB→GATER]
func (r *Router) apiLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 记录请求参数
		var reqLog string
		if method == "GET" {
			if query != "" {
				reqLog = "[WEB→GATER] " + method + " " + path + "?" + query
			} else {
				reqLog = "[WEB→GATER] " + method + " " + path
			}
		} else if method == "POST" || method == "PUT" {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil && len(bodyBytes) > 0 {
				// 还原body供后续handler读取
				c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				// 紧凑JSON输出（单行，方便日志采集和分析）
				var compactJSON map[string]interface{}
				if json.Unmarshal(bodyBytes, &compactJSON) == nil {
					if compact, fmtErr := json.Marshal(compactJSON); fmtErr == nil {
						reqLog = "[WEB→GATER] " + method + " " + path + " " + string(compact)
					} else {
						reqLog = "[WEB→GATER] " + method + " " + path + " " + string(bodyBytes)
					}
				} else {
					reqLog = "[WEB→GATER] " + method + " " + path + " " + string(bodyBytes)
				}
			} else {
				reqLog = "[WEB→GATER] " + method + " " + path
			}
		} else {
			reqLog = "[WEB→GATER] " + method + " " + path
		}
		r.logger.Info(reqLog)

		// 捕获响应
		w := &responseBodyWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = w

		c.Next()

		// 记录响应
		statusCode := w.Status()
		respBody := w.body.String()
		if respBody != "" {
			var compactResp map[string]interface{}
			if json.Unmarshal([]byte(respBody), &compactResp) == nil {
				if compact, fmtErr := json.Marshal(compactResp); fmtErr == nil {
					r.logger.Infof("[GATER→WEB] %s %s %d %s", method, path, statusCode, string(compact))
				} else {
					r.logger.Infof("[GATER→WEB] %s %s %d %s", method, path, statusCode, respBody)
				}
			} else {
				r.logger.Infof("[GATER→WEB] %s %s %d %s", method, path, statusCode, respBody)
			}
		} else {
			r.logger.Infof("[GATER→WEB] %s %s %d", method, path, statusCode)
		}
	}
}

// responseBodyWriter 用于捕获gin响应Body
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
