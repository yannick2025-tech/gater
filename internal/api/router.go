// Package api provides HTTP API handlers for the web frontend.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yannick2025-tech/nts-gater/internal/model"
	"github.com/yannick2025-tech/nts-gater/internal/report"
	"github.com/yannick2025-tech/nts-gater/internal/recorder"
	"github.com/yannick2025-tech/nts-gater/internal/scenario"
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
	logger         logging.Logger
}

// NewRouter 创建路由
func NewRouter(sessMgr *session.SessionManager, scenarioEngine *scenario.Engine, logger logging.Logger) *Router {
	return &Router{
		sessMgr:        sessMgr,
		scenarioEngine: scenarioEngine,
		connRegistry:   NewDeviceConnectionRegistry(),
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
		api.GET("/test/status/:sessionId", r.getTestStatus)
		api.GET("/test/results", r.getTestResults)
		api.GET("/test/detail/:sessionId", r.getTestDetail)
		api.GET("/test/messages/:sessionId", r.getMessages)
		api.POST("/test/decode", r.decodeMessage)
		api.POST("/test/export", r.exportReport)
		api.GET("/test/download", r.downloadPDF)
		api.POST("/test/config", r.configDownload)
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

// getSessions 获取所有TCP会话列表（活跃内存会话 + 数据库历史会话）
// GET /api/sessions
func (r *Router) getSessions(c *gin.Context) {
	// 1. 收集内存中活跃的session（用于去重）
	activeSessionIDs := make(map[string]bool)

	// 2. 内存中的活跃会话
	sessions := r.sessMgr.GetAllSessions()
	list := make([]gin.H, 0, len(sessions))
	for _, sess := range sessions {
		authState := sess.GetAuthState().String()
		item := gin.H{
			"sessionId":       sess.ID,
			"postNo":          sess.PostNo,
			"gunNumber":       fmt.Sprintf("%d", sess.PostNo),
			"authState":       authState,
			"isOnline":        sess.IsConnected(),
			"protocolName":    "XX标准协议",
			"protocolVersion": "v1.6.0",
			"connectedAt":     sess.CreatedAt.Format("2006-01-02 15:04:05"),
			"lastActive":      sess.LastActive.Format("2006-01-02 15:04:05"),
		}
		// 补充测试状态（如正在运行场景）
		if sc, ok := r.scenarioEngine.GetScenario(sess.ID); ok {
			if result := sc.Result(); result != nil {
				item["testStatus"] = string(result.State)
				item["testCase"] = result.ScenarioName
				item["testProgress"] = result.Progress
			}
		}
		list = append(list, item)
		activeSessionIDs[sess.ID] = true
	}

	// 3. 从数据库加载历史会话报告（补充不在内存中的已结束会话）
	if dbReports, err := report.GetAllSessionSummaries(); err == nil {
		for _, rep := range dbReports {
			// 跳过已在内存中活跃的session，避免重复
			if activeSessionIDs[rep.SessionID] {
				continue
			}
			list = append(list, gin.H{
				"sessionId":       rep.SessionID,
				"postNo":          rep.PostNo,
				"gunNumber":       fmt.Sprintf("%d", rep.PostNo),
				"authState":       rep.AuthState,
				"isOnline":        false,
				"protocolName":    rep.ProtocolName,
				"protocolVersion": rep.ProtocolVer,
				"connectedAt":     rep.StartTime.Format("2006-01-02 15:04:05"),
				"lastActive":      rep.EndTime.Format("2006-01-02 15:04:05"),
			})
		}
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

	// 2. 立即返回成功（不等待后续清理操作）
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"isOnline": false}})

	// 3. 异步执行耗时清理操作（按顺序：停场景 → 关记录器 → 存报告 → 移会话）
	go func() {
		if sess, ok := r.sessMgr.GetByPostNo(postNo); ok {
			// 3.1 先停止场景引擎（防止继续发送报文/轮询running）
			r.scenarioEngine.RemoveScenario(sess.ID)

			// 3.2 关闭记录器（停止接收新消息，锁定统计）
			if sess.Recorder != nil {
				sess.Recorder.Close()
			}

			// 3.3 构建会话摘要并保存报告到数据库
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

			err := report.SaveReport(summary, "XX标准协议", "v1.6.0", sess.GetAuthState().String())
			if err != nil {
				fmt.Printf("[disconnect] save report error for session %s: %v\n", sess.ID, err)
			} else {
				fmt.Printf("[disconnect] report saved for session %s (postNo=%d)\n", sess.ID, postNo)
			}

			// 3.4 最后移除会话（必须在SaveReport之后，因为Remove也会Close Recorder）
			r.sessMgr.Remove(sess.ID)
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

	sc, err := r.scenarioEngine.StartScenario(sess.ID, req.TestCase, req.Params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	result := sc.Result()
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId": sess.ID,
			"status":    string(result.State),
			"testCase":  req.TestCase,
			"progress":  result.Progress,
			"stepName":  result.StepName,
		},
	})
}

