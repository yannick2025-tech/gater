// Package msg provides 0x22 billing rules message definitions.
package msg

import "github.com/yannick2025-tech/nts-gater/internal/protocol/types"

// ==================== 0x22 分时段计费规则 ====================

type BillingRulesDownload struct {
	FeeNum  byte      `json:"feeNum"`
	ListFee []FeeItem `json:"listFee"`
}

func (m *BillingRulesDownload) Spec() types.MessageSpec { return MakeSpec(types.FuncBillingRules, types.DirectionDownload, "billing_rules_download", false, true) }

func (m *BillingRulesDownload) Decode(data []byte) error {
	off := 0; m.FeeNum, off, _ = ReadByte(data, off)
	m.ListFee = make([]FeeItem, m.FeeNum)
	for i := 0; i < int(m.FeeNum); i++ {
		var fi FeeItem
		fi.Hour, off, _ = ReadByte(data, off); fi.Min, off, _ = ReadByte(data, off)
		pf, off, _ := ReadBytes(data, off, 3); fi.PowerFee = bytes3ToUint32(pf)
		sf, off, _ := ReadBytes(data, off, 3); fi.SvcFee = bytes3ToUint32(sf)
		fi.Type, off, _ = ReadByte(data, off); fi.LimitedP, off, _ = ReadUint16LE(data, off)
		m.ListFee[i] = fi
	}
	return nil
}

func (m *BillingRulesDownload) Encode() ([]byte, error) {
	buf := make([]byte, 1+int(m.FeeNum)*9)
	off := WriteByte(buf, 0, m.FeeNum)
	for _, fi := range m.ListFee {
		off = WriteByte(buf, off, fi.Hour); off = WriteByte(buf, off, fi.Min)
		off += copy(buf[off:], uint32ToBytes3(fi.PowerFee))
		off += copy(buf[off:], uint32ToBytes3(fi.SvcFee))
		off = WriteByte(buf, off, fi.Type); off = WriteUint16LE(buf, off, fi.LimitedP)
	}
	return buf[:off], nil
}

func (m *BillingRulesDownload) Validate() []types.ValidationError { return nil }

func (m *BillingRulesDownload) ToJSONMap() map[string]interface{} {
	fees := make([]map[string]interface{}, len(m.ListFee))
	for i, f := range m.ListFee {
		fees[i] = map[string]interface{}{"hour": f.Hour, "min": f.Min, "powerFee": f.PowerFee, "svcFee": f.SvcFee, "type": f.Type, "limitedP": f.LimitedP}
	}
	return map[string]interface{}{"feeNum": m.FeeNum, "listFee": fees}
}

type BillingRulesReply struct{ ResultCode byte `json:"resultCode"` }

func (m *BillingRulesReply) Spec() types.MessageSpec { return MakeSpec(types.FuncBillingRules, types.DirectionReply, "billing_rules_reply", false, false) }

func (m *BillingRulesReply) Decode(data []byte) error { if len(data) < 1 { return errInsufficientData(1, len(data)) }; m.ResultCode = data[0]; return nil }

func (m *BillingRulesReply) Encode() ([]byte, error) { return []byte{m.ResultCode}, nil }

func (m *BillingRulesReply) Validate() []types.ValidationError { return nil }

func (m *BillingRulesReply) ToJSONMap() map[string]interface{} { return map[string]interface{}{"resultCode": m.ResultCode} }

// ==================== 0x16 占位订单 ====================

type OccupancyUpload struct {
	ChargeOrderNo    string `json:"chargeOrderNo"`    // BCD[10]
	DeviceOrderNo    string `json:"deviceOrderNo"`    // BCD[10]
	Type             byte   `json:"type"`
	StartElectricMeter uint64 `json:"startElectricMeter"` // BYTE[5]
	StopElectricMeter  uint64 `json:"stopElectricMeter"`  // BYTE[5]
	StopReason       byte   `json:"stopReason"`
	Time             string `json:"time"` // BCD[7]
}

func (m *OccupancyUpload) Spec() types.MessageSpec { return MakeSpec(types.FuncOccupancy, types.DirectionUpload, "occupancy_upload", false, true) }

