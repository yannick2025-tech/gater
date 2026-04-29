// Package session provides charging station session management.
package session

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/crypto"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	"github.com/yannick2025-tech/nts-gater/internal/recorder"
)

// AuthState 认证状态
type AuthState int

const (
	AuthNone   AuthState = iota // 未认证
	AuthPending                 // 认证中（已发0x0A，等待0x0B）
	Authenticated               // 已认证
)

// KeyState 密钥状态
type KeyState int

const (
	KeyFixed    KeyState = iota // 使用固定密钥
	KeySession                  // 使用会话密钥
)

func (s AuthState) String() string {
	switch s {
	case AuthNone:
		return "none"
	case AuthPending:
		return "pending"
	case Authenticated:
		return "authenticated"
	default:
		return "unknown"
	}
}

func (s KeyState) String() string {
	switch s {
	case KeyFixed:
		return "fixed"
	case KeySession:
		return "session"
	default:
		return "unknown"
	}
}

// Session 充电桩会话
type Session struct {
	ID            string
	PostNo        uint32
	ConnID        string
	AuthState     AuthState
	KeyState      KeyState
	Connected     bool                     // TCP连接是否在线
	FixedCipher   *crypto.AESCBCCipher // 固定密钥加密器
	SessionCipher *crypto.AESCBCCipher // 会话密钥加密器
	RandomKey     []byte               // 13位随机密钥（0x0A下发）
	Recorder      *recorder.SessionRecorder // 消息记录器
	CreatedAt     time.Time
	LastActive    time.Time
	Prices        []PriceConfig            // 时段费率配置（WEB端传入）

	// 充电状态追踪
	ChargingState *ChargingState           // 充电过程状态（0x03/0x06/0x05更新）
	SentPeakTypes  []byte                  // 0x04下发的峰谷类型列表（按时段顺序），供0x06校验用

	mu sync.RWMutex
}

// PriceConfig 时段费率配置
type PriceConfig struct {
	StartTime      string  // "HH:mm"
	EndTime        string  // "HH:mm"
	ElectricityFee float64 // 电费（元/kWh）
	ServiceFee     float64 // 服务费（元/kWh）
	PeakValleyType byte    // 峰谷类型: 1峰2尖3谷4平
}

// ChargingState 充电过程状态
type ChargingState struct {
	PlatformStartTime   time.Time // 平台充电开始时间（下发0x03时间点）
	FirstDataTime       time.Time // 收到第1个0x06时间
	LastDataTime        time.Time // 收到最后0x06时间（每次更新）
	PlatformStopTime    time.Time // 平台下发0x08时间
	ChargerStartTime    string    // 充电桩充电开始时间（从0x05的chargeStartTime取）
	ChargerStopTime     string    // 充电桩充电结束时间（从0x05的chargeStopTime取）
	ChargingOrderNo     string    // 充电订单号（从0x06取）
	CurrentElec         float64   // 当前电量（0x06的currentElec，放大10000倍前的原始值）
	LastElec            float64   // 上一次电量（用于校验递增）
	CurrentSOC          byte      // 当前SOC（0x06的currentSoc）
	StopSOC             byte      // 结束SOC（0x05的stopSoc）
	ChargingDataCount   int       // 收到0x06次数
	DataRecords         []ChargingDataRecord // 0x06分时段累计信息快照
	ValidationResults   []ValidationResult   // 校验结果
	IsChargingStopped   bool      // 充电是否已结束（收到0x05后设为true）
}

// ChargingDataRecord 0x06充电数据快照
type ChargingDataRecord struct {
	Timestamp time.Time
	Elec      float64
	SOC       byte
	Data      interface{} // 原始0x06消息ToJSONMap
}

// ValidationResult 校验结果
type ValidationResult struct {
	Timestamp time.Time
	FuncCode  byte
	Rule      string // 校验规则描述
	Passed    bool
	Message   string // 详细信息/告警内容
}

// GetEncryptFn 获取当前加密函数
func (s *Session) GetEncryptFn() func([]byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.KeyState == KeySession && s.SessionCipher != nil {
		return s.SessionCipher.Encrypt
	}
	return s.FixedCipher.Encrypt
}

// GetDecryptFn 获取当前解密函数
func (s *Session) GetDecryptFn() func([]byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.KeyState == KeySession && s.SessionCipher != nil {
		return s.SessionCipher.Decrypt
	}
	return s.FixedCipher.Decrypt
}

// SetAuthState 更新认证状态
func (s *Session) SetAuthState(state AuthState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AuthState = state
}

