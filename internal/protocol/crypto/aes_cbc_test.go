package crypto

import (
	"bytes"
	"testing"
)

// 文档示例:
// KEY = "4845727543536D5570716A3843596451" (32 ASCII字符 → 32字节)
// IV  = "70716A3843596451" (KEY后16字符 → 16字节)
// 加密前: 0x00
// 加密后: 41 6D F1 63 27 FA AD 87 9C 51 86 3D B1 07 53 55

func TestNewAESCBCCipher(t *testing.T) {
	cipher, err := NewAESCBCCipher("4845727543536D5570716A3843596451", "last_16_bytes_of_key")
	if err != nil {
		t.Fatalf("NewAESCBCCipher() error: %v", err)
	}
	if len(cipher.Key()) != 32 {
		t.Errorf("key length = %d, expected 32", len(cipher.Key()))
	}
	if len(cipher.IV()) != 16 {
		t.Errorf("IV length = %d, expected 16", len(cipher.IV()))
	}
}

func TestNewAESCBCCipher_IVMatches(t *testing.T) {
	cipher, err := NewAESCBCCipher("4845727543536D5570716A3843596451", "last_16_bytes_of_key")
	if err != nil {
		t.Fatalf("NewAESCBCCipher() error: %v", err)
	}
	// IV = KEY后16字符 "70716A3843596451" 的 ASCII 码
	expected := []byte{55, 48, 55, 49, 54, 65, 51, 56, 52, 51, 53, 57, 54, 52, 53, 49}
	if !bytes.Equal(cipher.IV(), expected) {
		t.Errorf("IV = %v, expected %v", cipher.IV(), expected)
	}
}

func TestNewAESCBCCipher_InvalidKeyLength(t *testing.T) {
	_, err := NewAESCBCCipher("short", "last_16_bytes_of_key")
	if err != ErrInvalidKeyLength {
		t.Errorf("expected ErrInvalidKeyLength, got: %v", err)
	}
}

func TestNewAESCBCCipher_InvalidIVRule(t *testing.T) {
	_, err := NewAESCBCCipher("4845727543536D5570716A3843596451", "invalid_rule")
	if err == nil {
		t.Error("expected error for invalid IV rule")
	}
}

func TestEncrypt_ProtocolExample(t *testing.T) {
	// 文档中的加密示例验证
	cipher, err := NewAESCBCCipher("4845727543536D5570716A3843596451", "last_16_bytes_of_key")
	if err != nil {
		t.Fatalf("NewAESCBCCipher() error: %v", err)
	}

	plaintext := []byte{0x00}
	expected := []byte{0x41, 0x6D, 0xF1, 0x63, 0x27, 0xFA, 0xAD, 0x87, 0x9C, 0x51, 0x86, 0x3D, 0xB1, 0x07, 0x53, 0x55}

	encrypted, err := cipher.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	if !bytes.Equal(encrypted, expected) {
		t.Errorf("Encrypt(0x00) = % X, expected % X", encrypted, expected)
	}
}

func TestDecrypt_ProtocolExample(t *testing.T) {
	cipher, err := NewAESCBCCipher("4845727543536D5570716A3843596451", "last_16_bytes_of_key")
	if err != nil {
		t.Fatalf("NewAESCBCCipher() error: %v", err)
	}

	ciphertext := []byte{0x41, 0x6D, 0xF1, 0x63, 0x27, 0xFA, 0xAD, 0x87, 0x9C, 0x51, 0x86, 0x3D, 0xB1, 0x07, 0x53, 0x55}
	expected := []byte{0x00}

	decrypted, err := cipher.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}

	if !bytes.Equal(decrypted, expected) {
		t.Errorf("Decrypt() = % X, expected % X", decrypted, expected)
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	cipher, err := NewAESCBCCipher("4845727543536D5570716A3843596451", "last_16_bytes_of_key")
	if err != nil {
		t.Fatalf("NewAESCBCCipher() error: %v", err)
	}

	plaintext := []byte("Hello, charging pile!")

	encrypted, err := cipher.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("round-trip failed: got %q, expected %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_EmptyData(t *testing.T) {
	cipher, _ := NewAESCBCCipher("4845727543536D5570716A3843596451", "last_16_bytes_of_key")

	encrypted, err := cipher.Encrypt([]byte{})
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}
	if encrypted != nil {
		t.Errorf("expected nil for empty plaintext, got %v", encrypted)
	}

	decrypted, err := cipher.Decrypt([]byte{})
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}
	if decrypted != nil {
		t.Errorf("expected nil for empty ciphertext, got %v", decrypted)
	}
}

func TestEncryptDecrypt_BlockAlignment(t *testing.T) {
	cipher, _ := NewAESCBCCipher("4845727543536D5570716A3843596451", "last_16_bytes_of_key")

	for _, length := range []int{1, 15, 16, 17, 31, 32, 33, 64, 100} {
		t.Run(string(rune(length)), func(t *testing.T) {
			plaintext := make([]byte, length)
			for i := range plaintext {
				plaintext[i] = byte(i % 256)
			}

			encrypted, err := cipher.Encrypt(plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error for length %d: %v", length, err)
			}
			if len(encrypted)%16 != 0 {
				t.Errorf("encrypted length %d not aligned to block size", len(encrypted))
			}

			decrypted, err := cipher.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt() error for length %d: %v", length, err)
			}
			if !bytes.Equal(decrypted, plaintext) {
				t.Errorf("round-trip failed for length %d", length)
			}
		})
	}
}

func TestDecrypt_InvalidLength(t *testing.T) {
	cipher, _ := NewAESCBCCipher("4845727543536D5570716A3843596451", "last_16_bytes_of_key")

	_, err := cipher.Decrypt([]byte{0x01, 0x02, 0x03})
	if err == nil {
		t.Error("expected error for non-aligned ciphertext")
	}
}

func TestPKCS7Pad(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		blockSize int
		expected  int
	}{
		{"exact block", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, 16, 32},
		{"one over block", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17}, 16, 32},
		{"single byte", []byte{0x01}, 16, 16},
		{"empty", []byte{}, 16, 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			padded := pkcs7Pad(tt.data, tt.blockSize)
			if len(padded) != tt.expected {
				t.Errorf("padded length = %d, expected %d", len(padded), tt.expected)
			}
			unpadded, err := pkcs7Unpad(padded, tt.blockSize)
			if err != nil {
				t.Fatalf("pkcs7Unpad() error: %v", err)
			}
			if !bytes.Equal(unpadded, tt.data) {
				t.Errorf("round-trip failed: got %v, expected %v", unpadded, tt.data)
			}
		})
	}
}

func TestGenerateRandomKey(t *testing.T) {
	key, err := GenerateRandomKey()
	if err != nil {
		t.Fatalf("GenerateRandomKey() error: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("key length = %d, expected 32", len(key))
	}
}

func TestGenerateRandomIV(t *testing.T) {
	iv, err := GenerateRandomIV()
	if err != nil {
		t.Fatalf("GenerateRandomIV() error: %v", err)
	}
	if len(iv) != 16 {
		t.Errorf("IV length = %d, expected 16", len(iv))
	}
}
