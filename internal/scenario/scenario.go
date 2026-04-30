// Package scenario provides the test scenario engine and Scenario interface.
package scenario

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yannick2025-tech/nts-gater/internal/generator"
	"github.com/yannick2025-tech/nts-gater/internal/model"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard/msg"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	"github.com/yannick2025-tech/nts-gater/internal/recorder"
	"github.com/yannick2025-tech/nts-gater/internal/report"
	"github.com/yannick2025-tech/nts-gater/internal/server"
	"github.com/yannick2025-tech/nts-gater/internal/session"
	logging "github.com/yannick2025-tech/gwc-logging"
)

// ScenarioState 场景状态
type ScenarioState string

const (
	StateIdle      ScenarioState = "idle"      // 空闲
	StateRunning   ScenarioState = "running"   // 运行中
	StatePaused    ScenarioState = "paused"    // 暂停
	StateCompleted ScenarioState = "completed" // 已完成
	StateFailed    ScenarioState = "failed"    // 失败
)

// ScenarioResult 场景执行结果
type ScenarioResult struct {
	ScenarioName string        `json:"scenarioName"`
	State        ScenarioState `json:"state"`
	StartTime    time.Time     `json:"startTime"`
	EndTime      time.Time     `json:"endTime"`
	Duration     time.Duration `json:"duration"`
	StepIndex    int           `json:"stepIndex"`    // 当前步骤
	StepTotal    int           `json:"stepTotal"`    // 总步骤数
	StepName     string        `json:"stepName"`     // 当前步骤名称
	Progress     int           `json:"progress"`     // 进度百分比 0-100
	ErrorMsg     string        `json:"errorMsg"`     // 错误信息
}

// Scenario 测试场景接口
type Scenario interface {
	// Name 场景名称
	Name() string

	// Start 启动场景
	Start() error

	// Stop 停止场景
	Stop()

	// State 获取当前状态
	State() ScenarioState

	// Result 获取执行结果
	Result() *ScenarioResult

	// OnMessage 收到消息时的回调（用于驱动状态机）
	OnMessage(funcCode byte, dir types.Direction, msg types.Message)

	// SetSendFunc 设置发送消息函数
	SetSendFunc(fn SendFunc)
}

// SendFunc 发送消息到充电桩的函数
type SendFunc func(msg types.Message) error

// ==================== ScenarioEngine ====================

// Engine 场景引擎
type Engine struct {
	sessMgr  *session.SessionManager
	srv      *server.Server
	proto    types.Protocol
	logger   logging.Logger
	scenarios map[string]Scenario // sessionID -> Scenario
}

// NewEngine 创建场景引擎
func NewEngine(sessMgr *session.SessionManager, srv *server.Server, proto types.Protocol, logger logging.Logger) *Engine {
	return &Engine{
		sessMgr:   sessMgr,
		srv:       srv,
		proto:     proto,
		logger:    logger,
		scenarios: make(map[string]Scenario),
	}
}

// StartScenario 启动测试场景，返回场景实例和场景UUID
func (e *Engine) StartScenario(sessionID string, testCase string, params map[string]interface{}) (Scenario, string, error) {
	sess, ok := e.sessMgr.Get(sessionID)
	if !ok {
		return nil, "", ErrSessionNotFound
	}

	// 检查是否已有运行中的场景
	if existing, ok := e.scenarios[sessionID]; ok && existing.State() == StateRunning {
		return nil, "", ErrScenarioAlreadyRunning
	}

	// 查找TCP连接（Web端模式可能没有真实TCP连接）
	var sendFn SendFunc
	if conn, ok := e.srv.FindConnectionByPostNo(sess.PostNo); ok {
		sendFn = e.createSendFn(sess, conn)
	} else {
		// Web端无真实TCP连接：使用空发送函数
		sendFn = func(msg types.Message) error { return nil }
	}

	// 创建场景
	var sc Scenario
	switch testCase {
	case "basic_charging":
		sc = NewBasicChargingScenario(sessionID, sess, e.proto, e.logger)
	case "sftp_upgrade":
		sc = NewSFTPUpgradeScenario(sessionID, sess, e.proto, e.logger)
	case "config_download":
		sc = NewConfigDownloadScenario(sessionID, sess, e.proto, e.logger)
	default:
		return nil, "", ErrUnknownTestCase
	}

	// 设置场景参数
	if paramsSetter, ok := sc.(interface{ SetParams(map[string]interface{}) }); ok && params != nil {
		paramsSetter.SetParams(params)
	}

	sc.SetSendFunc(sendFn)
	if err := sc.Start(); err != nil {
		return nil, "", err
	}

	e.scenarios[sessionID] = sc
	e.logger.Infof("[scenario] started %s for session=%s postNo=%d", sc.Name(), sessionID, sess.PostNo)

	// 生成场景UUID（与 CreateRunningReport 使用同一个ID）
	scenarioID := report.GenerateScenarioID()

	// 同步创建默认用例记录（直接传scenarioID，无需查询DB，无竞态条件）
	e.createDefaultTestCase(sessionID, sess, testCase, scenarioID)

	return sc, scenarioID, nil
}

