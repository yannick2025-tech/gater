package msg

import (
	"testing"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
)

func TestReadBCD(t *testing.T) {
	data := []byte{0x20, 0x25, 0x04, 0x14, 0x15, 0x30, 0x00}

	result, off, err := ReadBCD(data, 0, 7)
	if err != nil {
		t.Fatalf("ReadBCD() error: %v", err)
	}
	if off != 7 {
		t.Errorf("offset = %d, expected 7", off)
	}
	if result != "20250414153000" {
		t.Errorf("BCD = %q, expected %q", result, "20250414153000")
	}
}

func TestWriteBCD(t *testing.T) {
	buf := make([]byte, 7)
	_, err := WriteBCD(buf, 0, "20250414153000", 7)
	if err != nil {
		t.Fatalf("WriteBCD() error: %v", err)
	}
	expected := []byte{0x20, 0x25, 0x04, 0x14, 0x15, 0x30, 0x00}
	for i := range expected {
		if buf[i] != expected[i] {
			t.Errorf("buf[%d] = 0x%02X, expected 0x%02X", i, buf[i], expected[i])
		}
	}
}

func TestReadASCII(t *testing.T) {
	data := make([]byte, 50)
	copy(data, "HELLO WORLD")
	// remaining bytes are 0x00, which will be trimmed by ReadASCII

	result, off, err := ReadASCII(data, 0, 50)
	if err != nil {
		t.Fatalf("ReadASCII() error: %v", err)
	}
	if off != 50 {
		t.Errorf("offset = %d, expected 50", off)
	}
	if result != "HELLO WORLD" {
		t.Errorf("ASCII = %q, expected %q", result, "HELLO WORLD")
	}
}

func TestWriteASCII(t *testing.T) {
	buf := make([]byte, 10)
	WriteASCII(buf, 0, "AB", 10)
	for i := 2; i < 10; i++ {
		if buf[i] != 0x20 {
			t.Errorf("padding at %d = 0x%02X, expected 0x20", i, buf[i])
		}
	}
}

func TestReadTemperature(t *testing.T) {
	cfg := types.TempConfig{
		ValidMin:      0,
		ValidMax:      250,
		Offset:        -40,
		AbnormalValue: 0xFE,
		InvalidValue:  0xFF,
	}

	tests := []struct {
		name     string
		input    byte
		expected interface{}
	}{
		{"invalid", 0xFF, nil},
		{"abnormal", 0xFE, "abnormal"},
		{"zero_celsius", 40, 0},     // 40 + (-40) = 0
		{"positive", 80, 40},         // 80 + (-40) = 40
		{"negative", 20, int(-20)},   // 20 + (-40) = -20
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := ReadTemperature([]byte{tt.input}, 0, cfg)
			if err != nil {
				t.Fatalf("ReadTemperature() error: %v", err)
			}
			switch v := result.(type) {
			case nil:
				if tt.expected != nil {
					t.Errorf("expected %v, got nil", tt.expected)
				}
			case string:
				if v != tt.expected {
					t.Errorf("got %q, expected %q", v, tt.expected)
				}
			case int:
				if v != tt.expected.(int) {
					t.Errorf("got %d, expected %d", v, tt.expected)
				}
			}
		})
	}
}

func TestWriteTemperature(t *testing.T) {
	cfg := types.TempConfig{
		ValidMin:      0,
		ValidMax:      250,
		Offset:        -40,
		AbnormalValue: 0xFE,
		InvalidValue:  0xFF,
	}

	tests := []struct {
		name     string
		input    interface{}
		expected byte
	}{
		{"nil", nil, 0xFF},
		{"abnormal", "abnormal", 0xFE},
		{"zero_celsius", 0, 40},
		{"positive_40", 40, 80},
		{"negative_20", -20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, 1)
			WriteTemperature(buf, 0, tt.input, cfg)
			if buf[0] != tt.expected {
				t.Errorf("got 0x%02X, expected 0x%02X", buf[0], tt.expected)
			}
		})
	}
}

func TestReadUint16LE(t *testing.T) {
	data := []byte{0x34, 0x12} // little-endian: 0x1234
	val, off, err := ReadUint16LE(data, 0)
	if err != nil {
		t.Fatalf("ReadUint16LE() error: %v", err)
	}
	if val != 0x1234 {
		t.Errorf("got 0x%04X, expected 0x1234", val)
	}
	if off != 2 {
		t.Errorf("offset = %d, expected 2", off)
	}
}

func TestReadUint32LE(t *testing.T) {
	data := []byte{0x78, 0x56, 0x34, 0x12} // 0x12345678
	val, off, err := ReadUint32LE(data, 0)
	if err != nil {
		t.Fatalf("ReadUint32LE() error: %v", err)
	}
	if val != 0x12345678 {
		t.Errorf("got 0x%08X, expected 0x12345678", val)
	}
	if off != 4 {
		t.Errorf("offset = %d, expected 4", off)
	}
}

func TestInsufficientData(t *testing.T) {
	data := []byte{0x01}

	_, _, err := ReadByte(data, 5)
	if err == nil {
		t.Error("expected error for out of bounds ReadByte")
	}

	_, _, err = ReadBCD(data, 0, 10)
	if err == nil {
		t.Error("expected error for out of bounds ReadBCD")
	}

	_, _, err = ReadASCII(data, 0, 50)
	if err == nil {
		t.Error("expected error for out of bounds ReadASCII")
	}
}

func TestMakeSpec(t *testing.T) {
	spec := MakeSpec(0x0A, types.DirectionUpload, "auth_random", false, true)
	if spec.FuncCode != 0x0A {
		t.Errorf("func code = 0x%02X, expected 0x0A", spec.FuncCode)
	}
	if spec.Direction != types.DirectionUpload {
		t.Error("direction should be Upload")
	}
	if spec.Encrypt != false {
		t.Error("encrypt should be false")
	}
	if spec.NeedReply != true {
		t.Error("needReply should be true")
	}
}
