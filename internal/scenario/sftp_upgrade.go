// Package scenario provides the SFTP upgrade test scenario.
package scenario

import (
	"fmt"
	"sync"
	"time"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard/msg"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	"github.com/yannick2025-tech/nts-gater/internal/session"
	logging "github.com/yannick2025-tech/gwc-logging"
)

// ==================== SFTP升级测试场景 ====================
//
// 流程：
// 1. 平台下发 0xF6 SFTP升级请求 → 等待桩回复
// 2. 桩回复 0xF6 结果（00成功/01失败/02占用中）
// 3. 桩上报 0xF7 升级进度 → 可能多次
// 4. 升级完成（进度=03安装完成）

// SFTPUpgradeStep SFTP升级步骤
type SFTPUpgradeStep struct {
	Name      string
	FuncCode  byte
	Direction types.Direction
}

var sftpUpgradeSteps = []SFTPUpgradeStep{
	{Name: "发送SFTP升级请求(0xF6)", FuncCode: types.FuncSFTPUpgrade, Direction: types.DirectionDownload},
	{Name: "等待升级回复(0xF6)", FuncCode: types.FuncSFTPUpgrade, Direction: types.DirectionReply},
	{Name: "等待升级进度(0xF7)", FuncCode: types.FuncUpgradeProgress, Direction: types.DirectionUpload},
}

// SFTPUpgradeScenario SFTP升级测试场景
type SFTPUpgradeScenario struct {
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

	// SFTP升级参数
	sftpAddress  string
	sftpPort     uint16
	sftpUser     string
	sftpPassword string
	sftpFilePath string
	upgradeSeq   byte
}

// SFTPUpgradeConfig SFTP升级配置
type SFTPUpgradeConfig struct {
	Address  string `json:"address"`
	Port     uint16 `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	FilePath string `json:"filePath"`
}

// NewSFTPUpgradeScenario 创建SFTP升级测试场景
func NewSFTPUpgradeScenario(sessionID string, sess *session.Session, proto types.Protocol, logger logging.Logger) *SFTPUpgradeScenario {
	return &SFTPUpgradeScenario{
		sessionID:    sessionID,
		sess:         sess,
		proto:        proto,
		logger:       logger,
		state:        StateIdle,
		sftpAddress:  "192.168.1.100",
		sftpPort:     22,
		sftpUser:     "upgrade",
		sftpPassword: "upgrade123",
		sftpFilePath: "/firmware/update.bin",
		upgradeSeq:   1,
	}
}

// Name 场景名称
func (s *SFTPUpgradeScenario) Name() string { return "sftp_upgrade" }

// State 获取当前状态
func (s *SFTPUpgradeScenario) State() ScenarioState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Result 获取执行结果
func (s *SFTPUpgradeScenario) Result() *ScenarioResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := s.result
	return &r
}

// SetSendFunc 设置发送函数
func (s *SFTPUpgradeScenario) SetSendFunc(fn SendFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sendFn = fn
}

// Start 启动场景
func (s *SFTPUpgradeScenario) Start() error {
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
	s.stopCh = make(chan struct{})
	s.result = ScenarioResult{
		ScenarioName: s.Name(),
		State:        StateRunning,
		StartTime:    time.Now(),
		StepTotal:    len(sftpUpgradeSteps),
	}

	// 立即执行第一步：下发0xF6
	go s.executeStep()

	return nil
}

// Stop 停止场景
func (s *SFTPUpgradeScenario) Stop() {
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
func (s *SFTPUpgradeScenario) OnMessage(funcCode byte, dir types.Direction, m types.Message) {
	s.mu.RLock()
	running := s.state == StateRunning
	stepIdx := s.stepIdx
	s.mu.RUnlock()

	if !running {
		return
	}

	if stepIdx >= len(sftpUpgradeSteps) {
		return
	}

	expectedStep := sftpUpgradeSteps[stepIdx]

	// 步骤1：等待0xF6回复
	if expectedStep.FuncCode == types.FuncSFTPUpgrade && expectedStep.Direction == types.DirectionReply {
		if funcCode == types.FuncSFTPUpgrade && dir == types.DirectionReply {
			reply, ok := m.(*msg.SFTPUpgradeReply)
			if ok {
				s.logger.Infof("[scenario:%s] SFTP upgrade reply: result=%d", s.sessionID, reply.Result)
				if reply.Result != 0x00 {
					s.fail(fmt.Sprintf("upgrade rejected: result=%d", reply.Result))
					return
				}
			}
			s.advanceStep()
		}
	}

	// 步骤2：等待0xF7升级进度
	if expectedStep.FuncCode == types.FuncUpgradeProgress && expectedStep.Direction == types.DirectionUpload {
		if funcCode == types.FuncUpgradeProgress && dir == types.DirectionUpload {
			progress, ok := m.(*msg.UpgradeProgressUpload)
			if ok {
				s.logger.Infof("[scenario:%s] upgrade progress: seq=%d rate=%d",
					s.sessionID, progress.Seq, progress.ProgressRate)

				// 03=安装完成
				if progress.ProgressRate == 0x03 {
					s.logger.Infof("[scenario:%s] SFTP upgrade completed", s.sessionID)
					s.Stop()
				}
			}
		}
	}
}

// executeStep 执行当前步骤
func (s *SFTPUpgradeScenario) executeStep() {
	s.mu.RLock()
	stepIdx := s.stepIdx
	sendFn := s.sendFn
	s.mu.RUnlock()

	if stepIdx >= len(sftpUpgradeSteps) {
		s.Stop()
		return
	}

	step := sftpUpgradeSteps[stepIdx]

	s.mu.Lock()
	s.result.StepIndex = stepIdx
	s.result.StepName = step.Name
	s.result.Progress = (stepIdx * 100) / len(sftpUpgradeSteps)
	s.mu.Unlock()

	s.logger.Infof("[scenario:%s] executing step %d/%d: %s", s.sessionID, stepIdx+1, len(sftpUpgradeSteps), step.Name)

	// 步骤0：下发0xF6 SFTP升级请求
	if step.FuncCode == types.FuncSFTPUpgrade && step.Direction == types.DirectionDownload {
		upgradeMsg := &msg.SFTPUpgradeDownload{
			Seq:         s.upgradeSeq,
			UpgradeType: 0,   // 升级
			PackageType: 1,   // 更新包
			Version:     2,   // 版本号
			Address:     s.sftpAddress,
			Port:        s.sftpPort,
			UserName:    s.sftpUser,
			Password:    s.sftpPassword,
			FilePath:    s.sftpFilePath,
			CrcCode:     0, // TODO: 计算CRC
		}
		if err := sendFn(upgradeMsg); err != nil {
			s.fail(fmt.Sprintf("send 0xF6 failed: %v", err))
			return
		}
		s.logger.Infof("[scenario:%s] sent 0xF6 SFTP upgrade request", s.sessionID)
	}
}

// advanceStep 前进到下一步
func (s *SFTPUpgradeScenario) advanceStep() {
	s.mu.Lock()
	s.stepIdx++
	s.mu.Unlock()

	s.executeStep()
}

// fail 标记场景失败
func (s *SFTPUpgradeScenario) fail(errMsg string) {
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
