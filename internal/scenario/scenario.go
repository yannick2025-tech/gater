// Package scenario provides the test scenario engine and Scenario interface.
package scenario

import (
	"time"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
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

// StartScenario 启动测试场景
func (e *Engine) StartScenario(sessionID string, testCase string) (Scenario, error) {
	sess, ok := e.sessMgr.Get(sessionID)
	if !ok {
		return nil, ErrSessionNotFound
	}

	// 检查是否已有运行中的场景
	if existing, ok := e.scenarios[sessionID]; ok && existing.State() == StateRunning {
		return nil, ErrScenarioAlreadyRunning
	}

	// 查找TCP连接
	conn, ok := e.srv.FindConnectionByPostNo(sess.PostNo)
	if !ok {
		return nil, ErrConnectionNotFound
	}

	sendFn := e.createSendFn(sess, conn)

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
		return nil, ErrUnknownTestCase
	}

	sc.SetSendFunc(sendFn)
	if err := sc.Start(); err != nil {
		return nil, err
	}

	e.scenarios[sessionID] = sc
	e.logger.Infof("[scenario] started %s for session=%s postNo=%d", sc.Name(), sessionID, sess.PostNo)

	return sc, nil
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

// StartConfigScenario 启动配置下发场景（需要额外的JSON配置项）
func (e *Engine) StartConfigScenario(sessionID string, items []ConfigItem) (Scenario, error) {
	sess, ok := e.sessMgr.Get(sessionID)
	if !ok {
		return nil, ErrSessionNotFound
	}

	if existing, ok := e.scenarios[sessionID]; ok && existing.State() == StateRunning {
		return nil, ErrScenarioAlreadyRunning
	}

	conn, ok := e.srv.FindConnectionByPostNo(sess.PostNo)
	if !ok {
		return nil, ErrConnectionNotFound
	}

	sendFn := e.createSendFn(sess, conn)

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

		encryptFn := sess.GetEncryptFn()

		return conn.Send(header, data, encryptFn)
	}
}