func (m *OccupancyUpload) Decode(data []byte) error {
	off := 0; m.ChargeOrderNo, off, _ = ReadBCD(data, off, 10)
	m.DeviceOrderNo, off, _ = ReadBCD(data, off, 10); m.Type, off, _ = ReadByte(data, off)
	se, off, _ := ReadBytes(data, off, 5); m.StartElectricMeter = bytesToUint64(se)
	ee, off, _ := ReadBytes(data, off, 5); m.StopElectricMeter = bytesToUint64(ee)
	m.StopReason, off, _ = ReadByte(data, off); m.Time, off, _ = ReadBCD(data, off, 7) // BCD[7] UTC时间
	return nil
}

func (m *OccupancyUpload) Encode() ([]byte, error) { return nil, nil } // TODO
func (m *OccupancyUpload) Validate() []types.ValidationError { return nil }

func (m *OccupancyUpload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"chargeOrderNo": m.ChargeOrderNo, "deviceOrderNo": m.DeviceOrderNo, "type": m.Type, "stopReason": m.StopReason, "time": m.Time}
}

type OccupancyReply struct {
	ChargeOrderNo string `json:"chargeOrderNo"` // BCD[10]
	ResponseCode  byte   `json:"responseCode"`
}

func (m *OccupancyReply) Spec() types.MessageSpec { return MakeSpec(types.FuncOccupancy, types.DirectionReply, "occupancy_reply", false, false) }

func (m *OccupancyReply) Decode(data []byte) error {
	off := 0; m.ChargeOrderNo, off, _ = ReadBCD(data, off, 10)
	m.ResponseCode, off, _ = ReadByte(data, off); return nil
}

func (m *OccupancyReply) Encode() ([]byte, error) {
	buf := make([]byte, 11); off, _ := WriteBCD(buf, 0, m.ChargeOrderNo, 10)
	WriteByte(buf, off, m.ResponseCode); return buf, nil
}

func (m *OccupancyReply) Validate() []types.ValidationError { return nil }

func (m *OccupancyReply) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"chargeOrderNo": m.ChargeOrderNo, "responseCode": m.ResponseCode}
}

// ==================== 0x28 屏显模式 ====================

type ScreenDisplayDownload struct {
	ModeNumber     byte   `json:"modeNumber"`
	DisplayContent []byte `json:"displayContent"`
}

func (m *ScreenDisplayDownload) Spec() types.MessageSpec { return MakeSpec(types.FuncScreenDisplay, types.DirectionDownload, "screen_display_download", false, true) }

func (m *ScreenDisplayDownload) Decode(data []byte) error {
	if len(data) < 1 { return errInsufficientData(1, len(data)) }
	m.ModeNumber = data[0]
	if len(data) > 1 { m.DisplayContent = data[1:] }
	return nil
}

func (m *ScreenDisplayDownload) Encode() ([]byte, error) {
	buf := make([]byte, 1+len(m.DisplayContent))
	buf[0] = m.ModeNumber; copy(buf[1:], m.DisplayContent); return buf, nil
}

func (m *ScreenDisplayDownload) Validate() []types.ValidationError { return nil }

func (m *ScreenDisplayDownload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"modeNumber": m.ModeNumber, "displayContent": m.DisplayContent}
}

type ScreenDisplayReply struct{ ResultCode byte `json:"resultCode"` }

func (m *ScreenDisplayReply) Spec() types.MessageSpec { return MakeSpec(types.FuncScreenDisplay, types.DirectionReply, "screen_display_reply", false, false) }

func (m *ScreenDisplayReply) Decode(data []byte) error { if len(data) < 1 { return errInsufficientData(1, len(data)) }; m.ResultCode = data[0]; return nil }

func (m *ScreenDisplayReply) Encode() ([]byte, error) { return []byte{m.ResultCode}, nil }

func (m *ScreenDisplayReply) Validate() []types.ValidationError { return nil }

func (m *ScreenDisplayReply) ToJSONMap() map[string]interface{} { return map[string]interface{}{"resultCode": m.ResultCode} }
