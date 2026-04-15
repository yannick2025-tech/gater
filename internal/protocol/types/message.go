// Package types provides core message and message spec interfaces.
package types

import "io"

// Direction 消息方向
type Direction int

const (
	DirectionUpload   Direction = iota // 充电桩→平台
	DirectionDownload                  // 平台→充电桩
	DirectionReply                     // 回复消息
)

// MessageSpec 消息规格 - 描述一个功能码某个方向的消息元数据
type MessageSpec struct {
	FuncCode  byte      // 功能码
	Direction Direction // 消息方向
	Name      string    // 消息名称
	Encrypt   bool      // 是否加密
	NeedReply bool      // 是否需要回复
}

// Message 消息接口 - 所有功能码消息必须实现
type Message interface {
	// Spec 消息规格
	Spec() MessageSpec
	// Decode 从二进制数据解码消息体（已解密后的数据域）
	Decode(data []byte) error
	// Encode 编码消息体为二进制数据（编码后用于加密前的数据域）
	Encode() ([]byte, error)
	// Validate 校验消息字段值是否合法
	Validate() []ValidationError
	// ToJSONMap 转为JSON友好的map（与示例报文JSON字段名一致）
	ToJSONMap() map[string]interface{}
}

// ValidationError 校验错误
type ValidationError struct {
	Field   string // 字段名
	Code    string // 错误码
	Message string // 错误描述
}

// MessageRegistry 消息注册表接口
type MessageRegistry interface {
	// Register 注册消息构造函数
	Register(funcCode byte, dir Direction, factory func() Message)
	// Create 创建消息实例
	Create(funcCode byte, dir Direction) (Message, bool)
	// Spec 查询消息规格
	Spec(funcCode byte, dir Direction) (MessageSpec, bool)
	// AllSpecs 返回所有已注册的消息规格
	AllSpecs() []MessageSpec
	// NeedReply 判断某方向消息是否需要回复
	NeedReply(funcCode byte, dir Direction) bool
	// ReplyDirection 返回回复方向
	ReplyDirection(dir Direction) Direction
}

// MessageHeader 消息帧头（公共部分，不含数据域）
// 当前主要以standard消息为主，后续如果有其他协议进来，可能需要扩展。
type MessageHeader struct {
	StartByte   byte   // 起始域 0x32
	Version     byte   // 版本号
	FuncCode    byte   // 功能码
	PostNo      uint32 // 充电桩编号
	Charger     byte   // 枪号
	EncryptFlag byte   // 加密标志
	Checksum    byte   // 校验码
	DataLength  uint16 // 数据域长度
}

// FullMessage 完整消息（帧头+消息体）
type FullMessage struct {
	Header MessageHeader
	Body   Message
}

// EncodeFrame 编码完整帧（帧头+加密后的数据域），返回完整的二进制帧
func (h *MessageHeader) EncodeFrame(encryptedData []byte) []byte {
	totalLen := 12 + len(encryptedData)
	buf := make([]byte, totalLen)

	buf[0] = h.StartByte
	buf[1] = h.Version
	buf[2] = h.FuncCode
	buf[3] = byte(h.PostNo)
	buf[4] = byte(h.PostNo >> 8)
	buf[5] = byte(h.PostNo >> 16)
	buf[6] = byte(h.PostNo >> 24)
	buf[7] = h.Charger
	buf[8] = h.EncryptFlag
	buf[9] = h.Checksum
	buf[10] = byte(h.DataLength)
	buf[11] = byte(h.DataLength >> 8)

	copy(buf[12:], encryptedData)
	return buf
}

// DecodeHeader 从二进制数据解码帧头
func DecodeHeader(data []byte) (MessageHeader, error) {
	if len(data) < 12 {
		return MessageHeader{}, io.ErrUnexpectedEOF
	}
	return MessageHeader{
		StartByte:   data[0],
		Version:     data[1],
		FuncCode:    data[2],
		PostNo:      uint32(data[3]) | uint32(data[4])<<8 | uint32(data[5])<<16 | uint32(data[6])<<24,
		Charger:     data[7],
		EncryptFlag: data[8],
		Checksum:    data[9],
		DataLength:  uint16(data[10]) | uint16(data[11])<<8,
	}, nil
}