// createDefaultTestCase 为场景创建默认的测试用例记录（同步调用，scenarioID由调用方传入）
func (e *Engine) createDefaultTestCase(sessionID string, sess *session.Session, testCase string, scenarioID string) {
	caseID := generateCaseID(testCase)
	caseName := getCaseName(testCase)

	scenarioName := ""
	switch testCase {
	case "basic_charging":
		scenarioName = "业务场景测试"
	case "sftp_upgrade":
		scenarioName = "SFTP升级测试"
	case "config_download":
		scenarioName = "配置下发测试"
	}

	tc := &model.TestCase{
		SessionID:    sessionID,
		ScenarioID:   scenarioID,
		ScenarioName: scenarioName,
		CaseID:       caseID,
		CaseName:     caseName,
		CaseType:     testCase,
		Status:       "running",
		StartTime:    time.Now(),
	}

	if err := report.SaveTestCase(tc); err != nil {
		e.logger.Warnf("[scenario] save test case warning: %v", err)
	}

	// 设置 Recorder 当前用例（后续 RecordRecv/Send/Reply 会自动关联此 caseID）
	if sess.Recorder != nil {
		sess.Recorder.SetCurrentCase(caseID)
	}
}

// generateCaseID 生成用例编号
func generateCaseID(testCase string) string {
	switch testCase {
	case "basic_charging":
		return "TC-BC-01"
	case "sftp_upgrade":
		return "TC-SU-01"
	case "config_download":
		return "TC-CD-01"
	default:
		return "TC-00-01"
	}
}

// getCaseName 获取用例名称
func getCaseName(testCase string) string {
	switch testCase {
	case "basic_charging":
		return "业务场景测试"
	case "sftp_upgrade":
		return "SFTP升级测试"
	case "config_download":
		return "配置下发测试"
	default:
		return "未知测试"
	}
}

// StopScenario 停止测试场景
func (e *Engine) StopScenario(sessionID string) {
	sc, ok := e.scenarios[sessionID]
	if !ok {
		return
	}
	sc.Stop()
	e.logger.Infof("[scenario] stopped %s for session=%s", sc.Name(), sessionID)
}

// GetScenario 获取场景
func (e *Engine) GetScenario(sessionID string) (Scenario, bool) {
	sc, ok := e.scenarios[sessionID]
	return sc, ok
}

// OnMessage 收到消息时通知引擎（由 dispatcher 调用）
func (e *Engine) OnMessage(sessionID string, funcCode byte, dir types.Direction, msg types.Message) {
	sc, ok := e.scenarios[sessionID]
	if !ok {
		return
	}
	sc.OnMessage(funcCode, dir, msg)
}

// RemoveScenario 移除场景（会话结束时调用）
func (e *Engine) RemoveScenario(sessionID string) {
	if sc, ok := e.scenarios[sessionID]; ok {
		sc.Stop()
		delete(e.scenarios, sessionID)
	}
}

