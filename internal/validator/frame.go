// Package validator provides frame and message validation.
package validator

import (
	"fmt"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
)

// Error 验证错误
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// FrameValidator 帧验证器
type FrameValidator struct {
	proto types.Protocol
}

// New 创建帧验证器
func New(proto types.Protocol) *FrameValidator {
	return &FrameValidator{proto: proto}
}

// ValidateFuncCode 验证功能码是否已注册
func (v *FrameValidator) ValidateFuncCode(funcCode byte, dir types.Direction) *Error {
	_, ok := v.proto.Registry().Spec(funcCode, dir)
	if !ok {
		return &Error{
			Code:    "FUNC_CODE_UNKNOWN",
			Message: fmt.Sprintf("func code 0x%02X not registered", funcCode),
		}
	}
	return nil
}

// ValidateHeader 验证帧头合法性
func (v *FrameValidator) ValidateHeader(header types.MessageHeader) *Error {
	if header.StartByte != v.proto.FrameConfig().StartByte {
		return &Error{
			Code:    "INVALID_START_BYTE",
			Message: fmt.Sprintf("start byte 0x%02X, expected 0x%02X", header.StartByte, v.proto.FrameConfig().StartByte),
		}
	}

	if header.Version != v.proto.Version() {
		// 版本不匹配不再强校验，协议会持续升级，后续再考虑多版本兼容方案
		_ = header.Version // 占位，调用方自行决定是否记录日志
	}

	if header.EncryptFlag != 0x00 && header.EncryptFlag != 0x01 {
		return &Error{
			Code:    "INVALID_ENCRYPT_FLAG",
			Message: fmt.Sprintf("encrypt flag 0x%02X", header.EncryptFlag),
		}
	}

	return nil
}

// ValidatePostNo 验证充电桩编号合法性
// 协议规定：充电桩编号为8位数字（10000000~99999999），枪编号在充电桩编号后追加01、02等
func (v *FrameValidator) ValidatePostNo(postNo uint32) *Error {
	if postNo < 10000000 || postNo > 99999999 {
		return &Error{
			Code:    "INVALID_POST_NO",
			Message: fmt.Sprintf("postNo %d is not 8 digits, expected 10000000~99999999", postNo),
		}
	}
	return nil
}

// ValidateMessage 验证消息字段
func (v *FrameValidator) ValidateMessage(msg types.Message) []types.ValidationError {
	return msg.Validate()
}
