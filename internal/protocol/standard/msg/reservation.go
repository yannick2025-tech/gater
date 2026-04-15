// Package msg provides 0x07 reservation and 0x08 platform stop message definitions.
package msg

import "github.com/yannick2025-tech/nts-gater/internal/protocol/types"

// ==================== 0x07 预约充电 ====================

type ReservationDownload struct {
	ReservationNo      string `json:"reservationNo"`      // BCD[10]
	Operation          byte   `json:"operation"`           // 0预约 1取消
	ReservationDuration uint16 `json:"reservationDuration"` // 分钟
	CancelDuration     uint16 `json:"cancelDuration"`      // 分钟
}

func (m *ReservationDownload) Spec() types.MessageSpec { return MakeSpec(types.FuncReservation, types.DirectionDownload, "reservation_download", false, true) }

func (m *ReservationDownload) Decode(data []byte) error {
	off := 0; m.ReservationNo, off, _ = ReadBCD(data, off, 10)
	m.Operation, off, _ = ReadByte(data, off)
	m.ReservationDuration, off, _ = ReadUint16LE(data, off)
	m.CancelDuration, off, _ = ReadUint16LE(data, off); return nil
}

func (m *ReservationDownload) Encode() ([]byte, error) {
	buf := make([]byte, 15); off, _ := WriteBCD(buf, 0, m.ReservationNo, 10)
	off = WriteByte(buf, off, m.Operation); off = WriteUint16LE(buf, off, m.ReservationDuration)
	off = WriteUint16LE(buf, off, m.CancelDuration); return buf[:off], nil
}

func (m *ReservationDownload) Validate() []types.ValidationError { return nil }

func (m *ReservationDownload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"reservationNo": m.ReservationNo, "operation": m.Operation, "reservationDuration": m.ReservationDuration, "cancelDuration": m.CancelDuration}
}

type ReservationReply struct {
	ReservationNo string `json:"reservationNo"` // BCD[10]
	Operation     byte   `json:"operation"`
	ResultCode    byte   `json:"resultCode"` // 1成功 2已被预约 3充电中 4异常 5其他 6被占用
}

func (m *ReservationReply) Spec() types.MessageSpec { return MakeSpec(types.FuncReservation, types.DirectionReply, "reservation_reply", false, false) }

func (m *ReservationReply) Decode(data []byte) error {
	off := 0; m.ReservationNo, off, _ = ReadBCD(data, off, 10)
	m.Operation, off, _ = ReadByte(data, off); m.ResultCode, off, _ = ReadByte(data, off); return nil
}

func (m *ReservationReply) Encode() ([]byte, error) {
	buf := make([]byte, 12); off, _ := WriteBCD(buf, 0, m.ReservationNo, 10)
	off = WriteByte(buf, off, m.Operation); WriteByte(buf, off, m.ResultCode); return buf, nil
}

func (m *ReservationReply) Validate() []types.ValidationError { return nil }

func (m *ReservationReply) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"reservationNo": m.ReservationNo, "operation": m.Operation, "resultCode": m.ResultCode}
}

// ==================== 0x08 平台充电停止请求 ====================

type PlatformStopDownload struct {
	PlatformOrderNumber        string `json:"platformOrderNumber"`        // BCD[10]
	PileChargingFailureReason  byte   `json:"pileChargingFailureReason"`
}

func (m *PlatformStopDownload) Spec() types.MessageSpec { return MakeSpec(types.FuncPlatformStop, types.DirectionDownload, "platform_stop_download", false, false) }

func (m *PlatformStopDownload) Decode(data []byte) error {
	off := 0; m.PlatformOrderNumber, off, _ = ReadBCD(data, off, 10)
	m.PileChargingFailureReason, off, _ = ReadByte(data, off); return nil
}

func (m *PlatformStopDownload) Encode() ([]byte, error) {
	buf := make([]byte, 11); off, _ := WriteBCD(buf, 0, m.PlatformOrderNumber, 10)
	WriteByte(buf, off, m.PileChargingFailureReason); return buf, nil
}

func (m *PlatformStopDownload) Validate() []types.ValidationError { return nil }

func (m *PlatformStopDownload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"platformOrderNumber": m.PlatformOrderNumber, "pileChargingFailureReason": m.PileChargingFailureReason}
}