// GetAuthState 获取认证状态
func (s *Session) GetAuthState() AuthState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AuthState
}

// SetConnected 设置连接状态
func (s *Session) SetConnected(connected bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Connected = connected
}

// IsConnected 获取连接状态
func (s *Session) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Connected
}

// SetPrices 设置时段费率配置（WEB端传入）
func (s *Session) SetPrices(prices []PriceConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Prices = prices
}

// GetPrices 获取时段费率配置
func (s *Session) GetPrices() []PriceConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Prices
}

// SetSentPeakTypes 存储0x04下发的峰谷类型（供0x06校验用）
func (s *Session) SetSentPeakTypes(types []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SentPeakTypes = types
}

// GetSentPeakTypes 获取0x04下发的峰谷类型
func (s *Session) GetSentPeakTypes() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SentPeakTypes
}

// GetChargingState 获取充电状态（线程安全副本）
func (s *Session) GetChargingState() *ChargingState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ChargingState == nil {
		return nil
	}
	cp := *s.ChargingState
	return &cp
}

// InitChargingState 初始化充电状态（0x03下发时调用）
func (s *Session) InitChargingState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ChargingState == nil {
		s.ChargingState = &ChargingState{}
	}
}

// SetPlatformStartTime 设置平台充电开始时间
func (s *Session) SetPlatformStartTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ChargingState == nil {
		s.ChargingState = &ChargingState{}
	}
	s.ChargingState.PlatformStartTime = t
}

// SetPlatformStopTime 设置平台充电结束时间
func (s *Session) SetPlatformStopTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ChargingState == nil {
		s.ChargingState = &ChargingState{}
	}
	s.ChargingState.PlatformStopTime = t
}

// UpdateChargingData 更新0x06充电数据
func (s *Session) UpdateChargingData(elec float64, soc byte, orderNo string, data interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ChargingState == nil {
		s.ChargingState = &ChargingState{}
	}
	cs := s.ChargingState
	cs.ChargingDataCount++
	cs.LastElec = cs.CurrentElec
	cs.CurrentElec = elec
	cs.CurrentSOC = soc
	cs.ChargingOrderNo = orderNo
	cs.LastDataTime = time.Now()
	if cs.FirstDataTime.IsZero() {
		cs.FirstDataTime = cs.LastDataTime
	}
	cs.DataRecords = append(cs.DataRecords, ChargingDataRecord{
		Timestamp: cs.LastDataTime,
		Elec:      elec,
		SOC:       soc,
		Data:      data,
	})
}

// SetChargingStopped 设置充电结束（0x05收到后调用）
func (s *Session) SetChargingStopped(startTime, stopTime string, stopSoc byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ChargingState == nil {
		s.ChargingState = &ChargingState{}
	}
	s.ChargingState.ChargerStartTime = startTime
	s.ChargingState.ChargerStopTime = stopTime
	s.ChargingState.StopSOC = stopSoc
	s.ChargingState.IsChargingStopped = true
}

// AddValidationResult 添加校验结果
func (s *Session) AddValidationResult(funcCode byte, rule string, passed bool, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ChargingState == nil {
		s.ChargingState = &ChargingState{}
	}
	s.ChargingState.ValidationResults = append(s.ChargingState.ValidationResults, ValidationResult{
		Timestamp: time.Now(),
		FuncCode:  funcCode,
		Rule:      rule,
		Passed:    passed,
		Message:   message,
	})
}

// SetRandomKey 设置13位随机密钥（0x0A认证流程）
func (s *Session) SetRandomKey(key []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RandomKey = key
}

// GetRandomKey 获取13位随机密钥
func (s *Session) GetRandomKey() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.RandomKey
}

// SetKeyState 切换密钥状态
func (s *Session) SetKeyState(state KeyState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.KeyState = state
}

// UpdateActive 更新最后活跃时间
func (s *Session) UpdateActive() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActive = time.Now()
}

// SessionManager 会话管理器
type SessionManager struct {
	sessions   map[string]*Session // sessionID -> Session
	postNoMap  map[uint32]string   // postNo -> sessionID
	fixedKey   string
	ivRule     string
	mu         sync.RWMutex
	counter    uint64
	heartbeatTimeout time.Duration

	// DB 持久化回调（由 main.go 注入，避免循环依赖）
	onCreate  func(*Session) // session 创建时回调
	onRemove  func(string)   // session 移除时回调（传入 sessionID）
}

