// Package codec provides TCP stream scanning and frame extraction.
package codec

import (
	"bytes"
	"errors"
)

var (
	ErrIncompleteFrame = errors.New("incomplete frame data")
)

// FrameScanner TCP粘包处理器
// 协议帧格式：起始域(1) + 版本(1) + 功能码(1) + 桩编号(4) + 枪号(1) +
//             加密标志(1) + 校验码(1) + 消息长度(2) + 数据域(N)
// 帧头固定12字节，消息长度字段指明数据域长度
type FrameScanner struct {
	buf       bytes.Buffer
	headerSize int // 帧头大小，默认12
	startByte byte // 起始域，默认0x32
}

// NewFrameScanner 创建TCP粘包处理器
func NewFrameScanner() *FrameScanner {
	return &FrameScanner{
		headerSize: 12,
		startByte:  0x32,
	}
}

// Feed 写入接收到的TCP数据
func (s *FrameScanner) Feed(data []byte) {
	s.buf.Write(data)
}

// Next 提取下一个完整帧
// 返回：完整帧数据，如果没有完整帧则返回nil
func (s *FrameScanner) Next() []byte {
	for {
		// 至少需要帧头大小才能解析
		if s.buf.Len() < s.headerSize {
			return nil
		}

		// 读取帧头前12字节（不从buffer中移除）
		header := s.buf.Bytes()[:s.headerSize]

		// 检查起始字节
		if header[0] != s.startByte {
			// 跳过无效字节，查找下一个起始字节
			s.buf.Next(1)
			continue
		}

		// 读取数据域长度
		dataLength := uint16(header[10]) | uint16(header[11])<<8
		totalLen := s.headerSize + int(dataLength)

		// 检查是否有完整帧
		if s.buf.Len() < totalLen {
			return nil // 数据不完整，等待更多数据
		}

		// 提取完整帧
		frame := make([]byte, totalLen)
		n, _ := s.buf.Read(frame)
		return frame[:n]
	}
}

// Reset 重置缓冲区
func (s *FrameScanner) Reset() {
	s.buf.Reset()
}

// Len 返回缓冲区中剩余数据长度
func (s *FrameScanner) Len() int {
	return s.buf.Len()
}
