package codec

import (
	"testing"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/codec"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
)

func newTestProtocol() *standard.StandardProtocol {
	return standard.New()
}

func TestCalcChecksum(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected byte
	}{
		{
			name:     "empty data",
			data:     []byte{},
			expected: 0,
		},
		{
			name:     "single zero byte",
			data:     []byte{0x00},
			expected: 0,
		},
		{
			name:     "simple sum no overflow",
			data:     []byte{0x01, 0x02, 0x03},
			expected: 0x06,
		},
		{
			name:     "0x01 heartbeat upload data domain",
			data:     []byte{0x0B, 0x00, 0x00, 0x0B},
			expected: 0x16,
		},
		{
			name:     "sum exceeds 0xFF - complement needed",
			data:     []byte{0xFF, 0xFF, 0x02},
			expected: 0x00, // 0xFF+0xFF+0x02=0x200, low=0x00, complement=0x00
		},
		{
			name:     "sum exactly 0x100",
			data:     []byte{0x80, 0x80},
			expected: 0x00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := codec.CalcChecksum(tt.data)
			if got != tt.expected {
				t.Errorf("CalcChecksum() = 0x%02X, expected 0x%02X", got, tt.expected)
			}
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	checksum := codec.CalcChecksum(data)

	if !codec.VerifyChecksum(data, checksum) {
		t.Error("VerifyChecksum should return true for correct checksum")
	}
	if codec.VerifyChecksum(data, 0xFF) {
		t.Error("VerifyChecksum should return false for incorrect checksum")
	}
}

func TestFrameScanner_SingleFrame(t *testing.T) {
	scanner := codec.NewFrameScanner()

	// 0x0A upload: 12 bytes header, no data domain
	frame := []byte{0x32, 0x06, 0x0A, 0x02, 0xFC, 0x7B, 0x89, 0x00, 0x00, 0x00, 0x00, 0x00}
	scanner.Feed(frame)

	got := scanner.Next()
	if got == nil {
		t.Fatal("Next() should return a frame")
	}
	if len(got) != 12 {
		t.Errorf("frame length = %d, expected 12", len(got))
	}
	if got[0] != 0x32 {
		t.Errorf("start byte = 0x%02X, expected 0x32", got[0])
	}
	if got[2] != 0x0A {
		t.Errorf("func code = 0x%02X, expected 0x0A", got[2])
	}
}

func TestFrameScanner_PartialData(t *testing.T) {
	scanner := codec.NewFrameScanner()

	frame := []byte{0x32, 0x06, 0x0A, 0x02, 0xFC, 0x7B, 0x89, 0x00, 0x00, 0x00, 0x00, 0x00}
	scanner.Feed(frame[:6])

	if got := scanner.Next(); got != nil {
		t.Error("Next() should return nil for partial data")
	}

	scanner.Feed(frame[6:])
	got := scanner.Next()
	if got == nil {
		t.Fatal("Next() should return a frame after feeding remaining data")
	}
	if len(got) != 12 {
		t.Errorf("frame length = %d, expected 12", len(got))
	}
}

func TestFrameScanner_MultipleFrames(t *testing.T) {
	scanner := codec.NewFrameScanner()

	// Two frames with no data domain
	frame1 := []byte{0x32, 0x06, 0x23, 0x6E, 0xF1, 0xFB, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	frame2 := []byte{0x32, 0x06, 0x01, 0xFC, 0x7B, 0x89, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	scanner.Feed(append(frame1, frame2...))

	got1 := scanner.Next()
	if got1 == nil {
		t.Fatal("first frame should not be nil")
	}
	if got1[2] != 0x23 {
		t.Errorf("first frame func code = 0x%02X, expected 0x23", got1[2])
	}

	got2 := scanner.Next()
	if got2 == nil {
		t.Fatal("second frame should not be nil")
	}
	if got2[2] != 0x01 {
		t.Errorf("second frame func code = 0x%02X, expected 0x01", got2[2])
	}

	if got3 := scanner.Next(); got3 != nil {
		t.Error("no more frames expected")
	}
}

func TestFrameScanner_InvalidStartByte(t *testing.T) {
	scanner := codec.NewFrameScanner()

	garbage := []byte{0xAA, 0xBB, 0xCC}
	frame := []byte{0x32, 0x06, 0x01, 0xFC, 0x7B, 0x89, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	scanner.Feed(append(garbage, frame...))

	got := scanner.Next()
	if got == nil {
		t.Fatal("Next() should skip invalid bytes and return frame")
	}
	if got[0] != 0x32 {
		t.Errorf("start byte = 0x%02X, expected 0x32", got[0])
	}
}

func TestFrameScanner_Reset(t *testing.T) {
	scanner := codec.NewFrameScanner()
	scanner.Feed([]byte{0x32, 0x06})
	if scanner.Len() != 2 {
		t.Errorf("buffer len = %d, expected 2", scanner.Len())
	}
	scanner.Reset()
	if scanner.Len() != 0 {
		t.Errorf("buffer len after reset = %d, expected 0", scanner.Len())
	}
}

func TestFrameScanner_WithDataDomain(t *testing.T) {
	scanner := codec.NewFrameScanner()

	// 文档中的真实 0x01 心跳报文: 320601897BFC0201000B03000B0000
	// header: 32 06 01 89 7B FC 02 01 00 0B 03 00 (12字节)
	// data:   0B 00 00 (3字节: status=0x0B, errorCode=0x0000, signal=0x00)
	frame := []byte{0x32, 0x06, 0x01, 0x89, 0x7B, 0xFC, 0x02, 0x01, 0x00, 0x0B, 0x03, 0x00, 0x0B, 0x00, 0x00}
	scanner.Feed(frame)

	got := scanner.Next()
	if got == nil {
		t.Fatal("Next() should return a frame with data domain")
	}
	if len(got) != 15 {
		t.Errorf("frame length = %d, expected 15", len(got))
	}
	if got[2] != 0x01 {
		t.Errorf("func code = 0x%02X, expected 0x01", got[2])
	}
	// 数据长度字段: little-endian 0x03 0x00 = 3
	dataLen := uint16(got[10]) | uint16(got[11])<<8
	if dataLen != 3 {
		t.Errorf("data length = %d, expected 3", dataLen)
	}
}

func TestFrameDecoder_ValidFrame(t *testing.T) {
	proto := newTestProtocol()
	decoder := codec.NewFrameDecoder(proto)

	// 0x23 time sync upload: 32 06 23 FB F1 6E 03 00 00 00 00 00
	frame := []byte{0x32, 0x06, 0x23, 0x6E, 0xF1, 0xFB, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	result, err := decoder.Decode(frame, nil)
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	if result.Header.FuncCode != 0x23 {
		t.Errorf("func code = 0x%02X, expected 0x23", result.Header.FuncCode)
	}
	if result.Header.DataLength != 0 {
		t.Errorf("data length = %d, expected 0", result.Header.DataLength)
	}
	if result.Header.Version != 0x06 {
		t.Errorf("version = 0x%02X, expected 0x06", result.Header.Version)
	}
}

func TestFrameDecoder_InvalidStartByte(t *testing.T) {
	proto := newTestProtocol()
	decoder := codec.NewFrameDecoder(proto)

	frame := []byte{0x33, 0x06, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	_, err := decoder.Decode(frame, nil)
	if err != codec.ErrInvalidStartByte {
		t.Errorf("expected ErrInvalidStartByte, got: %v", err)
	}
}

func TestFrameDecoder_ChecksumMismatch(t *testing.T) {
	proto := newTestProtocol()
	decoder := codec.NewFrameDecoder(proto)

	// 0x01 heartbeat upload with data domain but wrong checksum
	frame := []byte{0x32, 0x06, 0x01, 0xFC, 0x7B, 0x89, 0x01, 0x00, 0xFF, 0x03, 0x00, 0x0B, 0x00, 0x00, 0x0B}
	_, err := decoder.Decode(frame, nil)
	if err == nil {
		t.Error("expected checksum mismatch error")
	}
}

func TestFrameDecoder_FrameTooShort(t *testing.T) {
	proto := newTestProtocol()
	decoder := codec.NewFrameDecoder(proto)

	frame := []byte{0x32, 0x06}
	_, err := decoder.Decode(frame, nil)
	if err != codec.ErrFrameTooShort {
		t.Errorf("expected ErrFrameTooShort, got: %v", err)
	}
}

func TestValidateFrame(t *testing.T) {
	frame := []byte{0x32, 0x06, 0x0A, 0x02, 0xFC, 0x7B, 0x89, 0x00, 0x00, 0x00, 0x00, 0x00}
	totalLen, err := codec.ValidateFrame(frame)
	if err != nil {
		t.Fatalf("ValidateFrame() error: %v", err)
	}
	if totalLen != 12 {
		t.Errorf("totalLen = %d, expected 12", totalLen)
	}
}

func TestFrameEncoder_NoEncrypt(t *testing.T) {
	proto := newTestProtocol()
	encoder := codec.NewFrameEncoder(proto)

	header := types.MessageHeader{
		StartByte:  0x32,
		Version:    0x06,
		FuncCode:   0x0A,
		PostNo:     0x027BF1FB,
		EncryptFlag: 0x00,
	}
	data := []byte{}

	frame, err := encoder.Encode(header, data, nil)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}
	if len(frame) != 12 {
		t.Errorf("frame length = %d, expected 12", len(frame))
	}
	if frame[0] != 0x32 {
		t.Errorf("start byte = 0x%02X, expected 0x32", frame[0])
	}

	// round-trip: decode should succeed
	decoder := codec.NewFrameDecoder(proto)
	result, err := decoder.Decode(frame, nil)
	if err != nil {
		t.Fatalf("round-trip Decode() error: %v", err)
	}
	if result.Header.FuncCode != 0x0A {
		t.Errorf("round-trip func code = 0x%02X, expected 0x0A", result.Header.FuncCode)
	}
}
