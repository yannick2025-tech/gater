// Package crypto provides AES-256-CBC encryption and decryption.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

var (
	ErrInvalidKeyLength = errors.New("key must be exactly 32 bytes for AES-256")
	ErrInvalidIVLength  = errors.New("IV must be 16 bytes for AES-CBC")
	ErrPKCS7Unpad       = errors.New("invalid PKCS7 padding")
)

// AESCBCCipher AES-256-CBC 加密器
type AESCBCCipher struct {
	key []byte // 32-byte key
	iv  []byte // 16-byte IV
}

// NewAESCBCCipher 从 ASCII 密钥字符串创建 AES-256-CBC 加密器
// keyStr: 32字符 ASCII 字符串，每个字符的 ASCII 码作为一字节密钥
// ivRule: "last_16_bytes_of_key" 表示取密钥后16字符作为IV
//
// 示例: keyStr = "4845727543536D5570716A3843596451"
//
//	KEY = [52,56,52,53,55,50,55,53,52,51,53,51,54,68,53,53,55,48,55,49,54,65,51,56,52,51,53,57,54,52,53,49]
//	IV  = [55,48,55,49,54,65,51,56,52,51,53,57,54,52,53,49] (后16字符)
func NewAESCBCCipher(keyStr string, ivRule string) (*AESCBCCipher, error) {
	key := []byte(keyStr)
	if len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}

	var iv []byte
	switch ivRule {
	case "last_16_bytes_of_key":
		iv = make([]byte, 16)
		copy(iv, key[16:])
	default:
		return nil, fmt.Errorf("unsupported IV rule: %s", ivRule)
	}

	return &AESCBCCipher{key: key, iv: iv}, nil
}

// Encrypt AES-256-CBC + PKCS7 加密
func (c *AESCBCCipher) Encrypt(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, nil
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	padded := pkcs7Pad(plaintext, aes.BlockSize)

	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, c.iv)
	mode.CryptBlocks(ciphertext, padded)

	return ciphertext, nil
}

// Decrypt AES-256-CBC + PKCS7 解密
func (c *AESCBCCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, nil
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext length %d is not a multiple of block size %d", len(ciphertext), aes.BlockSize)
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, c.iv)
	mode.CryptBlocks(plaintext, ciphertext)

	return pkcs7Unpad(plaintext, aes.BlockSize)
}

// Key 返回密钥副本
func (c *AESCBCCipher) Key() []byte {
	cp := make([]byte, len(c.key))
	copy(cp, c.key)
	return cp
}

// IV 返回IV副本
func (c *AESCBCCipher) IV() []byte {
	cp := make([]byte, len(c.iv))
	copy(cp, c.iv)
	return cp
}

// pkcs7Pad PKCS7 填充
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	pad := make([]byte, padding)
	for i := range pad {
		pad[i] = byte(padding)
	}
	return append(data, pad...)
}

// pkcs7Unpad PKCS7 去填充
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize || padding > len(data) {
		return nil, ErrPKCS7Unpad
	}
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, ErrPKCS7Unpad
		}
	}
	return data[:len(data)-padding], nil
}

// GenerateRandomKey 生成32字节随机密钥
func GenerateRandomKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// GenerateRandomIV 生成16字节随机IV
func GenerateRandomIV() ([]byte, error) {
	iv := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	return iv, nil
}
