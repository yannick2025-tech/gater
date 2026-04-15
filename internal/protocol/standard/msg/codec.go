// Package msg provides BCD/ASCII/BYTE field read/write utilities.
package msg

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
)

// ==================== 读取辅助函数 ====================

// ReadByte 读取1字节无符号
func ReadByte(data []byte, offset int) (byte, int, error) {
	if offset+1 > len(data) {
		return 0, offset, fmt.Errorf("insufficient data for BYTE at offset %d", offset)
	}
	return data[offset], offset + 1, nil
}

// ReadUint16LE 读取2字节小端无符号
func ReadUint16LE(data []byte, offset int) (uint16, int, error) {
	if offset+2 > len(data) {
		return 0, offset, fmt.Errorf("insufficient data for UINT16 at offset %d", offset)
	}
	return binary.LittleEndian.Uint16(data[offset:]), offset + 2, nil
}

// ReadUint32LE 读取4字节小端无符号
func ReadUint32LE(data []byte, offset int) (uint32, int, error) {
	if offset+4 > len(data) {
		return 0, offset, fmt.Errorf("insufficient data for UINT32 at offset %d", offset)
	}
	return binary.LittleEndian.Uint32(data[offset:]), offset + 4, nil
}

// ReadBCD 读取BCD编码，返回原始十六进制字符串
func ReadBCD(data []byte, offset, length int) (string, int, error) {
	if offset+length > len(data) {
		return "", offset, fmt.Errorf("insufficient data for BCD at offset %d", offset)
	}
	result := fmt.Sprintf("%X", data[offset:offset+length])
	return result, offset + length, nil
}

// ReadASCII 读取ASCII字符串，去除右侧空格和0填充
func ReadASCII(data []byte, offset, length int) (string, int, error) {
	if offset+length > len(data) {
		return "", offset, fmt.Errorf("insufficient data for ASCII at offset %d", offset)
	}
	s := string(data[offset : offset+length])
	// 去除右侧空格和0填充
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == 0) {
		s = s[:len(s)-1]
	}
	return s, offset + length, nil
}

// ReadBytes 读取指定长度的原始字节
func ReadBytes(data []byte, offset, length int) ([]byte, int, error) {
	if offset+length > len(data) {
		return nil, offset, fmt.Errorf("insufficient data for BYTES at offset %d", offset)
	}
	result := make([]byte, length)
	copy(result, data[offset:offset+length])
	return result, offset + length, nil
}

// ReadTemperature 读取温度值，自动处理偏移和特殊值
func ReadTemperature(data []byte, offset int, cfg types.TempConfig) (interface{}, int, error) {
	if offset+1 > len(data) {
		return nil, offset, fmt.Errorf("insufficient data for TEMPERATURE at offset %d", offset)
	}
	v := data[offset]
	if v == cfg.InvalidValue {
		return nil, offset + 1, nil // null
	}
	if v == cfg.AbnormalValue {
		return "abnormal", offset + 1, nil
	}
	return int(v) + cfg.Offset, offset + 1, nil
}

// ==================== 写入辅助函数 ====================

// WriteByte 写入1字节
func WriteByte(buf []byte, offset int, v byte) int {
	buf[offset] = v
	return offset + 1
}

// WriteUint16LE 写入2字节小端
func WriteUint16LE(buf []byte, offset int, v uint16) int {
	binary.LittleEndian.PutUint16(buf[offset:], v)
	return offset + 2
}

// WriteUint32LE 写入4字节小端
func WriteUint32LE(buf []byte, offset int, v uint32) int {
	binary.LittleEndian.PutUint32(buf[offset:], v)
	return offset + 4
}

// WriteBCD 写入BCD编码（从十六进制字符串）
func WriteBCD(buf []byte, offset int, hexStr string, length int) (int, error) {
	// 将十六进制字符串转为字节
	padded := hexStr
	for len(padded) < length*2 {
		padded = "0" + padded
	}
	b, err := hexToBytes(padded)
	if err != nil {
		return offset, err
	}
	// 取后length字节
	start := 0
	if len(b) > length {
		start = len(b) - length
	}
	copy(buf[offset:], b[start:])
	return offset + length, nil
}

// WriteASCII 写入ASCII字符串，右侧空格填充
func WriteASCII(buf []byte, offset int, s string, length int) int {
	b := []byte(s)
	for i := 0; i < length; i++ {
		if i < len(b) {
			buf[offset+i] = b[i]
		} else {
			buf[offset+i] = 0x20 // 空格填充
		}
	}
	return offset + length
}

// WriteTemperature 写入温度值
func WriteTemperature(buf []byte, offset int, temp interface{}, cfg types.TempConfig) int {
	switch v := temp.(type) {
	case nil:
		buf[offset] = cfg.InvalidValue
	case string:
		buf[offset] = cfg.AbnormalValue
	case int:
		if v < cfg.Offset || v > cfg.Offset+cfg.ValidMax {
			buf[offset] = cfg.InvalidValue
		} else {
			buf[offset] = byte(v - cfg.Offset)
		}
	case float64:
		iv := int(math.Round(v))
		if iv < cfg.Offset || iv > cfg.Offset+cfg.ValidMax {
			buf[offset] = cfg.InvalidValue
		} else {
			buf[offset] = byte(iv - cfg.Offset)
		}
	}
	return offset + 1
}

// ==================== 工具函数 ====================

// hexToBytes 十六进制字符串转字节
func hexToBytes(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		s = "0" + s
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		var hi, lo byte
		_, err := fmt.Sscanf(s[i:i+1], "%X", &hi)
		if err != nil {
			return nil, err
		}
		_, err = fmt.Sscanf(s[i+1:i+2], "%X", &lo)
		if err != nil {
			return nil, err
		}
		b[i/2] = hi<<4 | lo
	}
	return b, nil
}

// MakeSpec 快速创建MessageSpec
func MakeSpec(code byte, dir types.Direction, name string, encrypt, needReply bool) types.MessageSpec {
	return types.MessageSpec{
		FuncCode:  code,
		Direction: dir,
		Name:      name,
		Encrypt:   encrypt,
		NeedReply: needReply,
	}
}
