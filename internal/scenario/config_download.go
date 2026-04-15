// Package scenario provides the platform config download test scenario.
package scenario

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard/msg"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	"github.com/yannick2025-tech/nts-gater/internal/session"
	logging "github.com/yannick2025-tech/gwc-logging"
)

// ==================== 平台下发配置测试场景 ====================
//
// 流程：
// 用户在WEB页面通过JSON输入配置参数，前端将JSON通过API发给gater，
// gater校验JSON合法性后下发二进制报文到充电桩，等待桩回复。
//
// 支持的功能码：
// - 0xC2 配置信息下发
// - 0x22 分时段计费规则下发
// - 0x0C 设备参数查询
//
// 每次下发一条，等待桩回复后再处理下一条。

// ConfigDownloadStep 配置下发步骤
type ConfigDownloadStep struct {
	Name      string `json:"name"`
	FuncCode  byte   `json:"funcCode"`
	Direction types.Direction
	Completed bool   `json:"completed"`
	Result    string `json:"result"`
}

// ConfigDownloadScenario 平台下发配置测试场景
type ConfigDownloadScenario struct {
	sessionID string
	sess      *session.Session
	proto     types.Protocol
	logger    logging.Logger

	mu       sync.RWMutex
	state    ScenarioState
	stepIdx  int
	result   ScenarioResult
	sendFn   SendFunc
	stopCh   chan struct{}

	// 配置下发队列：JSON参数列表
	steps []ConfigDownloadStep

	// 待下发的消息（校验通过后构建）
	pendingMsg types.Message
}

// ConfigDownloadRequest 前端请求结构
type ConfigDownloadRequest struct {
	Items []ConfigItem `json:"items"`
}

// ConfigItem 单个配置项
type ConfigItem struct {
	FuncCode byte            `json:"funcCode"` // 0xC2 / 0x22 / 0x0C
	Payload  json.RawMessage `json:"payload"`  // JSON格式的消息体
}

// NewConfigDownloadScenario 创建平台下发配置测试场景
func NewConfigDownloadScenario(sessionID string, sess *session.Session, proto types.Protocol, logger logging.Logger) *ConfigDownloadScenario {
	return &ConfigDownloadScenario{
		sessionID: sessionID,
		sess:      sess,
		proto:     proto,
		logger:    logger,
		state:     StateIdle,
	}
}

// Name 场景名称
func (s *ConfigDownloadScenario) Name() string { return "config_download" }

// State 获取当前状态
func (s *ConfigDownloadScenario) State() ScenarioState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.state
}

// Result 获取执行结果
func (s *ConfigDownloadScenario) Result() *ScenarioResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r := s.result

	return &r
}

// SetSendFunc 设置发送函数
func (s *ConfigDownloadScenario) SetSendFunc(fn SendFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sendFn = fn
}

// SetConfigItems 设置配置项列表（在Start之前调用）
func (s *ConfigDownloadScenario) SetConfigItems(items []ConfigItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	steps := make([]ConfigDownloadStep, 0, len(items))

	for i, item := range items {
		stepName := ""
		var pendingMsg types.Message

		switch item.FuncCode {
		case types.FuncConfigDownload: // 0xC2
			stepName = fmt.Sprintf("下发配置信息(0xC2) [%d]", i+1)
			var payload msg.ConfigDownloadMsg
			if err := json.Unmarshal(item.Payload, &payload); err != nil {
				return fmt.Errorf("item[%d]: invalid 0xC2 payload: %v", i, err)
			}
			if errs := payload.Validate(); len(errs) > 0 {
				return fmt.Errorf("item[%d]: 0xC2 validation failed: %v", i, errs)
			}
			pendingMsg = &payload

		case types.FuncBillingRules: // 0x22
			stepName = fmt.Sprintf("下发计费规则(0x22) [%d]", i+1)
			var payload msg.BillingRulesDownload
			if err := json.Unmarshal(item.Payload, &payload); err != nil {
				return fmt.Errorf("item[%d]: invalid 0x22 payload: %v", i, err)
			}
			if errs := payload.Validate(); len(errs) > 0 {
				return fmt.Errorf("item[%d]: 0x22 validation failed: %v", i, errs)
			}
			pendingMsg = &payload

		case types.FuncDeviceQuery: // 0x0C
			stepName = fmt.Sprintf("查询设备参数(0x0C) [%d]", i+1)
			var payload msg.DeviceQueryDownload
			if err := json.Unmarshal(item.Payload, &payload); err != nil {
				return fmt.Errorf("item[%d]: invalid 0x0C payload: %v", i, err)
			}
			if errs := payload.Validate(); len(errs) > 0 {
				return fmt.Errorf("item[%d]: 0x0C validation failed: %v", i, errs)
			}
			pendingMsg = &payload

		default:
			return fmt.Errorf("item[%d]: unsupported funcCode 0x%02X (supported: 0xC2, 0x22, 0x0C)", i, item.FuncCode)
		}

		_ = pendingMsg // will be used in Start
		steps = append(steps, ConfigDownloadStep{
			Name:      stepName,
			FuncCode:  item.FuncCode,
			Direction: types.DirectionDownload,
		})
	}

	s.steps = steps

	return nil
}

