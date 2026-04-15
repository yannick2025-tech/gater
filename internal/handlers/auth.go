package handlers

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"fmt"
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
// 流程：生成13位随机数 → AES加密 → 回复给充电桩
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

	// AES-256-CBC加密13位随机数（使用固定密钥）
	encryptFn := ctx.Session.GetEncryptFn()
	encrypted, err := encryptFn(randomBytes)
	if err != nil {
		return fmt.Errorf("encrypt random bytes: %w", err)
	}

	// 构建回复帧
	replyHeader := types.MessageHeader{
		StartByte:  ctx.Proto.FrameConfig().StartByte,
		Version:    ctx.Proto.Version(),
		FuncCode:   types.FuncAuthRandom,
		PostNo:     ctx.PostNo,
		Charger:    ctx.Charger,
		EncryptFlag: 0x01,
	}

	ctx.Logger.Infof("[%s] auth: reply 0x0A with encrypted random bytes (len=%d)", ctx.Session.ID, len(encrypted))
	return ctx.Reply(replyHeader, encrypted)
}

// HandleAuthDataUpload 处理充电桩上报0x0B
// 流程：验证MD5密钥校验算法 → 更新认证状态 → 回复结果+当前时间
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

	return ctx.Reply(replyHeader, replyData)
}

// HandleKeyUpdateUpload 处理充电桩回复0x21（密钥更新结果）
func (h *AuthHandler) HandleKeyUpdateUpload(ctx *dispatcher.Context) error {
	keyUpdateMsg, ok := ctx.Message.(*msg.KeyUpdateReply)
	if !ok {
		return fmt.Errorf("expected KeyUpdateReply, got %T", ctx.Message)
	}

	if keyUpdateMsg.SecretUpdateStatus == 0 {
		ctx.Logger.Infof("[%s] key update SUCCESS for postNo=%d", ctx.Session.ID, ctx.PostNo)
	} else {
		ctx.Logger.Warnf("[%s] key update FAILED for postNo=%d, status=%d", ctx.Session.ID, ctx.PostNo, keyUpdateMsg.SecretUpdateStatus)
	}
	return nil
}

// computeAuthHash 计算认证校验哈希
// 算法：13位随机数倒序 + 16位固定密钥拼接为29字节 → BCD编码为58字节 → MD5 → 取前16字节 → 倒序
func computeAuthHash(randomKey []byte, fixedKey []byte) ([]byte, error) {
	// 1. 13位随机数倒序
	reversed := make([]byte, 13)
	for i := 0; i < 13; i++ {
		reversed[i] = randomKey[12-i]
	}

	// 2. 拼接: 倒序随机数(13) + 固定密钥(16) = 29字节
	combined := make([]byte, 0, 29)
	combined = append(combined, reversed...)
	combined = append(combined, fixedKey[:16]...)

	// 3. 当作BCD码，转换成58字节
	bcdBytes := bytesToBCD(combined)

	// 4. MD5
	hash := md5.Sum(bcdBytes)
	hash16 := hash[:16]

	// 5. 取前16字节倒序
	result := make([]byte, 16)
	for i := 0; i < 16; i++ {
		result[i] = hash16[15-i]
	}

	return result, nil
}

// bytesToBCD 将字节数组作为BCD码转换为双倍长度的字节数组
// 每个字节拆成两个BCD数字
func bytesToBCD(data []byte) []byte {
	result := make([]byte, len(data)*2)
	for i, b := range data {
		result[i*2] = (b >> 4) & 0x0F
		result[i*2+1] = b & 0x0F
	}
	return result
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
