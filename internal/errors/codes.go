// Package errors provides business error code definitions.
package errors

import (
	gwcerrors "github.com/yannick2025-tech/gwc-errors"
)

// 服务码
const ServiceCodeNTSGater = 100

// 模块码
const (
	ModuleCodeCodec    = 1 // 编解码模块
	ModuleCodeValidator = 2 // 验证器模块
	ModuleCodeSession  = 3 // 会话管理模块
	ModuleCodeReport   = 4 // 测试报告模块
	ModuleCodeServer   = 5 // TCP服务模块
	ModuleCodeAPI      = 6 // API模块
)

// 错误码构建器
var (
	CodecBuilder    = gwcerrors.NewBusinessCodeBuilder(ServiceCodeNTSGater, ModuleCodeCodec)
	ValidatorBuilder = gwcerrors.NewBusinessCodeBuilder(ServiceCodeNTSGater, ModuleCodeValidator)
	SessionBuilder  = gwcerrors.NewBusinessCodeBuilder(ServiceCodeNTSGater, ModuleCodeSession)
	ReportBuilder   = gwcerrors.NewBusinessCodeBuilder(ServiceCodeNTSGater, ModuleCodeReport)
	ServerBuilder   = gwcerrors.NewBusinessCodeBuilder(ServiceCodeNTSGater, ModuleCodeServer)
	APIBuilder      = gwcerrors.NewBusinessCodeBuilder(ServiceCodeNTSGater, ModuleCodeAPI)
)

// ========== 编解码模块错误码 ==========
const (
	ErrFrameFormatInvalid    = 1000010001 // 帧格式错误
	ErrChecksumMismatch      = 1000010002 // 校验和不匹配
	ErrFuncCodeInvalid       = 1000010003 // 功能码非法
	ErrDecryptFailed         = 1000010004 // 数据域解密失败
	ErrEncryptFailed         = 1000010005 // 数据域加密失败
	ErrFieldDecodeFailed     = 1000010006 // 字段解码失败
	ErrFieldEncodeFailed     = 1000010007 // 字段编码失败
	ErrFrameTooShort         = 1000010008 // 帧长度不足
	ErrLengthMismatch        = 1000010009 // 长度域与实际数据不匹配
	ErrVersionUnsupported    = 1000010010 // 协议版本不支持
	ErrMessageNotRegistered  = 1000010011 // 消息未注册
	ErrProtocolDefLoadFailed = 1000010012 // 协议定义加载失败
)

// ========== 验证器模块错误码 ==========
const (
	ErrFieldRequired      = 1000020001 // 必填字段缺失
	ErrFieldValueOutOfRange = 1000020002 // 字段值超出范围
	ErrEnumValueInvalid   = 1000020003 // 枚举值非法
	ErrMessageTimeout     = 1000020004 // 消息超时
	ErrBitmapInvalid      = 1000020005 // 位图域非法
	ErrBCDFormatInvalid   = 1000020006 // BCD格式非法
	ErrASCIIFormatInvalid = 1000020007 // ASCII格式非法
	ErrTemperatureInvalid = 1000020008 // 温度值非法
	ErrLoopCountInvalid   = 1000020009 // 循环域数量非法
	ErrConstValueMismatch = 1000020010 // 常量值不匹配
)

// ========== 会话管理模块错误码 ==========
const (
	ErrSessionNotAuthenticated  = 1000030001 // 未认证
	ErrKeyVerificationFailed    = 1000030002 // 密钥校验失败
	ErrHeartbeatTimeout         = 1000030003 // 心跳超时
	ErrSessionNotFound          = 1000030004 // 会话不存在
	ErrSessionAlreadyExists     = 1000030005 // 会话已存在
	ErrKeyNotNegotiated         = 1000030006 // 密钥未协商
	ErrChargingInProgress       = 1000030007 // 充电进行中
	ErrNoActiveOrder            = 1000030008 // 无活跃订单
)

// ========== 测试报告模块错误码 ==========
const (
	ErrReportNotFound      = 1000040001 // 报告不存在
	ErrReportGenerateFailed = 1000040002 // 报告生成失败
	ErrPDFGenerateFailed   = 1000040003 // PDF生成失败
	ErrPDFFileNotFound     = 1000040004 // PDF文件不存在
	ErrReportSaveFailed    = 1000040005 // 报告保存失败
)

// ========== TCP服务模块错误码 ==========
const (
	ErrTCPListenFailed    = 1000050001 // TCP监听失败
	ErrConnectionClosed   = 1000050002 // 连接已关闭
	ErrConnectionReset    = 1000050003 // 连接重置
	ErrPacketSplitFailed  = 1000050004 // 粘包拆包失败
	ErrMessageDispatchFailed = 1000050005 // 消息分发失败
)

// ========== API模块错误码 ==========
const (
	ErrInvalidParameter   = 1000060001 // 参数无效
	ErrQueryFailed        = 1000060002 // 查询失败
	ErrInternalError      = 1000060003 // 内部错误
)

