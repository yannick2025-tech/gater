package handlers

import (
	"bytes"
	"crypto/md5"
	"testing"
)

func TestBytesToBCD(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []byte
	}{
		{"0x00", []byte{0x00}, []byte{0x00, 0x00}},
		{"0x12", []byte{0x12}, []byte{0x01, 0x02}},
		{"0xAB", []byte{0xAB}, []byte{0x0A, 0x0B}},
		{"0xFF", []byte{0xFF}, []byte{0x0F, 0x0F}},
		{"multi", []byte{0x12, 0x34}, []byte{0x01, 0x02, 0x03, 0x04}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bytesToBCD(tt.in)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("bytesToBCD(% X) = % X, want % X", tt.in, got, tt.want)
			}
		})
	}
}

func TestComputeAuthHash_RoundTrip(t *testing.T) {
	// 测试：用已知随机数和固定密钥计算hash，验证算法一致
	randomKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D}
	fixedKey := []byte("4845727543536D5570716A3843596451") // 32字节

	hash1, err := computeAuthHash(randomKey, fixedKey)
	if err != nil {
		t.Fatalf("computeAuthHash() error: %v", err)
	}

	// 再次计算，结果应一致
	hash2, err := computeAuthHash(randomKey, fixedKey)
	if err != nil {
		t.Fatalf("computeAuthHash() error: %v", err)
	}

	if !bytes.Equal(hash1, hash2) {
		t.Errorf("computeAuthHash not deterministic: % X vs % X", hash1, hash2)
	}

	// 不同随机数应产生不同hash
	otherKey := []byte{0x0D, 0x0C, 0x0B, 0x0A, 0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01}
	hash3, _ := computeAuthHash(otherKey, fixedKey)
	if bytes.Equal(hash1, hash3) {
		t.Error("different random keys should produce different hashes")
	}

	// hash应为16字节
	if len(hash1) != 16 {
		t.Errorf("hash length = %d, expected 16", len(hash1))
	}
}

func TestComputeAuthHash_AlgorithmSteps(t *testing.T) {
	// 手动验证算法步骤
	randomKey := []byte{0x01, 0x02, 0x03} // 简化，实际应13字节
	padded := make([]byte, 13)
	copy(padded, randomKey) // 补0到13字节
	fixedKey := []byte("4845727543536D5570716A3843596451")

	// Step 1: 倒序
	reversed := make([]byte, 13)
	for i := 0; i < 13; i++ {
		reversed[i] = padded[12-i]
	}

	// Step 2: 拼接 13+16=29
	combined := make([]byte, 0, 29)
	combined = append(combined, reversed...)
	combined = append(combined, fixedKey[:16]...)

	if len(combined) != 29 {
		t.Fatalf("combined length = %d, expected 29", len(combined))
	}

	// Step 3: BCD
	bcdBytes := bytesToBCD(combined)
	if len(bcdBytes) != 58 {
		t.Fatalf("BCD length = %d, expected 58", len(bcdBytes))
	}

	// Step 4: MD5
	hash := md5.Sum(bcdBytes)

	// Step 5: 取前16字节倒序
	result := make([]byte, 16)
	for i := 0; i < 16; i++ {
		result[i] = hash[15-i]
	}

	// 验证通过 computeAuthHash 得到相同结果
	computed, err := computeAuthHash(padded, fixedKey)
	if err != nil {
		t.Fatalf("computeAuthHash() error: %v", err)
	}

	if !bytes.Equal(computed, result) {
		t.Errorf("computeAuthHash = % X, expected % X", computed, result)
	}
}

func TestUint16ToBytes(t *testing.T) {
	b := Uint16ToBytes(0x1234)
	if b[0] != 0x34 || b[1] != 0x12 {
		t.Errorf("got % X, expected 34 12", b)
	}
}

func TestUint32ToBytes(t *testing.T) {
	b := Uint32ToBytes(0x12345678)
	if b[0] != 0x78 || b[1] != 0x56 || b[2] != 0x34 || b[3] != 0x12 {
		t.Errorf("got % X, expected 78 56 34 12", b)
	}
}
