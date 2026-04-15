// Package types provides the Protocol interface and related type definitions.
package types

// Protocol 协议接口 - 定义一个充电协议的核心能力
// 不同的充电协议实现此接口即可接入系统
type Protocol interface {

	// Name 协议名称
	Name() string
	// Version 协议版本
	Version() byte
	// FrameConfig 帧格式配置
	FrameConfig() FrameConfig
	// CryptoConfig 加密配置
	CryptoConfig() CryptoConfig
	// TempConfig 温度编码配置
	TempConfig() TempConfig
	// Registry 消息注册表
	Registry() MessageRegistry
	// IsFixedKeyFuncCode 判断功能码是否使用固定密钥
	IsFixedKeyFuncCode(code byte) bool
}

// FrameConfig 帧格式配置
type FrameConfig struct {
	StartByte  byte // 起始域，固定0x32
	HeaderSize int  // 帧头大小（不含数据域）：1+1+1+4+1+1+1+2=12
}

// CryptoConfig 加密配置
type CryptoConfig struct {
	Algorithm          string // 加密算法: "AES256-CBC-PKCS7"
	FixedKey           string // 固定密钥（32字符ASCII字符串，每个字符ASCII码作为一字节）
	IVRule             string // IV规则: "last_16_bytes_of_key" 取KEY后16字符
	ZeroLengthNoEncrypt bool   // 数据域长度为0时不加密
}

// TempConfig 温度编码配置
type TempConfig struct {
	ValidMin      int    // 有效值最小值
	ValidMax      int    // 有效值最大值
	Offset        int    // 温度偏移量
	AbnormalValue byte   // 异常值
	InvalidValue  byte   // 无效值
}

// NewDefaultFrameConfig 默认帧配置
func NewDefaultFrameConfig() FrameConfig {
	return FrameConfig{
		StartByte:  0x32,
		HeaderSize: 12,
	}
}
