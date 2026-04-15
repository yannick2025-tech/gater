// Package session provides charging station session management.
package session

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

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
	FixedCipher   *crypto.AESCBCCipher // 固定密钥加密器
	SessionCipher *crypto.AESCBCCipher // 会话密钥加密器
	RandomKey     []byte               // 13位随机密钥（0x0A下发）
	Recorder      *recorder.SessionRecorder // 消息记录器
	CreatedAt     time.Time
	LastActive    time.Time

	mu sync.RWMutex
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
	id := fmt.Sprintf("sess-%d", m.counter)

	now := time.Now()
	sess := &Session{
		ID:          id,
		PostNo:      postNo,
		ConnID:      connID,
		AuthState:   AuthNone,
		KeyState:    KeyFixed,
		FixedCipher: fixedCipher,
		Recorder:    recorder.NewSessionRecorder(id, postNo),
		CreatedAt:   now,
		LastActive:  now,
	}

	m.mu.Lock()
	m.sessions[id] = sess
	m.postNoMap[postNo] = id
	m.mu.Unlock()

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
	defer m.mu.Unlock()
	if sess, ok := m.sessions[id]; ok {
		if sess.Recorder != nil {
			sess.Recorder.Close()
		}
		delete(m.postNoMap, sess.PostNo)
		delete(m.sessions, id)
	}
}

// Count 返回会话数量
func (m *SessionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
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
