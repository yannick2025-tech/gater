// Package msg 定义了接入认证和秘钥更新相关的消息结构体和编解码逻辑
package msg

import (
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
)

// ==================== 0x0A 接入认证-随机数 ====================

// Auth0AUpload 充电桩上报0x0A
type Auth0AUpload struct {
	B0x00 byte `json:"b0x00"`
}

func (m *Auth0AUpload) Spec() types.MessageSpec {
	return MakeSpec(types.FuncAuthRandom, types.DirectionUpload, "access_auth_random_upload", true, true)
}

func (m *Auth0AUpload) Decode(data []byte) error {
	if len(data) < 1 {
		return errInsufficientData(1, len(data))
	}
	m.B0x00 = data[0]
	return nil
}

func (m *Auth0AUpload) Encode() ([]byte, error) {
	return []byte{m.B0x00}, nil
}

func (m *Auth0AUpload) Validate() []types.ValidationError {
	var errs []types.ValidationError
	if m.B0x00 != 0x00 {
		errs = append(errs, types.ValidationError{Field: "b0x00", Code: "FIELD_INVALID", Message: "b0x00 must be 0x00"})
	}
	return errs
}

func (m *Auth0AUpload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"b0x00": m.B0x00}
}

// Auth0AReply 平台回复0x0A
type Auth0AReply struct {
	RandomKey []byte `json:"bodyT"` // 13字节随机数，JSON中为base64
}

func (m *Auth0AReply) Spec() types.MessageSpec {
	return MakeSpec(types.FuncAuthRandom, types.DirectionReply, "access_auth_random_reply", true, false)
}

func (m *Auth0AReply) Decode(data []byte) error {
	if len(data) < 13 {
		return errInsufficientData(13, len(data))
	}
	m.RandomKey = make([]byte, 13)
	copy(m.RandomKey, data[:13])
	return nil
}

func (m *Auth0AReply) Encode() ([]byte, error) {
	if len(m.RandomKey) != 13 {
		return nil, errFieldLength("randomKey", 13, len(m.RandomKey))
	}
	result := make([]byte, 13)
	copy(result, m.RandomKey)
	return result, nil
}

func (m *Auth0AReply) Validate() []types.ValidationError { return nil }

func (m *Auth0AReply) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"bodyT": m.RandomKey}
}

// ==================== 0x0B 接入认证-加密数据 ====================

// Auth0BUpload 充电桩上报0x0B
type Auth0BUpload struct {
	Md5Sum           []byte `json:"md5Sum"`           // 16字节
	FirmwareType     uint16 `json:"firmwareType"`     // 固件类型号
	FirmwareVersion  uint16 `json:"firmwareVersion"`  // 固件版本号
	Sim              string `json:"sim"`              // BCD[13] SIM卡号
	ConfigVersion    string `json:"configVersion"`    // ASCII[32] 配置版本
	FaultCodeVersion uint16 `json:"faultCodeVersion"` // 故障码版本
}

func (m *Auth0BUpload) Spec() types.MessageSpec {
	return MakeSpec(types.FuncAuthEncrypted, types.DirectionUpload, "access_auth_encrypted_upload", true, true)
}

func (m *Auth0BUpload) Decode(data []byte) error {
	off := 0
	var err error
	m.Md5Sum, off, err = ReadBytes(data, off, 16)
	if err != nil {
		return err
	}
	m.FirmwareType, off, err = ReadUint16LE(data, off)
	if err != nil {
		return err
	}
	m.FirmwareVersion, off, err = ReadUint16LE(data, off)
	if err != nil {
		return err
	}
	m.Sim, off, err = ReadBCD(data, off, 13)
	if err != nil {
		return err
	}
	m.ConfigVersion, off, err = ReadASCII(data, off, 32)
	if err != nil {
		return err
	}
	m.FaultCodeVersion, off, err = ReadUint16LE(data, off)
	if err != nil {
		return err
	}
	return nil
}

func (m *Auth0BUpload) Encode() ([]byte, error) {
	buf := make([]byte, 16+2+2+13+32+2)
	off := 0
	copy(buf[off:], m.Md5Sum)
	off += 16
	off = WriteUint16LE(buf, off, m.FirmwareType)
	off = WriteUint16LE(buf, off, m.FirmwareVersion)
	bcdOff, _ := WriteBCD(buf, off, m.Sim, 13)
	off = bcdOff
	off = WriteASCII(buf, off, m.ConfigVersion, 32)
	off = WriteUint16LE(buf, off, m.FaultCodeVersion)
	return buf[:off], nil
}

