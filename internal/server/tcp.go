// Package server provides TCP server for charging station connections.
package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/yannick2025-tech/nts-gater/internal/config"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/codec"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	logging "github.com/yannick2025-tech/gwc-logging"
)

// MessageHandler 消息处理回调
type MessageHandler func(conn *Connection, header types.MessageHeader, data []byte, rawFrame []byte)

// Connection TCP连接
type Connection struct {
	ID           string
	RemoteAddr   net.Addr
	PostNo       uint32 // 充电桩编号
	Charger      byte   // 枪号
	Conn         net.Conn
	LastActive   time.Time
	Protocol     types.Protocol
	Scanner      *codec.FrameScanner
	Encoder      *codec.FrameEncoder
	Decoder      *codec.FrameDecoder
	mu           sync.Mutex
	sendBuf      []byte
	closed       bool // 防止重复关闭
}

// Send 发送消息（线程安全）
func (c *Connection) Send(header types.MessageHeader, data []byte, encryptFn func([]byte) ([]byte, error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	frame, err := c.Encoder.Encode(header, data, encryptFn)
	if err != nil {
		return fmt.Errorf("encode frame failed: %w", err)
	}

	_, err = c.Conn.Write(frame)
	if err != nil {
		return fmt.Errorf("write conn failed: %w", err)
	}

	c.LastActive = time.Now()
	return nil
}

// SendFrame 发送已编码的完整帧（线程安全）
func (c *Connection) SendFrame(frame []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.Conn.Write(frame)
	if err != nil {
		return fmt.Errorf("write conn failed: %w", err)
	}

	c.LastActive = time.Now()
	return nil
}

// Close 关闭连接（幂等，重复调用安全）
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil // 已关闭，跳过
	}
	c.closed = true
	// 设置读截止时间为过去，加速 handleConnection 的 Read() 返回 error
	if c.Conn != nil {
		c.Conn.SetReadDeadline(time.Now().Add(-1 * time.Second))
		return c.Conn.Close()
	}
	return nil
}

// DisconnectHandler 断开连接回调（用于生成报告等）
type DisconnectHandler func(conn *Connection, postNo uint32)

// Server TCP服务器
type Server struct {
	cfg         *config.Config
	proto       types.Protocol
	logger      logging.Logger
	handler     MessageHandler
	onDisconnect DisconnectHandler
	listener    net.Listener
	mu          sync.RWMutex
	connections map[string]*Connection
	connCount   uint64
}

// New 创建TCP服务器
func New(cfg *config.Config, proto types.Protocol, logger logging.Logger) *Server {
	return &Server{
		cfg:         cfg,
		proto:       proto,
		logger:      logger,
		connections: make(map[string]*Connection),
	}
}

// OnMessage 设置消息处理回调
func (s *Server) OnMessage(handler MessageHandler) {
	s.handler = handler
}

// OnDisconnect 设置断开连接回调
func (s *Server) OnDisconnect(handler DisconnectHandler) {
	s.onDisconnect = handler
}

// Start 启动TCP服务器
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s failed: %w", addr, err)
	}
	s.listener = ln
	s.logger.Infof("TCP server listening on %s", addr)

	go s.acceptLoop(ctx)
	return nil
}

// Stop 停止TCP服务器
func (s *Server) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
	s.mu.Lock()
	for id, conn := range s.connections {
		conn.Close()
		delete(s.connections, id)
	}
	s.mu.Unlock()
	s.logger.Info("TCP server stopped")
}

// ConnectionCount 返回当前连接数
func (s *Server) ConnectionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections)
}

// acceptLoop 接受连接循环
func (s *Server) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				s.logger.Errorf("accept connection failed: %v", err)
				continue
			}
		}

		s.connCount++
		c := &Connection{
			ID:       fmt.Sprintf("conn-%d", s.connCount),
			RemoteAddr: conn.RemoteAddr(),
			Conn:     conn,
			LastActive: time.Now(),
			Protocol: s.proto,
			Scanner:  codec.NewFrameScanner(),
			Encoder:  codec.NewFrameEncoder(s.proto),
			Decoder:  codec.NewFrameDecoder(s.proto),
		}

		s.mu.Lock()
		s.connections[c.ID] = c
		s.mu.Unlock()

		s.logger.Infof("[%s] new connection from %s, total=%d", c.ID, conn.RemoteAddr(), s.ConnectionCount())

		go s.handleConnection(ctx, c)
	}
}

// handleConnection 处理单个连接
func (s *Server) handleConnection(ctx context.Context, conn *Connection) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Errorf("[%s] panic in handleConnection: %v", conn.ID, r)
		}
		postNo := conn.PostNo
		conn.Close()
		s.mu.Lock()
		delete(s.connections, conn.ID)
		s.mu.Unlock()
		s.logger.Infof("[%s] disconnected, postNo=%d, total=%d", conn.ID, postNo, s.ConnectionCount())

		// 断开连接回调（生成报告、移除会话等）
		if s.onDisconnect != nil && postNo > 0 {
			s.onDisconnect(conn, postNo)
		}
	}()

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn.Conn.SetReadDeadline(time.Now().Add(s.cfg.Server.ReadTimeout))
		n, err := conn.Conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 读超时，检查是否心跳超时
				if time.Since(conn.LastActive) > s.cfg.Server.HeartbeatTimeout {
					s.logger.Warnf("[%s] heartbeat timeout, closing", conn.ID)
					return
				}
				continue
			}
			return // 连接断开或其他错误
		}

		if n == 0 {
			continue
		}

		conn.LastActive = time.Now()
		conn.Scanner.Feed(buf[:n])

		// 提取所有完整帧
		for {
			frame := conn.Scanner.Next()
			if frame == nil {
				break
			}

			// 解码帧
			result, err := conn.Decoder.Decode(frame, nil) // TODO: 接入加密
			if err != nil {
				s.logger.Warnf("[%s] decode frame failed: %v, hex=% X", conn.ID, err, frame)
				continue
			}

			s.logger.Debugf("[%s] recv func=0x%02X postNo=%d charger=%d len=%d hex=% X",
				conn.ID, result.Header.FuncCode, result.Header.PostNo,
				result.Header.Charger, len(result.DecryptedData), frame)

			// 更新连接的桩编号和枪号
			conn.mu.Lock()
			conn.PostNo = result.Header.PostNo
			conn.Charger = result.Header.Charger
			conn.mu.Unlock()

			// 回调处理
			if s.handler != nil {
				s.handler(conn, result.Header, result.DecryptedData, frame)
			}
		}
	}
}

// Broadcast 广播消息到所有连接
func (s *Server) Broadcast(header types.MessageHeader, data []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, conn := range s.connections {
		if err := conn.Send(header, data, nil); err != nil {
			s.logger.Errorf("[%s] broadcast failed: %v", conn.ID, err)
		}
	}
}

// GetConnection 通过ID获取连接
func (s *Server) GetConnection(id string) (*Connection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conn, ok := s.connections[id]
	return conn, ok
}

// FindConnectionByPostNo 通过桩编号查找连接
func (s *Server) FindConnectionByPostNo(postNo uint32) (*Connection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, conn := range s.connections {
		if conn.PostNo == postNo {
			return conn, true
		}
	}
	return nil, false
}
