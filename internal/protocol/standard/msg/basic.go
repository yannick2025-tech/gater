package msg

import (
	"fmt"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
)

// ==================== 0x01 心跳 ====================

// HeartbeatUpload 充电桩上报心跳
type HeartbeatUpload struct {
	Status        byte   `json:"status"`        // 状态码
	ErrorCode     uint16 `json:"errorCode"`     // 故障状态
	SignalQuality *byte  `json:"signalQuality"` // 信号质量，0xFF时为null
}

func (m *HeartbeatUpload) Spec() types.MessageSpec {
	return MakeSpec(types.FuncHeartbeat, types.DirectionUpload, "heartbeat_upload", false, true)
}
func (m *HeartbeatUpload) Decode(data []byte) error {
	if len(data) < 4 {
		return errInsufficientData(4, len(data))
	}
	off := 0
	m.Status, off, _ = ReadByte(data, off)
	m.ErrorCode, off, _ = ReadUint16LE(data, off)
	sq, _, _ := ReadByte(data, off)
	if sq == 0xFF {
		m.SignalQuality = nil
	} else {
		m.SignalQuality = &sq
	}
	return nil
}
func (m *HeartbeatUpload) Encode() ([]byte, error) {
	buf := make([]byte, 4)
	off := 0
	off = WriteByte(buf, off, m.Status)
	off = WriteUint16LE(buf, off, m.ErrorCode)
	if m.SignalQuality != nil {
		off = WriteByte(buf, off, *m.SignalQuality)
	} else {
		off = WriteByte(buf, off, 0xFF)
	}
	return buf[:off], nil
}
func (m *HeartbeatUpload) Validate() []types.ValidationError {
	var errs []types.ValidationError
	if _, ok := StatusCodes[m.Status]; !ok {
		errs = append(errs, types.ValidationError{Field: "status", Code: "FIELD_INVALID", Message: "invalid status code"})
	}
	return errs
}
func (m *HeartbeatUpload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{
		"status": m.Status, "errorCode": m.ErrorCode,
		"signalQuality": m.SignalQuality,
	}
}

// HeartbeatReply 平台回复心跳
type HeartbeatReply struct{}

func (m *HeartbeatReply) Spec() types.MessageSpec {
	return MakeSpec(types.FuncHeartbeat, types.DirectionReply, "heartbeat_reply", false, false)
}
func (m *HeartbeatReply) Decode(_ []byte) error { return nil }
func (m *HeartbeatReply) Encode() ([]byte, error) { return []byte{}, nil }
func (m *HeartbeatReply) Validate() []types.ValidationError { return nil }
func (m *HeartbeatReply) ToJSONMap() map[string]interface{} { return map[string]interface{}{} }

// ==================== 0x23 对时 ====================

// TimeSyncUpload 充电桩上报对时
type TimeSyncUpload struct{}

func (m *TimeSyncUpload) Spec() types.MessageSpec {
	return MakeSpec(types.FuncTimeSync, types.DirectionUpload, "time_sync_upload", false, true)
}
func (m *TimeSyncUpload) Decode(_ []byte) error { return nil }
func (m *TimeSyncUpload) Encode() ([]byte, error) { return []byte{}, nil }
func (m *TimeSyncUpload) Validate() []types.ValidationError { return nil }
func (m *TimeSyncUpload) ToJSONMap() map[string]interface{} { return map[string]interface{}{} }

// TimeSyncReply 平台回复对时
type TimeSyncReply struct {
	DateTime string `json:"dateTime"` // BCD[7] yyyymmddhhmmss
}

func (m *TimeSyncReply) Spec() types.MessageSpec {
	return MakeSpec(types.FuncTimeSync, types.DirectionReply, "time_sync_reply", false, false)
}
func (m *TimeSyncReply) Decode(data []byte) error {
	if len(data) < 7 {
		return errInsufficientData(7, len(data))
	}
	hex, _, _ := ReadBCD(data, 0, 7)
	// BCD格式: YYYYMMDDHHMMSS -> "2024-02-22 15:00:25"
	if len(hex) >= 14 {
		m.DateTime = fmt.Sprintf("%s-%s-%s %s:%s:%s",
			hex[0:4], hex[4:6], hex[6:8], hex[8:10], hex[10:12], hex[12:14])
	} else {
		m.DateTime = hex
	}
	return nil
}
func (m *TimeSyncReply) Encode() ([]byte, error) {
	buf := make([]byte, 7)
	off, _ := WriteBCD(buf, 0, m.DateTime, 7)
	return buf[:off], nil
}
func (m *TimeSyncReply) Validate() []types.ValidationError { return nil }
func (m *TimeSyncReply) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"dateTime": m.DateTime}
}
