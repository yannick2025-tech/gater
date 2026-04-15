package scenario

import (
	"fmt"
	"sync"
	"time"

	"github.com/yannick2025-tech/nts-gater/internal/generator"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard/msg"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	"github.com/yannick2025-tech/nts-gater/internal/session"
	logging "github.com/yannick2025-tech/gwc-logging"
)

// ==================== 基础充电测试场景 ====================
//
// 流程（参考 service-flow.MD 的 APP启动充电流程）：
// 1. 平台下发 0x03 启动充电 → 等待桩回复
// 2. 桩上报 0x04 启动充电请求 → 平台回复0x04（含计费策略）
// 3. 桩上报 0x10 静态BMS数据 → 平台回复
// 4. 桩周期上报 0x06 充电过程数据 → 平台回复（持续N次）
// 5. 桩上报 0x24 充电中BMS数据 → 平台回复
// 6. 平台下发 0x08 停止充电 → 等待桩回复
// 7. 桩上报 0x05 停止充电 → 平台回复
//
// 注意：0x03/0x08 由平台主动下发，0x04/0x05/0x06/0x10/0x24 由桩上报。
// 在测试场景中，桩的消息由真实充电桩发送（或模拟器），
// 场景引擎只负责：①下发启动报文 ②等待桩回复 ③下发停止报文 ④等待桩回复

// BasicChargingStep 基础充电步骤
type BasicChargingStep struct {
	Name      string
	FuncCode  byte // 触发功能码（0=平台主动触发，非0=等待桩上报）
	Direction types.Direction
}

// 基础充电测试步骤定义
var basicChargingSteps = []BasicChargingStep{
	{Name: "发送启动充电请求(0x03)", FuncCode: types.FuncPlatformStart, Direction: types.DirectionDownload},
	{Name: "等待桩启动确认(0x04)", FuncCode: types.FuncChargerStart, Direction: types.DirectionUpload},
	{Name: "充电进行中(0x06)", FuncCode: types.FuncChargingData, Direction: types.DirectionUpload},
	{Name: "发送停止充电请求(0x08)", FuncCode: types.FuncPlatformStop, Direction: types.DirectionDownload},
	{Name: "等待桩停止确认(0x05)", FuncCode: types.FuncChargerStop, Direction: types.DirectionUpload},
}

// BasicChargingScenario 基础充电测试场景
type BasicChargingScenario struct {
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

	// 充电测试参数
	chargingDuration time.Duration // 充电持续时间（收到第一个0x06后等待多久发送0x08）
	chargingStart    time.Time     // 充电开始时间
	maxChargingData  int           // 最多接收0x06次数
	chargingDataCnt  int           // 已接收0x06次数
}

// NewBasicChargingScenario 创建基础充电测试场景
func NewBasicChargingScenario(sessionID string, sess *session.Session, proto types.Protocol, logger logging.Logger) *BasicChargingScenario {
	return &BasicChargingScenario{
		sessionID:        sessionID,
		sess:             sess,
		proto:            proto,
		logger:           logger,
		state:            StateIdle,
		chargingDuration: 2 * time.Minute, // 默认充电2分钟
		maxChargingData:  4,               // 最多接收4次0x06
	}
}

// Name 场景名称
func (s *BasicChargingScenario) Name() string { return "basic_charging" }

// State 获取当前状态
func (s *BasicChargingScenario) State() ScenarioState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Result 获取执行结果
func (s *BasicChargingScenario) Result() *ScenarioResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := s.result
	return &r
}

// SetSendFunc 设置发送函数
func (s *BasicChargingScenario) SetSendFunc(fn SendFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sendFn = fn
}

// Start 启动场景
func (s *BasicChargingScenario) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateRunning {
		return ErrScenarioAlreadyRunning
	}
	if s.sendFn == nil {
		return ErrSendNotSet
	}

	s.state = StateRunning
	s.stepIdx = 0
	s.chargingDataCnt = 0
	s.stopCh = make(chan struct{})
	s.result = ScenarioResult{
		ScenarioName: s.Name(),
		State:        StateRunning,
		StartTime:    time.Now(),
		StepTotal:    len(basicChargingSteps),
	}

	// 立即执行第一步：下发 0x03 启动充电
	go s.executeStep()

	return nil
}