// getTestStatus 查询测试进度
// GET /api/test/status/:sessionId
func (r *Router) getTestStatus(c *gin.Context) {
	sessionID := c.Param("sessionId")

	// 1. 先尝试从内存获取session
	sess, ok := r.sessMgr.Get(sessionID)
	if !ok {
		// 2. session已不在内存中（已被断开/超时移除）→ 查数据库
		testReport, err := report.GetTestReportBySessionID(sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "report not found"})
			return
		}
		// 返回数据库中的最终状态
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
				"testCase":  "",
			},
		})
		return
	}

	status := "connected"
	progress := 0
	stepName := ""
	testCase := ""

	if sc, ok := r.scenarioEngine.GetScenario(sessionID); ok {
		result := sc.Result()
		status = string(result.State)
		progress = result.Progress
		stepName = result.StepName
		testCase = result.ScenarioName
	} else if sess.Recorder != nil && sess.Recorder.EndTime().IsZero() {
		status = "connected"
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId": sessionID,
			"status":    status,
			"progress":  progress,
			"stepName":  stepName,
			"testCase":  testCase,
		},
	})
}

// getTestResults 获取测试结果列表
// GET /api/test/results?gunNumber=&page=1&pageSize=10&startTime=&endTime=
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

	reports, total, err := report.GetTestReports(page, pageSize, startTime, endTime)
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

// getTestDetail 获取测试详情
// GET /api/test/detail/:sessionId
func (r *Router) getTestDetail(c *gin.Context) {
	sessionID := c.Param("sessionId")

	testReport, err := report.GetTestReportBySessionID(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "report not found"})
		return
	}

	stats, _ := report.GetFuncCodeStatsBySessionID(sessionID)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId": sessionID,
			"startTime": testReport.StartTime,
			"endTime":   testReport.EndTime,
			"status": func() string {
				if testReport.IsPass {
					return "pass"
				}
				return "fail"
			}(),
		"statistics": stats,
	},
	})
}

// getMessages 获取会话的所有报文存档
// GET /api/test/messages/:sessionId?funcCode=&status=
func (r *Router) getMessages(c *gin.Context) {
	sessionID := c.Param("sessionId")
	funcCode := c.Query("funcCode")
	status := c.Query("status")

	archives, err := report.GetMessageArchivesBySessionID(sessionID, funcCode, status)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"list": []interface{}{}}})
		return
	}

	if archives == nil {
		archives = []model.MessageArchive{}
	}

	// Convert to JSON-friendly format
	list := make([]gin.H, len(archives))
	for i, a := range archives {
		list[i] = gin.H{
			"id":        a.ID,
			"sessionId": sessionID,
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

// exportReport 导出测试报告
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

	testReport, err := report.GetTestReportBySessionID(req.SessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "report not found"})
		return
	}

	stats, _ := report.GetFuncCodeStatsBySessionID(req.SessionID)
	archives, _ := report.GetMessageArchivesBySessionID(req.SessionID, "", "")

	pdfPath, err := report.GeneratePDF(testReport, stats, archives)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "generate pdf failed: " + err.Error()})
		return
	}

	_ = report.UpdateReportPDFPath(req.SessionID, pdfPath)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId": req.SessionID,
			"pdfUrl":    "/api/test/download?path=" + pdfPath,
			"pdfPath":   pdfPath,
		},
	})
}

// downloadPDF 下载PDF文件
// GET /api/test/download?path=reports/report_xxx.pdf
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

	sc, err := r.scenarioEngine.StartConfigScenario(sess.ID, req.Items)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

		result := sc.Result()
	c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"sessionId": sess.ID,
				"status":    string(result.State),
				"testCase":  "config_download",
				"progress":  result.Progress,
				"stepName":  result.StepName,
				"stepTotal": result.StepTotal,
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
