// Package api provides HTTP API handlers for the web frontend.
package api

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yannick2025-tech/nts-gater/internal/report"
	"github.com/yannick2025-tech/nts-gater/internal/scenario"
	"github.com/yannick2025-tech/nts-gater/internal/session"
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
}

// NewRouter 创建路由
func NewRouter(sessMgr *session.SessionManager, scenarioEngine *scenario.Engine) *Router {
	return &Router{
		sessMgr:        sessMgr,
		scenarioEngine: scenarioEngine,
		connRegistry:   NewDeviceConnectionRegistry(),
	}
}

// Setup 设置路由
func (r *Router) Setup(engine *gin.Engine) {
	api := engine.Group("/api")
	{
		// 设备管理
		api.GET("/device/status", r.getDeviceStatus)
		api.POST("/device/connect", r.toggleConnection)
		api.POST("/device/disconnect", r.disconnectDevice)

		// 测试管理
		api.POST("/test/start", r.startTest)
		api.GET("/test/status/:sessionId", r.getTestStatus)
		api.GET("/test/results", r.getTestResults)
		api.GET("/test/detail/:sessionId", r.getTestDetail)
		api.POST("/test/decode", r.decodeMessage)
		api.POST("/test/export", r.exportReport)
		api.GET("/test/download", r.downloadPDF)
		api.POST("/test/config", r.configDownload)
	}
}

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

// handleDisconnect 统一断开逻辑：清理Web注册表 + 清理TCP会话
func (r *Router) handleDisconnect(c *gin.Context, postNo uint32) {
	// 1. 从Web注册表移除
	r.connRegistry.Unregister(postNo)

	// 2. 如果有TCP会话也一并清理
	if sess, ok := r.sessMgr.GetByPostNo(postNo); ok {
		if sess.Recorder != nil {
			sess.Recorder.Close()
		}
		r.sessMgr.Remove(sess.ID)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"isOnline": false}})
}

// ==================== 测试接口 ====================

// startTest 开始测试
// POST /api/test/start
// Body: {"testCase":"basic_charging","gunNumber":"5023001201","params":{...}}
func (r *Router) startTest(c *gin.Context) {
	var req struct {
		TestCase  string                 `json:"testCase"`
		GunNumber string                 `json:"gunNumber"`
		Params    map[string]interface{} `json:"params"`
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

	// 统一使用 connRegistry 校验连接状态（Web注册表 + TCP会话）
	if !r.connRegistry.IsConnected(pn, r.sessMgr) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "device not connected"})
		return
	}

	// 尝试获取或创建会话用于测试
	sess, hasTCPSession := r.sessMgr.GetByPostNo(pn)
	if !hasTCPSession {
		// 无TCP会话时创建一个虚拟会话用于Web端测试场景
		sess, _ = r.sessMgr.Create(pn, "web-ui-"+req.GunNumber)
		// 将此会话标记为已认证（因为Web端已经确认连接）
		sess.SetAuthState(session.Authenticated)
	}

	sc, err := r.scenarioEngine.StartScenario(sess.ID, req.TestCase)
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

	sess, ok := r.sessMgr.Get(sessionID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "session not found"})
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