// 注册中文错误消息
func init() {
	gwcerrors.RegisterMessages(gwcerrors.Chinese, map[int]string{
		// 编解码模块
		ErrFrameFormatInvalid:    "帧格式错误",
		ErrChecksumMismatch:      "校验和不匹配",
		ErrFuncCodeInvalid:       "功能码非法",
		ErrDecryptFailed:         "数据域解密失败",
		ErrEncryptFailed:         "数据域加密失败",
		ErrFieldDecodeFailed:     "字段解码失败",
		ErrFieldEncodeFailed:     "字段编码失败",
		ErrFrameTooShort:         "帧长度不足",
		ErrLengthMismatch:        "长度域与实际数据不匹配",
		ErrVersionUnsupported:    "协议版本不支持",
		ErrMessageNotRegistered:  "消息未注册",
		ErrProtocolDefLoadFailed: "协议定义加载失败",

		// 验证器模块
		ErrFieldRequired:        "必填字段缺失",
		ErrFieldValueOutOfRange: "字段值超出范围",
		ErrEnumValueInvalid:     "枚举值非法",
		ErrMessageTimeout:       "消息超时",
		ErrBitmapInvalid:        "位图域非法",
		ErrBCDFormatInvalid:     "BCD格式非法",
		ErrASCIIFormatInvalid:   "ASCII格式非法",
		ErrTemperatureInvalid:   "温度值非法",
		ErrLoopCountInvalid:     "循环域数量非法",
		ErrConstValueMismatch:   "常量值不匹配",

		// 会话管理模块
		ErrSessionNotAuthenticated: "未认证",
		ErrKeyVerificationFailed:   "密钥校验失败",
		ErrHeartbeatTimeout:        "心跳超时",
		ErrSessionNotFound:         "会话不存在",
		ErrSessionAlreadyExists:    "会话已存在",
		ErrKeyNotNegotiated:        "密钥未协商",
		ErrChargingInProgress:      "充电进行中",
		ErrNoActiveOrder:           "无活跃订单",

		// 测试报告模块
		ErrReportNotFound:       "报告不存在",
		ErrReportGenerateFailed: "报告生成失败",
		ErrPDFGenerateFailed:    "PDF生成失败",
		ErrPDFFileNotFound:      "PDF文件不存在",
		ErrReportSaveFailed:     "报告保存失败",

		// TCP服务模块
		ErrTCPListenFailed:       "TCP监听失败",
		ErrConnectionClosed:      "连接已关闭",
		ErrConnectionReset:       "连接重置",
		ErrPacketSplitFailed:     "粘包拆包失败",
		ErrMessageDispatchFailed: "消息分发失败",

		// API模块
		ErrInvalidParameter: "参数无效",
		ErrQueryFailed:      "查询失败",
		ErrInternalError:    "内部错误",
	})

	gwcerrors.RegisterMessages(gwcerrors.USAEnglish, map[int]string{
		// Codec module
		ErrFrameFormatInvalid:    "invalid frame format",
		ErrChecksumMismatch:      "checksum mismatch",
		ErrFuncCodeInvalid:       "invalid function code",
		ErrDecryptFailed:         "data domain decrypt failed",
		ErrEncryptFailed:         "data domain encrypt failed",
		ErrFieldDecodeFailed:     "field decode failed",
		ErrFieldEncodeFailed:     "field encode failed",
		ErrFrameTooShort:         "frame too short",
		ErrLengthMismatch:        "length field mismatch",
		ErrVersionUnsupported:    "unsupported protocol version",
		ErrMessageNotRegistered:  "message not registered",
		ErrProtocolDefLoadFailed: "protocol definition load failed",

		// Validator module
		ErrFieldRequired:        "required field missing",
		ErrFieldValueOutOfRange: "field value out of range",
		ErrEnumValueInvalid:     "invalid enum value",
		ErrMessageTimeout:       "message timeout",
		ErrBitmapInvalid:        "invalid bitmap field",
		ErrBCDFormatInvalid:     "invalid BCD format",
		ErrASCIIFormatInvalid:   "invalid ASCII format",
		ErrTemperatureInvalid:   "invalid temperature value",
		ErrLoopCountInvalid:     "invalid loop count",
		ErrConstValueMismatch:   "const value mismatch",

		// Session module
		ErrSessionNotAuthenticated: "not authenticated",
		ErrKeyVerificationFailed:   "key verification failed",
		ErrHeartbeatTimeout:        "heartbeat timeout",
		ErrSessionNotFound:         "session not found",
		ErrSessionAlreadyExists:    "session already exists",
		ErrKeyNotNegotiated:        "key not negotiated",
		ErrChargingInProgress:      "charging in progress",
		ErrNoActiveOrder:           "no active order",

		// Report module
		ErrReportNotFound:       "report not found",
		ErrReportGenerateFailed: "report generate failed",
		ErrPDFGenerateFailed:    "PDF generate failed",
		ErrPDFFileNotFound:      "PDF file not found",
		ErrReportSaveFailed:     "report save failed",

		// Server module
		ErrTCPListenFailed:       "TCP listen failed",
		ErrConnectionClosed:      "connection closed",
		ErrConnectionReset:       "connection reset",
		ErrPacketSplitFailed:     "packet split failed",
		ErrMessageDispatchFailed: "message dispatch failed",

		// API module
		ErrInvalidParameter: "invalid parameter",
		ErrQueryFailed:      "query failed",
		ErrInternalError:    "internal error",
	})
}

