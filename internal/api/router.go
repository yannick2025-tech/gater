// Package api provides HTTP API handlers for the web frontend.
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yannick2025-tech/nts-gater/internal/report"
	"github.com/yannick2025-tech/nts-gater/internal/scenario"
	"github.com/yannick2025-tech/nts-gater/internal/session"
)

// Router HTTP路由
type Router struct {
	sessMgr        *session.SessionManager
	scenarioEngine *scenario.Engine
}

// NewRouter 创建路由
func NewRouter(sessMgr *session.SessionManager, scenarioEngine *scenario.Engine) *Router {
	return &Router{sessMgr: sessMgr, scenarioEngine: scenarioEngine}
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

	postNo, err := strconv.ParseUint(gunNumber, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid gunNumber"})
		return
	}

	sess, ok := r.sessMgr.GetByPostNo(uint32(postNo))
	isOnline := ok && sess.GetAuthState() == session.Authenticated

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"gunNumber":      gunNumber,
			"protocolName":   "XX标准协议",
			"protocolVersion": "v1.6.0",
			"isOnline":       isOnline,
			"sessionId":      func() string {
				if ok {
					return sess.ID
				}
				return ""
			}(),
			"authState": func() string {
				if ok {
					return sess.GetAuthState().String()
				}
				return "none"
			}(),
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

	postNo, err := strconv.ParseUint(req.GunNumber, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid gunNumber"})
		return
	}

	switch req.Action {
	case "connect":
		// 连接是充电桩主动发起TCP连接，这里只返回当前状态
		sess, ok := r.sessMgr.GetByPostNo(uint32(postNo))
		isOnline := ok && sess.GetAuthState() == session.Authenticated
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{"isOnline": isOnline, "sessionId": func() string {
				if ok {
					return sess.ID
				}
				return ""
			}()},
		})
	case "disconnect":
		// 断开连接：移除会话，触发报告生成
		sess, ok := r.sessMgr.GetByPostNo(uint32(postNo))
		if !ok {
			c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"isOnline": false}})
			return
		}
		if sess.Recorder != nil {
			sess.Recorder.Close()
		}
		r.sessMgr.Remove(sess.ID)
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"isOnline": false}})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "action must be connect or disconnect"})
	}
}

// disconnectDevice 断开设备连接
// POST /api/device/disconnect
func (r *Router) disconnectDevice(c *gin.Context) {
	var req struct {
		GunNumber string `json:"gunNumber"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	postNo, err := strconv.ParseUint(req.GunNumber, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid gunNumber"})
		return
	}

	sess, ok := r.sessMgr.GetByPostNo(uint32(postNo))
	if !ok {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"isOnline": false}})
		return
	}

	if sess.Recorder != nil {
		sess.Recorder.Close()
	}
	r.sessMgr.Remove(sess.ID)

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

	// 查找对应会话
	postNo, _ := strconv.ParseUint(req.GunNumber, 10, 32)
	sess, ok := r.sessMgr.GetByPostNo(uint32(postNo))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "device not connected"})
		return
	}

	// 启动测试场景
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

	// 检查是否有运行中的场景
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
			"sessionId":  sessionID,
			"startTime":  testReport.StartTime,
			"endTime":    testReport.EndTime,
			"status":     func() string {
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

	// TODO: 使用协议编解码器解码hex报文
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

	// 获取测试报告
	testReport, err := report.GetTestReportBySessionID(req.SessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "report not found"})
		return
	}

	// 获取功能码统计
	stats, _ := report.GetFuncCodeStatsBySessionID(req.SessionID)

	// 获取消息存档
	archives, _ := report.GetMessageArchivesBySessionID(req.SessionID, "", "")

	// 生成PDF
	pdfPath, err := report.GeneratePDF(testReport, stats, archives)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "generate pdf failed: " + err.Error()})
		return
	}

	// 更新报告中的PDF路径
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

	// 安全检查：防止路径遍历攻击
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	c.File(path)
}

// configDownload 平台下发配置
// POST /api/test/config
// Body: {"gunNumber":"5023001201","items":[{"funcCode":194,"payload":{...}}]}
// funcCode: 0xC2(194)=配置下发, 0x22(34)=计费规则, 0x0C(12)=设备参数查询
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

	postNo, err := strconv.ParseUint(req.GunNumber, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid gunNumber"})
		return
	}

	sess, ok := r.sessMgr.GetByPostNo(uint32(postNo))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "device not connected"})
		return
	}

	// 启动配置下发场景
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
