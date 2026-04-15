package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yannick2025-tech/nts-gater/internal/report"
	"github.com/yannick2025-tech/nts-gater/internal/session"
)

// Router HTTP路由
type Router struct {
	sessMgr *session.SessionManager
}

// NewRouter 创建路由
func NewRouter(sessMgr *session.SessionManager) *Router {
	return &Router{sessMgr: sessMgr}
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
// Body: {"testCase":"basic","balance":"100.00","stopCode":"1234",...}
func (r *Router) startTest(c *gin.Context) {
	var req struct {
		TestCase string `json:"testCase"`
		GunNumber string `json:"gunNumber"`
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

	// TODO: M2.2 测试场景引擎 - 根据testCase启动对应场景
	// 当前只记录测试开始
	_ = sess

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId": sess.ID,
			"status":    "running",
			"testCase":  req.TestCase,
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

	status := "completed"
	if sess.Recorder != nil && sess.Recorder.EndTime().IsZero() {
		status = "running"
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId": sessionID,
			"status":    status,
			"progress":  0, // TODO: 根据测试场景计算进度
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

	// TODO: M2.5 PDF生成
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"sessionId": req.SessionID,
			"pdfUrl":    "",
		},
	})
}