// Stop 停止场景
func (s *BasicChargingScenario) Stop() {
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
func (s *BasicChargingScenario) OnMessage(funcCode byte, dir types.Direction, m types.Message) {
	s.mu.RLock()
	running := s.state == StateRunning
	stepIdx := s.stepIdx
	s.mu.RUnlock()

	if !running {
		return
	}

	// 获取当前步骤期望的消息
	if stepIdx >= len(basicChargingSteps) {
		return
	}

	expectedStep := basicChargingSteps[stepIdx]

	// 如果当前步骤是等待桩上报的消息
	if expectedStep.FuncCode != 0 && expectedStep.Direction == types.DirectionUpload {
		if funcCode == expectedStep.FuncCode && dir == expectedStep.Direction {
			s.logger.Infof("[scenario:%s] received expected message: step=%s func=0x%02X",
				s.sessionID, expectedStep.Name, funcCode)
			s.advanceStep()
		}
	}

	// 特殊处理：充电过程中持续接收0x06
	if funcCode == types.FuncChargingData && dir == types.DirectionUpload && stepIdx == 2 {
		s.mu.Lock()
		s.chargingDataCnt++
		cnt := s.chargingDataCnt
		s.mu.Unlock()

		// 达到最大次数或超时后，主动下发0x08
		if cnt >= s.maxChargingData {
			s.logger.Infof("[scenario:%s] charging data count reached %d, sending stop", s.sessionID, cnt)
			s.advanceStep() // 进入步骤3：发送0x08
		}
	}
}

// executeStep 执行当前步骤
func (s *BasicChargingScenario) executeStep() {
	s.mu.RLock()
	stepIdx := s.stepIdx
	sendFn := s.sendFn
	s.mu.RUnlock()

	if stepIdx >= len(basicChargingSteps) {
		s.Stop()
		return
	}

	step := basicChargingSteps[stepIdx]

	s.mu.Lock()
	s.result.StepIndex = stepIdx
	s.result.StepName = step.Name
	s.result.Progress = (stepIdx * 100) / len(basicChargingSteps)
	s.mu.Unlock()

	s.logger.Infof("[scenario:%s] executing step %d/%d: %s", s.sessionID, stepIdx+1, len(basicChargingSteps), step.Name)

	// 步骤0：平台下发0x03启动充电
	if step.FuncCode == types.FuncPlatformStart && step.Direction == types.DirectionDownload {
		startMsg := &msg.PlatformStartDownload{
			StartupType:          6, // 远程鉴权-命令
			AuthenticationNumber: fmt.Sprintf("SCENARIO-%s-%d", s.sessionID, time.Now().Unix()),
		}
		if err := sendFn(startMsg); err != nil {
			s.fail(fmt.Sprintf("send 0x03 failed: %v", err))
			return
		}
		s.logger.Infof("[scenario:%s] sent 0x03 platform start charge", s.sessionID)
		// 等待桩的0x04回复
	}

	// 步骤3：平台下发0x08停止充电
	if step.FuncCode == types.FuncPlatformStop && step.Direction == types.DirectionDownload {
		stopMsg := &msg.PlatformStopDownload{
			PlatformOrderNumber:       generator.GenerateOrderNo(),
			PileChargingFailureReason: 0x00, // 正常停止
		}
		if err := sendFn(stopMsg); err != nil {
			s.fail(fmt.Sprintf("send 0x08 failed: %v", err))
			return
		}
		s.logger.Infof("[scenario:%s] sent 0x08 platform stop charge", s.sessionID)
		// 等待桩的0x05回复
	}
}

// advanceStep 前进到下一步
func (s *BasicChargingScenario) advanceStep() {
	s.mu.Lock()
	s.stepIdx++
	s.mu.Unlock()

	s.executeStep()
}

// fail 标记场景失败
func (s *BasicChargingScenario) fail(errMsg string) {
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