// NewManager 创建会话管理器
func NewManager(proto types.Protocol, heartbeatTimeout time.Duration) *SessionManager {
	return &SessionManager{
		sessions:         make(map[string]*Session),
		postNoMap:        make(map[uint32]string),
		fixedKey:         proto.CryptoConfig().FixedKey,
		ivRule:           proto.CryptoConfig().IVRule,
		heartbeatTimeout: heartbeatTimeout,
	}
}

// SetOnCreate 设置 session 创建时的回调
func (m *SessionManager) SetOnCreate(fn func(*Session)) {
	m.onCreate = fn
}

// SetOnRemove 设置 session 移除时的回调
func (m *SessionManager) SetOnRemove(fn func(string)) {
	m.onRemove = fn
}

// Create 创建会话（充电桩新连接时调用）
// 如果该桩号已有活跃会话，返回 ErrDuplicatePostNo
func (m *SessionManager) Create(postNo uint32, connID string) (*Session, error) {
	// 拒绝重复桩号连接
	m.mu.RLock()
	if _, exists := m.postNoMap[postNo]; exists {
		m.mu.RUnlock()
		return nil, fmt.Errorf("postNo %d already has an active session, connection rejected", postNo)
	}
	m.mu.RUnlock()

	fixedCipher, err := crypto.NewAESCBCCipher(m.fixedKey, m.ivRule)
	if err != nil {
		return nil, fmt.Errorf("create fixed cipher: %w", err)
	}

	atomic.AddUint64(&m.counter, 1)
	id := strings.ToUpper(strings.ReplaceAll(uuid.New().String(), "", "")[:16]) // 取前16位大写作为短UUID，如 A1B2C3D4E5F6G7H8

	now := time.Now()
	sess := &Session{
		ID:          id,
		PostNo:      postNo,
		ConnID:      connID,
		AuthState:   AuthNone,
		KeyState:    KeyFixed,
		Connected:   true, // 新建会话时连接在线
		FixedCipher: fixedCipher,
		Recorder:    recorder.NewSessionRecorder(id, postNo),
		CreatedAt:   now,
		LastActive:  now,
	}

	m.mu.Lock()
	m.sessions[id] = sess
	m.postNoMap[postNo] = id
	m.mu.Unlock()

	// 触发创建回调（异步 DB 持久化）
	if m.onCreate != nil {
		go m.onCreate(sess)
	}

	return sess, nil
}

// Get 通过sessionID获取会话
func (m *SessionManager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[id]
	return sess, ok
}

// GetByPostNo 通过桩编号获取会话
func (m *SessionManager) GetByPostNo(postNo uint32) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.postNoMap[postNo]
	if !ok {
		return nil, false
	}
	sess, ok := m.sessions[id]
	return sess, ok
}

// Remove 移除会话并关闭记录器
func (m *SessionManager) Remove(id string) {
	m.mu.Lock()
	var sessionID string
	if sess, ok := m.sessions[id]; ok {
		if sess.Recorder != nil {
			sess.Recorder.Close()
		}
		delete(m.postNoMap, sess.PostNo)
		delete(m.sessions, id)
		sessionID = id
	}
	m.mu.Unlock()

	// 触发移除回调（异步 DB 更新在线状态）
	if sessionID != "" && m.onRemove != nil {
		go m.onRemove(sessionID)
	}
}

// Count 返回会话数量
func (m *SessionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// GetAllSessions 返回所有活跃会话的副本（用于会话列表展示）
func (m *SessionManager) GetAllSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Session, 0, len(m.sessions))
	for _, sess := range m.sessions {
		result = append(result, sess)
	}
	return result
}

// SetSessionKey 设置会话密钥（0x21密钥更新成功后调用）
func (m *SessionManager) SetSessionKey(sessionID string, keyStr string) error {
	m.mu.RLock()
	sess, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	sessionCipher, err := crypto.NewAESCBCCipher(keyStr, m.ivRule)
	if err != nil {
		return fmt.Errorf("create session cipher: %w", err)
	}

	sess.mu.Lock()
	sess.SessionCipher = sessionCipher
	sess.KeyState = KeySession
	sess.mu.Unlock()

	return nil
}

// FindHeartbeatTimeout 查找心跳超时的会话
func (m *SessionManager) FindHeartbeatTimeout() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var expired []*Session
	for _, sess := range m.sessions {
		if time.Since(sess.LastActive) > m.heartbeatTimeout {
			expired = append(expired, sess)
		}
	}
	return expired
}