func (m *Auth0BUpload) Validate() []types.ValidationError {
	var errs []types.ValidationError
	if m.FirmwareVersion < 200 {
		errs = append(errs, types.ValidationError{Field: "firmwareVersion", Code: "FIELD_INVALID", Message: "firmwareVersion must >= 200"})
	}
	return errs
}

func (m *Auth0BUpload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{
		"md5Sum": m.Md5Sum, "firmwareType": m.FirmwareType,
		"firmwareVersion": m.FirmwareVersion, "sim": m.Sim,
		"configVersion": m.ConfigVersion, "faultCodeVersion": m.FaultCodeVersion,
	}
}

// Auth0BReply 平台回复0x0B
type Auth0BReply struct {
	AuthStatus byte   `json:"authStatus"` // 0成功 1失败
	Time       []byte `json:"time"`       // 6字节 UTC时间
}

func (m *Auth0BReply) Spec() types.MessageSpec {
	return MakeSpec(types.FuncAuthEncrypted, types.DirectionReply, "access_auth_encrypted_reply", true, false)
}

func (m *Auth0BReply) Decode(data []byte) error {
	if len(data) < 7 {
		return errInsufficientData(7, len(data))
	}
	m.AuthStatus = data[0]
	m.Time = make([]byte, 6)
	copy(m.Time, data[1:7])
	return nil
}

func (m *Auth0BReply) Encode() ([]byte, error) {
	buf := make([]byte, 7)
	buf[0] = m.AuthStatus
	copy(buf[1:], m.Time)
	return buf, nil
}

func (m *Auth0BReply) Validate() []types.ValidationError {
	var errs []types.ValidationError
	if m.AuthStatus > 1 {
		errs = append(errs, types.ValidationError{Field: "authStatus", Code: "FIELD_INVALID", Message: "authStatus must be 0 or 1"})
	}
	return errs
}

func (m *Auth0BReply) ToJSONMap() map[string]interface{} {
	// time转为毫秒时间戳，与示例JSON一致
	return map[string]interface{}{"authStatus": m.AuthStatus, "time": m.Time}
}

// ==================== 0x21 秘钥更新 ====================

// KeyUpdateDownload 平台下发0x21
type KeyUpdateDownload struct {
	OriginalKey []byte `json:"originalKey"` // 16字节原始密钥
	NewAesKey   []byte `json:"newAesKey"`   // 16字节新密钥
}

func (m *KeyUpdateDownload) Spec() types.MessageSpec {
	return MakeSpec(types.FuncKeyUpdate, types.DirectionDownload, "key_update_download", true, true)
}

func (m *KeyUpdateDownload) Decode(data []byte) error {
	if len(data) < 32 {
		return errInsufficientData(32, len(data))
	}
	m.OriginalKey = make([]byte, 16)
	m.NewAesKey = make([]byte, 16)
	copy(m.OriginalKey, data[:16])
	copy(m.NewAesKey, data[16:32])
	return nil
}

func (m *KeyUpdateDownload) Encode() ([]byte, error) {
	buf := make([]byte, 32)
	copy(buf[:16], m.OriginalKey)
	copy(buf[16:], m.NewAesKey)
	return buf, nil
}

func (m *KeyUpdateDownload) Validate() []types.ValidationError { return nil }

func (m *KeyUpdateDownload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"newAesKey": fmtHex(m.NewAesKey)}
}

// KeyUpdateReply 充电桩回复0x21
type KeyUpdateReply struct {
	NewAesKey          []byte `json:"newAesKey"`          // 16字节
	SecretUpdateStatus byte   `json:"secretUpdateStatus"` // 0成功 1失败
}

func (m *KeyUpdateReply) Spec() types.MessageSpec {
	return MakeSpec(types.FuncKeyUpdate, types.DirectionReply, "key_update_reply", true, false)
}

func (m *KeyUpdateReply) Decode(data []byte) error {
	if len(data) < 17 {
		return errInsufficientData(17, len(data))
	}
	m.NewAesKey = make([]byte, 16)
	copy(m.NewAesKey, data[:16])
	m.SecretUpdateStatus = data[16]
	return nil
}

func (m *KeyUpdateReply) Encode() ([]byte, error) {
	buf := make([]byte, 17)
	copy(buf[:16], m.NewAesKey)
	buf[16] = m.SecretUpdateStatus
	return buf, nil
}

func (m *KeyUpdateReply) Validate() []types.ValidationError {
	var errs []types.ValidationError
	if m.SecretUpdateStatus > 1 {
		errs = append(errs, types.ValidationError{Field: "secretUpdateStatus", Code: "FIELD_INVALID", Message: "must be 0 or 1"})
	}
	return errs
}

func (m *KeyUpdateReply) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"newAesKey": fmtHex(m.NewAesKey), "secretUpdateStatus": m.SecretUpdateStatus}
}