// SendStopCharge 发送0x08停止充电消息到充电桩
func (e *Engine) SendStopCharge(sessionID string) error {
	sess, ok := e.sessMgr.Get(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	conn, ok := e.srv.FindConnectionByPostNo(sess.PostNo)
	if !ok {
		return fmt.Errorf("connection not found for postNo=%d", sess.PostNo)
	}

	stopMsg := &msg.PlatformStopDownload{
		PlatformOrderNumber:       generator.GenerateOrderNo(),
		PileChargingFailureReason: 0x00, // 正常停止
	}

	sendFn := e.createSendFn(sess, conn)
	if err := sendFn(stopMsg); err != nil {
		return fmt.Errorf("send 0x08 failed: %w", err)
	}

	e.logger.Infof("[engine] sent 0x08 stop charge for session=%s postNo=%d", sessionID, sess.PostNo)
	return nil
}

// StartConfigScenario 启动配置下发场景（需要额外的JSON配置项）
func (e *Engine) StartConfigScenario(sessionID string, items []ConfigItem) (Scenario, error) {
	sess, ok := e.sessMgr.Get(sessionID)
	if !ok {
		return nil, ErrSessionNotFound
	}

	if existing, ok := e.scenarios[sessionID]; ok && existing.State() == StateRunning {
		return nil, ErrScenarioAlreadyRunning
	}

	// 查找TCP连接（Web端模式可能没有真实TCP连接）
	var sendFn SendFunc
	conn, ok := e.srv.FindConnectionByPostNo(sess.PostNo)
	if !ok {
		// Web端无真实TCP连接：使用空发送函数
		sendFn = func(msg types.Message) error { return nil }
	} else {
		sendFn = e.createSendFn(sess, conn)
	}

	sc := NewConfigDownloadScenario(sessionID, sess, e.proto, e.logger)
	if err := sc.SetConfigItems(items); err != nil {
		return nil, err
	}

	sc.SetSendFunc(sendFn)
	if err := sc.Start(); err != nil {
		return nil, err
	}

	e.scenarios[sessionID] = sc
	e.logger.Infof("[scenario] started %s for session=%s postNo=%d items=%d", sc.Name(), sessionID, sess.PostNo, len(items))

	return sc, nil
}

// createSendFunc 创建消息发送函数
func (e *Engine) createSendFn(sess *session.Session, conn *server.Connection) SendFunc {
	return func(msg types.Message) error {
		data, err := msg.Encode()
		if err != nil {
			return err
		}

		spec := msg.Spec()
		header := types.MessageHeader{
			StartByte:   e.proto.FrameConfig().StartByte,
			Version:     e.proto.Version(),
			FuncCode:    spec.FuncCode,
			PostNo:      sess.PostNo,
			Charger:     1,
			EncryptFlag: 0x00,
		}
		if spec.Encrypt {
			header.EncryptFlag = 0x01
		}

		// 编码完整帧（用于日志和存档）
		encryptFn := sess.GetEncryptFn()
		frame, encErr := conn.Encoder.Encode(header, data, encryptFn)
		if encErr != nil {
			return fmt.Errorf("encode frame failed: %w", encErr)
		}

		frameHex := fmt.Sprintf("% X", frame)
		msgJSON, _ := json.Marshal(msg.ToJSONMap())
		msgJSONStr := string(msgJSON)

		// 记录平台主动下发消息到 Recorder
		if sess.Recorder != nil {
			sess.Recorder.RecordSend(spec.FuncCode, recorder.StatusSuccess, frameHex, msgJSONStr, "")
		}

		// 日志
		e.logger.Infof("[%s] [GATER→Post] [0x%02X] postNo=%d charger=%d dataLen=%d",
			sess.ID, spec.FuncCode, sess.PostNo, header.Charger, len(data))
		e.logger.Infof("[%s] [GATER→Post] [0x%02X] HEX: %s", sess.ID, spec.FuncCode, frameHex)
		e.logger.Infof("[%s] [GATER→Post] [0x%02X] JSON: %s", sess.ID, spec.FuncCode, msgJSONStr)

		return conn.SendFrame(frame)
	}
}
