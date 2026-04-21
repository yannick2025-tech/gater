// Package handlers provides authentication, key update and basic business handlers.
package handlers

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/yannick2025-tech/nts-gater/internal/dispatcher"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/codec"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/crypto"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard/msg"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	"github.com/yannick2025-tech/nts-gater/internal/session"
	logging "github.com/yannick2025-tech/gwc-logging"
)

// Register 注册所有业务处理器
func Register(dp *dispatcher.Dispatcher, sessMgr *session.SessionManager, logger logging.Logger) {
	auth := NewAuthHandler(sessMgr, logger)
	dp.RegisterFunc(types.FuncAuthRandom, types.DirectionUpload, auth.HandleAuthRandomUpload)
	dp.RegisterFunc(types.FuncAuthEncrypted, types.DirectionUpload, auth.HandleAuthDataUpload)
	dp.RegisterFunc(types.FuncKeyUpdate, types.DirectionUpload, auth.HandleKeyUpdateUpload)

	basic := NewBasicHandler(logger)
	dp.RegisterFunc(types.FuncHeartbeat, types.DirectionUpload, basic.HandleHeartbeat)
	dp.RegisterFunc(types.FuncTimeSync, types.DirectionUpload, basic.HandleTimeSync)

	registerChargingHandlers(dp, logger)
	registerMiscHandlers(dp, logger)
}

// ==================== 认证处理器 ====================

