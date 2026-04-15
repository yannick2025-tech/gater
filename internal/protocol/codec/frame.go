// Package codec provides frame encoding and decoding.
package codec

import (
	"errors"
	"fmt"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
)

var (
	ErrInvalidStartByte = errors.New("invalid start byte")
	ErrFrameTooShort    = errors.New("frame too short")
	ErrChecksumMismatch = errors.New("checksum mismatch")
)

// FrameEncoder 帧编码器
type FrameEncoder struct {
	proto types.Protocol
}

// NewFrameEncoder 创建帧编码器
func NewFrameEncoder(proto types.Protocol) *FrameEncoder {
	return &FrameEncoder{proto: proto}
}

// Encode 编码完整帧
// 流程：消息体编码 → 加密（如需）→ 计算校验和 → 组装帧头 → 拼接完整帧
func (e *FrameEncoder) Encode(header types.MessageHeader, bodyData []byte, encryptFn func([]byte) ([]byte, error)) ([]byte, error) {
	var encryptedData []byte
	var err error

	// 判断是否需要加密
	if header.EncryptFlag == 0x01 && len(bodyData) > 0 && encryptFn != nil {
		encryptedData, err = encryptFn(bodyData)
		if err != nil {
			return nil, fmt.Errorf("encrypt failed: %w", err)
		}
	} else {
		encryptedData = bodyData
	}

	// 更新数据域长度（加密后的长度）
	header.DataLength = uint16(len(encryptedData))

	// 计算校验和（基于加密后的数据域）
	header.Checksum = CalcChecksum(encryptedData)

	// 组装完整帧
	return header.EncodeFrame(encryptedData), nil
}

// FrameDecoder 帧解码器
type FrameDecoder struct {
	proto types.Protocol
}

// NewFrameDecoder 创建帧解码器
func NewFrameDecoder(proto types.Protocol) *FrameDecoder {
	return &FrameDecoder{proto: proto}
}

// DecodeResult 解码结果
type DecodeResult struct {
	Header         types.MessageHeader
	DecryptedData  []byte // 解密后的数据域
	RawData        []byte // 加密后的原始数据域
}

// Decode 解码完整帧
// 流程：解析帧头 → 校验起始域 → 提取数据域 → 校验和验证 → 解密（如需）
func (d *FrameDecoder) Decode(frame []byte, decryptFn func([]byte) ([]byte, error)) (*DecodeResult, error) {
	// 最少需要帧头12字节
	if len(frame) < 12 {
		return nil, ErrFrameTooShort
	}

	// 解码帧头
	header, err := types.DecodeHeader(frame)
	if err != nil {
		return nil, fmt.Errorf("decode header failed: %w", err)
	}

	// 校验起始域
	if header.StartByte != d.proto.FrameConfig().StartByte {
		return nil, ErrInvalidStartByte
	}

	// 校验帧长度
	expectedLen := 12 + int(header.DataLength)
	if len(frame) < expectedLen {
		return nil, fmt.Errorf("frame length mismatch: expected %d, got %d", expectedLen, len(frame))
	}

	// 提取加密后的数据域
	rawData := frame[12 : 12+header.DataLength]

	// 校验和验证
	expectedChecksum := CalcChecksum(rawData)
	if expectedChecksum != header.Checksum {
		return nil, fmt.Errorf("%w: expected 0x%02X, got 0x%02X", ErrChecksumMismatch, expectedChecksum, header.Checksum)
	}

	// 解密数据域（如需）
	var decryptedData []byte
	if header.EncryptFlag == 0x01 && len(rawData) > 0 && decryptFn != nil {
		decryptedData, err = decryptFn(rawData)
		if err != nil {
			return nil, fmt.Errorf("decrypt failed: %w", err)
		}
	} else {
		decryptedData = rawData
	}

	return &DecodeResult{
		Header:        header,
		DecryptedData: decryptedData,
		RawData:       rawData,
	}, nil
}

// ValidateFrame 验证帧基本格式（不包含数据域解密）
func ValidateFrame(data []byte) (int, error) {
	if len(data) < 12 {
		return 0, ErrFrameTooShort
	}

	// 检查起始字节
	if data[0] != 0x32 {
		return 0, ErrInvalidStartByte
	}

	// 读取数据域长度
	dataLength := uint16(data[10]) | uint16(data[11])<<8
	totalLen := 12 + int(dataLength)

	return totalLen, nil
}