// Start 启动场景
func (s *ConfigDownloadScenario) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateRunning {
		return ErrScenarioAlreadyRunning
	}

	if s.sendFn == nil {
		return ErrSendNotSet
	}

	if len(s.steps) == 0 {
		return fmt.Errorf("no config items to download")
	}

	s.state = StateRunning
	s.stepIdx = 0
	s.stopCh = make(chan struct{})
	s.result = ScenarioResult{
		ScenarioName: s.Name(),
		State:        StateRunning,
		StartTime:    time.Now(),
		StepTotal:    len(s.steps),
	}

	// 执行第一步
	go s.executeStep()

	return nil
}

// Stop 停止场景
func (s *ConfigDownloadScenario) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateRunning {
		return
	}

	s.state = StateCompleted
	s.result.State = StateCompleted
	s.result.EndTime = time.Now()
	s.result.Duration = s.result.EndTime.Sub(s.result.StartTime)
	s.result.Progress = 100

	if s.stopCh != nil {
		close(s.stopCh)
	}
}

// OnMessage 收到消息回调
func (s *ConfigDownloadScenario) OnMessage(funcCode byte, dir types.Direction, m types.Message) {
	s.mu.RLock()
	running := s.state == StateRunning
	stepIdx := s.stepIdx
	s.mu.RUnlock()

	if !running || stepIdx >= len(s.steps) {
		return
	}

	currentStep := s.steps[stepIdx]

	// 等待桩回复（Reply方向）
	if dir == types.DirectionReply && funcCode == currentStep.FuncCode {
		s.logger.Infof("[scenario:%s] received reply for step %d: func=0x%02X",
			s.sessionID, stepIdx, funcCode)

		// 标记步骤完成
		s.mu.Lock()
		s.steps[stepIdx].Completed = true
		s.steps[stepIdx].Result = "success"
		s.mu.Unlock()

		s.advanceStep()
	}
}

// executeStep 执行当前步骤
func (s *ConfigDownloadScenario) executeStep() {
	s.mu.RLock()
	stepIdx := s.stepIdx
	sendFn := s.sendFn
	s.mu.RUnlock()

	if stepIdx >= len(s.steps) {
		s.Stop()
		return
	}

	step := s.steps[stepIdx]

	s.mu.Lock()
	s.result.StepIndex = stepIdx
	s.result.StepName = step.Name
	s.result.Progress = (stepIdx * 100) / len(s.steps)
	s.mu.Unlock()

	s.logger.Infof("[scenario:%s] executing step %d/%d: %s", s.sessionID, stepIdx+1, len(s.steps), step.Name)

	// 构建并发送消息
	var sendMsg types.Message
	switch step.FuncCode {
	case types.FuncConfigDownload:
		sendMsg = &msg.ConfigDownloadMsg{}
	case types.FuncBillingRules:
		sendMsg = &msg.BillingRulesDownload{}
	case types.FuncDeviceQuery:
		sendMsg = &msg.DeviceQueryDownload{}
	default:
		s.fail(fmt.Sprintf("unsupported funcCode 0x%02X", step.FuncCode))
		return
	}

	if err := sendFn(sendMsg); err != nil {
		s.fail(fmt.Sprintf("send 0x%02X failed: %v", step.FuncCode, err))
		return
	}

	s.logger.Infof("[scenario:%s] sent 0x%02X config download", s.sessionID, step.FuncCode)
}

// advanceStep 前进到下一步
func (s *ConfigDownloadScenario) advanceStep() {
	s.mu.Lock()
	s.stepIdx++
	s.mu.Unlock()

	s.executeStep()
}

// fail 标记场景失败
func (s *ConfigDownloadScenario) fail(errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = StateFailed
	s.result.State = StateFailed
	s.result.EndTime = time.Now()
	s.result.Duration = s.result.EndTime.Sub(s.result.StartTime)
	s.result.ErrorMsg = errMsg

	s.logger.Errorf("[scenario:%s] failed: %s", s.sessionID, errMsg)

	if s.stopCh != nil {
		close(s.stopCh)
	}
}