// AuthHandler 认证业务处理器
type AuthHandler struct {
	sessMgr *session.SessionManager
	logger  logging.Logger
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(sessMgr *session.SessionManager, logger logging.Logger) *AuthHandler {
	return &AuthHandler{sessMgr: sessMgr, logger: logger}
}

// HandleAuthRandomUpload 处理充电桩上报0x0A
// 流程：生成13位随机数 → 回复给充电桩（FrameEncoder会自动加密，不要手动加密）
func (h *AuthHandler) HandleAuthRandomUpload(ctx *dispatcher.Context) error {
	// 生成13位随机数
	randomBytes := make([]byte, 13)
	if _, err := rand.Read(randomBytes); err != nil {
		return fmt.Errorf("generate random bytes: %w", err)
	}

	// 保存到会话（用于后续0x0B校验）
	ctx.Session.SetRandomKey(randomBytes)
	ctx.Session.SetAuthState(session.AuthPending)

	ctx.Logger.Infof("[%s] auth: generated 13 random bytes for postNo=%d", ctx.Session.ID, ctx.PostNo)

	// 构建回复帧：传入明文13字节随机数，EncryptFlag=0x01，FrameEncoder会自动加密
	replyHeader := types.MessageHeader{
		StartByte:   ctx.Proto.FrameConfig().StartByte,
		Version:     ctx.Proto.Version(),
		FuncCode:    types.FuncAuthRandom,
		PostNo:      ctx.PostNo,
		Charger:     ctx.Charger,
		EncryptFlag: 0x01,
	}

	ctx.Logger.Infof("[%s] auth: reply 0x0A with random bytes (len=%d)", ctx.Session.ID, len(randomBytes))
	return ctx.Reply(replyHeader, randomBytes)
}

// HandleAuthDataUpload 处理充电桩上报0x0B
// 流程：验证MD5密钥校验算法 → 更新认证状态 → 回复结果+当前时间 → 认证成功则下发0x21密钥更新
func (h *AuthHandler) HandleAuthDataUpload(ctx *dispatcher.Context) error {
	authMsg, ok := ctx.Message.(*msg.Auth0BUpload)
	if !ok {
		return fmt.Errorf("expected AuthDataUpload, got %T", ctx.Message)
	}

	// 获取保存的13位随机数
	randomKey := ctx.Session.GetRandomKey()

	if len(randomKey) != 13 {
		return fmt.Errorf("session has no random key (len=%d), auth flow not started", len(randomKey))
	}

	// 校验密钥算法
	expectedHash, err := computeAuthHash(randomKey, ctx.Session.FixedCipher.Key())
	if err != nil {
		return fmt.Errorf("compute auth hash: %w", err)
	}

	// 比对充电桩上报的MD5值
	authSuccess := string(authMsg.Md5Sum) == string(expectedHash)

	if authSuccess {
		ctx.Session.SetAuthState(session.Authenticated)
		ctx.Logger.Infof("[%s] auth: 0x0B authentication SUCCESS for postNo=%d", ctx.Session.ID, ctx.PostNo)
	} else {
		ctx.Session.SetAuthState(session.AuthNone)
		ctx.Logger.Warnf("[%s] auth: 0x0B authentication FAILED for postNo=%d, expected=% X, got=% X",
			ctx.Session.ID, ctx.PostNo, expectedHash, authMsg.Md5Sum)
	}

	// 构建回复：认证状态 + 当前时间
	replyData := make([]byte, 7)
	if authSuccess {
		replyData[0] = 0x00 // 成功
	} else {
		replyData[0] = 0x01 // 失败
	}
	// 当前UTC时间: BCD[6] yy-mm-dd-hh-mm-ss (base: 2000年)
	now := time.Now().UTC()
	replyData[1] = byte(now.Year() - 2000)
	replyData[2] = byte(now.Month())
	replyData[3] = byte(now.Day())
	replyData[4] = byte(now.Hour())
	replyData[5] = byte(now.Minute())
	replyData[6] = byte(now.Second())

	replyHeader := types.MessageHeader{
		StartByte:  ctx.Proto.FrameConfig().StartByte,
		Version:    ctx.Proto.Version(),
		FuncCode:   types.FuncAuthEncrypted,
		PostNo:     ctx.PostNo,
		Charger:    ctx.Charger,
		EncryptFlag: 0x01,
	}

	if err := ctx.Reply(replyHeader, replyData); err != nil {
		return err
	}

	// 认证成功后，主动下发0x21密钥更新
	if authSuccess && ctx.SendDownload != nil {
		go h.sendKeyUpdate(ctx)
	}

	return nil
}

// sendKeyUpdate 下发0x21密钥更新
// 密钥生成流程：13字节随机数 → 固定密钥AES加密 → 16字节密文 → hex编码32字符ASCII（即新密钥）→ BCD16 → 倒序 → 下发
func (h *AuthHandler) sendKeyUpdate(ctx *dispatcher.Context) {
	// 1. 生成13字节随机数
	randomBytes := make([]byte, 13)
	if _, err := rand.Read(randomBytes); err != nil {
		ctx.Logger.Errorf("[%s] auth: generate random bytes for 0x21 failed: %v", ctx.Session.ID, err)
		return
	}

	// 2. 用固定密钥AES-256-CBC-PKCS7加密13字节随机数 → 16字节密文
	encryptedKey, err := ctx.Session.FixedCipher.Encrypt(randomBytes)
	if err != nil {
		ctx.Logger.Errorf("[%s] auth: encrypt random bytes for 0x21 failed: %v", ctx.Session.ID, err)
		return
	}

	// 3. 16字节密文hex编码 → 32字符大写ASCII字符串，这就是新密钥
	//    后续所有消息用此32字符作为AES-256的KEY，IV取后16字符
	newKeyStr := strings.ToUpper(hex.EncodeToString(encryptedKey))

	// 4. 32字符hex字符串 → BCD16编码 → 16字节 → 倒序 → 下发给充电桩
	//    桩收到后：倒序 → BCD16转32字符 → 得到新密钥
	newKeyBCD := reverseBytes(hexStrToBCD16(newKeyStr))

	// 5. 原始密钥：协议规定固定使用0xFF填充16字节
	originalKey := make([]byte, 16)
	for i := range originalKey {
		originalKey[i] = 0xFF
	}

	// 6. 保存待生效的会话密钥（0x21回复成功后切换）
	ctx.Session.SetPendingSessionKey(newKeyStr)

	ctx.Logger.Infof("[%s] auth: sending 0x21 key update for postNo=%d, newKey=%s", ctx.Session.ID, ctx.PostNo, newKeyStr)

	// 7. 构建并发送0x21密钥更新消息
	keyUpdateMsg := &msg.KeyUpdateDownload{
		OriginalKey: originalKey,
		NewAesKey:   newKeyBCD,
	}
	if err := ctx.SendDownload(keyUpdateMsg); err != nil {
		ctx.Logger.Errorf("[%s] auth: send 0x21 key update failed: %v", ctx.Session.ID, err)
		return
	}
	ctx.Logger.Infof("[%s] auth: 0x21 key update sent for postNo=%d", ctx.Session.ID, ctx.PostNo)
}

// HandleKeyUpdateUpload 处理充电桩回复0x21（密钥更新结果）
func (h *AuthHandler) HandleKeyUpdateUpload(ctx *dispatcher.Context) error {
	keyUpdateMsg, ok := ctx.Message.(*msg.KeyUpdateReply)
	if !ok {
		return fmt.Errorf("expected KeyUpdateReply, got %T", ctx.Message)
	}

	if keyUpdateMsg.SecretUpdateStatus == 0 {
		// 密钥更新成功：切换到会话密钥
		pendingKey := ctx.Session.GetPendingSessionKey()
		if pendingKey != "" {
			if err := h.sessMgr.SetSessionKey(ctx.Session.ID, pendingKey); err != nil {
				ctx.Logger.Errorf("[%s] key update SUCCESS but failed to apply session key: %v", ctx.Session.ID, err)
			} else {
				ctx.Session.SetPendingSessionKey("")
				ctx.Logger.Infof("[%s] key update SUCCESS for postNo=%d, switched to session key", ctx.Session.ID, ctx.PostNo)
			}
		} else {
			ctx.Logger.Infof("[%s] key update SUCCESS for postNo=%d (no pending key)", ctx.Session.ID, ctx.PostNo)
		}
	} else {
		ctx.Logger.Warnf("[%s] key update FAILED for postNo=%d, status=%d", ctx.Session.ID, ctx.PostNo, keyUpdateMsg.SecretUpdateStatus)
	}
	return nil
}

// computeAuthHash 计算认证校验哈希
// 算法：13位随机数倒序 + 16位固定密钥(hex解码)拼接为29字节 → BCD编码为58字节 → MD5 → 取前16字节 → 倒序
func computeAuthHash(randomKey []byte, fixedKey []byte) ([]byte, error) {
	// 1. 13位随机数倒序
	reversed := make([]byte, 13)
	for i := 0; i < 13; i++ {
		reversed[i] = randomKey[12-i]
	}

	// 2. 固定密钥hex解码为16字节原始数据
	fixedKeyHex, err := hex.DecodeString(string(fixedKey))
	if err != nil {
		return nil, fmt.Errorf("hex decode fixed key: %w", err)
	}

	// 3. 拼接: 倒序随机数(13) + 固定密钥hex(16) = 29字节
	combined := make([]byte, 0, 29)
	combined = append(combined, reversed...)
	combined = append(combined, fixedKeyHex...)

	// 4. 当作BCD码，转换成58字节
	bcdBytes := bytesToBCD(combined)

	// 5. MD5
	hash := md5.Sum(bcdBytes)
	hash16 := hash[:16]

	// 6. 取前16字节倒序
	result := make([]byte, 16)
	for i := 0; i < 16; i++ {
		result[i] = hash16[15-i]
	}

	return result, nil
}

// bytesToBCD 将字节数组转成BCD码（即hex编码为大写ASCII字符串）
// 协议中"转成BCD码"的实际含义：每字节→2个hex字符→大写ASCII字节
// 例如: [0x4A, 0x43] → "4A43" → [0x34, 0x41, 0x34, 0x33]
func bytesToBCD(data []byte) []byte {
	return []byte(strings.ToUpper(hex.EncodeToString(data)))
}

// ==================== 基础处理器 ====================

// BasicHandler 心跳/对时处理器
type BasicHandler struct {
	logger logging.Logger
}

// NewBasicHandler 创建基础处理器
func NewBasicHandler(logger logging.Logger) *BasicHandler {
	return &BasicHandler{logger: logger}
}

// HandleHeartbeat 处理充电桩上报0x01心跳
// 回复空包
func (h *BasicHandler) HandleHeartbeat(ctx *dispatcher.Context) error {
	replyHeader := types.MessageHeader{
		StartByte:  ctx.Proto.FrameConfig().StartByte,
		Version:    ctx.Proto.Version(),
		FuncCode:   types.FuncHeartbeat,
		PostNo:     ctx.PostNo,
		Charger:    ctx.Charger,
		EncryptFlag: 0x00,
	}
	return ctx.Reply(replyHeader, nil)
}

// HandleTimeSync 处理充电桩上报0x23对时
// 回复当前UTC时间
func (h *BasicHandler) HandleTimeSync(ctx *dispatcher.Context) error {
	now := time.Now().UTC()
	replyData := make([]byte, 7)
	replyData[0] = byte(now.Year() - 2000)
	replyData[1] = byte(now.Month())
	replyData[2] = byte(now.Day())
	replyData[3] = byte(now.Hour())
	replyData[4] = byte(now.Minute())
	replyData[5] = byte(now.Second())
	replyData[6] = 0x00 // 毫秒(低) 保留

	replyHeader := types.MessageHeader{
		StartByte:  ctx.Proto.FrameConfig().StartByte,
		Version:    ctx.Proto.Version(),
		FuncCode:   types.FuncTimeSync,
		PostNo:     ctx.PostNo,
		Charger:    ctx.Charger,
		EncryptFlag: 0x00,
	}
	return ctx.Reply(replyHeader, replyData)
}

// ==================== 辅助函数 ====================

// GenerateSessionKey 生成32字节随机会话密钥
func GenerateSessionKey() (string, error) {
	keyBytes, err := crypto.GenerateRandomKey()
	if err != nil {
		return "", err
	}
	return string(keyBytes), nil
}

// Uint16ToBytes uint16转小端字节
func Uint16ToBytes(v uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	return b
}

// Uint32ToBytes uint32转小端字节
func Uint32ToBytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

// CalcChecksumForReply 计算数据域校验和
func CalcChecksumForReply(data []byte) byte {
	return codec.CalcChecksum(data)
}

// hexStrToBCD16 将32字符hex字符串编码为16字节BCD
// 协议中0x21密钥更新的"新密钥"字段：32字符hex → 每2字符合并为1字节 → 16字节
// 例如: "35727013372387479964400522855388" → [0x35, 0x72, 0x70, 0x13, ...]
func hexStrToBCD16(hexStr string) []byte {
	result := make([]byte, 16)
	for i := 0; i < 16 && i*2+1 < len(hexStr); i++ {
		b := hexStr[i*2 : i*2+2]
		var v byte
		for _, c := range b {
			v <<= 4
			if c >= '0' && c <= '9' {
				v |= byte(c - '0')
			} else if c >= 'A' && c <= 'F' {
				v |= byte(c - 'A' + 10)
			} else if c >= 'a' && c <= 'f' {
				v |= byte(c - 'a' + 10)
			}
		}
		result[i] = v
	}
	return result
}

// reverseBytes 字节数组倒序
func reverseBytes(data []byte) []byte {
	n := len(data)
	result := make([]byte, n)
	for i := 0; i < n; i++ {
		result[i] = data[n-1-i]
	}
	return result
}
