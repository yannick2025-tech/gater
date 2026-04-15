package dispatcher

import (
	"fmt"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	"github.com/yannick2025-tech/nts-gater/internal/session"
	logging "github.com/yannick2025-tech/gwc-logging"
)

// Handler 业务处理器接口
type Handler interface {
	Handle(ctx *Context) error
}

// HandlerFunc 处理函数类型
type HandlerFunc func(ctx *Context) error

func (f HandlerFunc) Handle(ctx *Context) error {
	return f(ctx)
}

// ReplyFunc 回复函数类型
type ReplyFunc func(header types.MessageHeader, data []byte) error

// Context 分发上下文
type Context struct {
	PostNo   uint32
	Charger  byte
	FuncCode byte
	Dir      types.Direction
	Data     []byte       // 已解密的消息体
	Message  types.Message // 已解码的消息
	Session  *session.Session
	Logger   logging.Logger
	Reply    ReplyFunc    // 发送回复
	Proto    types.Protocol
}

// ReplyMessage 便捷回复：编码消息并发送
func (c *Context) ReplyMessage(msg types.Message) error {
	data, err := msg.Encode()
	if err != nil {
		return fmt.Errorf("encode reply message failed: %w", err)
	}

	spec := msg.Spec()
	header := types.MessageHeader{
		StartByte:  c.Proto.FrameConfig().StartByte,
		Version:    c.Proto.Version(),
		FuncCode:   spec.FuncCode,
		PostNo:     c.PostNo,
		Charger:    c.Charger,
		EncryptFlag: 0x00, // 默认不加密，handler可自行调整
	}
	if spec.Encrypt {
		header.EncryptFlag = 0x01
	}

	return c.Reply(header, data)
}

// Dispatcher 消息分发器
type Dispatcher struct {
	proto    types.Protocol
	sessMgr  *session.SessionManager
	logger   logging.Logger
	handlers map[dispatchKey]Handler
}

type dispatchKey struct {
	funcCode byte
	dir      types.Direction
}

// New 创建消息分发器
func New(proto types.Protocol, sessMgr *session.SessionManager, logger logging.Logger) *Dispatcher {
	return &Dispatcher{
		proto:    proto,
		sessMgr:  sessMgr,
		logger:   logger,
		handlers: make(map[dispatchKey]Handler),
	}
}

// Register 注册业务处理器
func (d *Dispatcher) Register(funcCode byte, dir types.Direction, handler Handler) {
	key := dispatchKey{funcCode: funcCode, dir: dir}
	d.handlers[key] = handler
}

// RegisterFunc 注册业务处理函数
func (d *Dispatcher) RegisterFunc(funcCode byte, dir types.Direction, fn HandlerFunc) {
	d.Register(funcCode, dir, fn)
}

// Dispatch 分发消息到对应的处理器
func (d *Dispatcher) Dispatch(ctx *Context) error {
	key := dispatchKey{funcCode: ctx.FuncCode, dir: ctx.Dir}
	handler, ok := d.handlers[key]
	if !ok {
		return fmt.Errorf("no handler for func=0x%02X dir=%v", ctx.FuncCode, ctx.Dir)
	}
	return handler.Handle(ctx)
}

// HasHandler 检查是否有注册的处理器
func (d *Dispatcher) HasHandler(funcCode byte, dir types.Direction) bool {
	_, ok := d.handlers[dispatchKey{funcCode: funcCode, dir: dir}]
	return ok
}
