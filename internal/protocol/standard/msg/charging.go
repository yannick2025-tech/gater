package msg

import (
	"encoding/binary"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
)

// ==================== 公共类型 ====================

// FeeItem 计费项
type FeeItem struct {
	Hour     byte   `json:"hour"`     // BCD[1]
	Min      byte   `json:"min"`      // BCD[1]
	PowerFee uint32 `json:"powerFee"` // BYTE[3] 4位小数
	SvcFee   uint32 `json:"svcFee"`   // BYTE[3] 4位小数
	Type     byte   `json:"type"`     // 尖峰谷平
	LimitedP uint16 `json:"limitedP"` // 限制功率
}

// ==================== 字节转换辅助 ====================

func bytesToUint64(b []byte) uint64 {
	var v uint64
	for i := len(b) - 1; i >= 0; i-- {
		v = v<<8 | uint64(b[i])
	}
	return v
}

func uint64ToBytes(v uint64, size int) []byte {
	b := make([]byte, size)
	for i := 0; i < size; i++ {
		b[i] = byte(v >> (i * 8))
	}
	return b
}

func bytes3ToUint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16
}

func uint32ToBytes3(v uint32) []byte {
	return []byte{byte(v), byte(v >> 8), byte(v >> 16)}
}

// ==================== StartupTypes 启动类型副本 ====================

var StartupTypes = map[byte]string{
	1: "远程鉴权-扫码", 2: "远程鉴权-刷卡", 3: "本地鉴权-刷卡",
	4: "本地鉴权-按钮或屏幕启动", 5: "本地鉴权-插枪启动", 6: "远程鉴权-命令",
	7: "本地鉴权-桩满足条件启动", 11: "远程鉴权-VIN",
}

// ==================== 0x03 平台充电启动请求 ====================

// PlatformStartDownload 平台下发启动充电
type PlatformStartDownload struct {
	StartupType          byte   `json:"startupType"`
	AuthenticationNumber string `json:"authenticationNumber"` // ASCII[50]
}

func (m *PlatformStartDownload) Spec() types.MessageSpec {
	return MakeSpec(types.FuncPlatformStart, types.DirectionDownload, "platform_start_download", false, false)
}
func (m *PlatformStartDownload) Decode(data []byte) error {
	off := 0
	m.StartupType, off, _ = ReadByte(data, off)
	m.AuthenticationNumber, off, _ = ReadASCII(data, off, 50)
	return nil
}
func (m *PlatformStartDownload) Encode() ([]byte, error) {
	buf := make([]byte, 1+50)
	off := 0
	off = WriteByte(buf, off, m.StartupType)
	off = WriteASCII(buf, off, m.AuthenticationNumber, 50)
	return buf[:off], nil
}
func (m *PlatformStartDownload) Validate() []types.ValidationError {
	var errs []types.ValidationError
	if _, ok := StartupTypes[m.StartupType]; !ok {
		errs = append(errs, types.ValidationError{Field: "startupType", Code: "FIELD_INVALID", Message: "invalid startup type"})
	}
	return errs
}
func (m *PlatformStartDownload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"startupType": m.StartupType, "authenticationNumber": m.AuthenticationNumber}
}

// ==================== 0x04 桩充电启动请求 ====================

// ChargerStartUpload 充电桩上报充电启动请求
type ChargerStartUpload struct {
	DeviceOrderNo             string `json:"deviceOrderNo"`              // BCD[10]
	StartupType               byte   `json:"startupType"`
	StartingElectricity       uint64 `json:"startingElectricity"`        // BYTE[5] 4位小数
	PileChargingFailureReason byte   `json:"pileChargingFailureReason"`
	AuthenticationNumber      string `json:"authenticationNumber"`       // ASCII[50]
	Time                      string `json:"time"`                       // BCD[7]
}

func (m *ChargerStartUpload) Spec() types.MessageSpec {
	return MakeSpec(types.FuncChargerStart, types.DirectionUpload, "charger_start_upload", false, true)
}
func (m *ChargerStartUpload) Decode(data []byte) error {
	off := 0
	m.DeviceOrderNo, off, _ = ReadBCD(data, off, 10)
	m.StartupType, off, _ = ReadByte(data, off)
	elec, _, _ := ReadBytes(data, off, 5); off += 5; m.StartingElectricity = bytesToUint64(elec)
	m.PileChargingFailureReason, off, _ = ReadByte(data, off)
	m.AuthenticationNumber, off, _ = ReadASCII(data, off, 50)
	m.Time, off, _ = ReadBCD(data, off, 7)
	return nil
}
func (m *ChargerStartUpload) Encode() ([]byte, error) {
	buf := make([]byte, 10+1+5+1+50+7)
	off := 0
	off, _ = WriteBCD(buf, off, m.DeviceOrderNo, 10)
	off = WriteByte(buf, off, m.StartupType)
	off += copy(buf[off:], uint64ToBytes(m.StartingElectricity, 5))
	off = WriteByte(buf, off, m.PileChargingFailureReason)
	off = WriteASCII(buf, off, m.AuthenticationNumber, 50)
	off, _ = WriteBCD(buf, off, m.Time, 7)
	return buf[:off], nil
}
func (m *ChargerStartUpload) Validate() []types.ValidationError { return nil }
func (m *ChargerStartUpload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{
		"deviceOrderNo": m.DeviceOrderNo, "startupType": m.StartupType,
		"startingElectricity": m.StartingElectricity,
		"pileChargingFailureReason": m.PileChargingFailureReason,
		"authenticationNumber": m.AuthenticationNumber, "time": m.Time,
	}
}

// ChargerStartReply 平台回复0x04
type ChargerStartReply struct {
	ChargingOrderNumber         string    `json:"chargingOrderNumber"`          // BCD[10]
	ChargingPileOrderNumber     string    `json:"chargingPileOrderNumber"`      // BCD[10]
	AccountBalance              uint32    `json:"accountBalance"`               // BYTE[4]
	LimitTheMaximumChargeCharge float64   `json:"limitTheMaximumChargeCharge"`  // BYTE[4]
	LimitTheChargingTime        uint16    `json:"limitTheChargingTime"`         // BYTE[2]
	LimitTheAmountOfCharging    float64   `json:"limitTheAmountOfCharging"`     // BYTE[4]
	LimitChargingServiceFees    float64   `json:"limitChargingServiceFees"`     // BYTE[4]
	LimitChargingCharges        float64   `json:"limitChargingCharges"`         // BYTE[4]
	LimitSOC                    byte      `json:"limitSOC"`
	ErrorCode                   byte      `json:"errorCode"`
	AuthenticationNumber        string    `json:"authenticationNumber"`         // ASCII[50]
	TheScreenDisplayModeNumber  byte      `json:"theScreenDisplayModeNumber"`
	FeeNum                      byte      `json:"feeNum"`
	ListFee                     []FeeItem `json:"listFee"`
	StopCode                    string    `json:"stopCode"`                     // BCD[2]
}

func (m *ChargerStartReply) Spec() types.MessageSpec {
	return MakeSpec(types.FuncChargerStart, types.DirectionReply, "charger_start_reply", false, false)
}
func (m *ChargerStartReply) Decode(data []byte) error {
	off := 0
	m.ChargingOrderNumber, off, _ = ReadBCD(data, off, 10)
	m.ChargingPileOrderNumber, off, _ = ReadBCD(data, off, 10)
	m.AccountBalance, off, _ = ReadUint32LE(data, off)
	lmcc, off, _ := ReadUint32LE(data, off); m.LimitTheMaximumChargeCharge = float64(lmcc) / 100
	m.LimitTheChargingTime, off, _ = ReadUint16LE(data, off)
	lac, off, _ := ReadUint32LE(data, off); m.LimitTheAmountOfCharging = float64(lac) / 100
	lcsf, off, _ := ReadUint32LE(data, off); m.LimitChargingServiceFees = float64(lcsf) / 100
	lcc, off, _ := ReadUint32LE(data, off); m.LimitChargingCharges = float64(lcc) / 100
	m.LimitSOC, off, _ = ReadByte(data, off)
	m.ErrorCode, off, _ = ReadByte(data, off)
	m.AuthenticationNumber, off, _ = ReadASCII(data, off, 50)
	m.TheScreenDisplayModeNumber, off, _ = ReadByte(data, off)
	m.FeeNum, off, _ = ReadByte(data, off)
	m.ListFee = make([]FeeItem, m.FeeNum)
	for i := 0; i < int(m.FeeNum); i++ {
		var fi FeeItem
		fi.Hour, off, _ = ReadByte(data, off)
		fi.Min, off, _ = ReadByte(data, off)
		pf, _, _ := ReadBytes(data, off, 3); off += 3; fi.PowerFee = bytes3ToUint32(pf)
		sf, _, _ := ReadBytes(data, off, 3); off += 3; fi.SvcFee = bytes3ToUint32(sf)
		fi.Type, off, _ = ReadByte(data, off)
		fi.LimitedP, off, _ = ReadUint16LE(data, off)
		m.ListFee[i] = fi
	}
	m.StopCode, off, _ = ReadBCD(data, off, 2)
	return nil
}
func (m *ChargerStartReply) Encode() ([]byte, error) {
	buf := make([]byte, 10+10+4+4+2+4+4+4+1+1+50+1+1+int(m.FeeNum)*9+2)
	off := 0
	off, _ = WriteBCD(buf, off, m.ChargingOrderNumber, 10)
	off, _ = WriteBCD(buf, off, m.ChargingPileOrderNumber, 10)
	off = WriteUint32LE(buf, off, m.AccountBalance)
	off = WriteUint32LE(buf, off, uint32(m.LimitTheMaximumChargeCharge*100))
	off = WriteUint16LE(buf, off, m.LimitTheChargingTime)
	off = WriteUint32LE(buf, off, uint32(m.LimitTheAmountOfCharging*100))
	off = WriteUint32LE(buf, off, uint32(m.LimitChargingServiceFees*100))
	off = WriteUint32LE(buf, off, uint32(m.LimitChargingCharges*100))
	off = WriteByte(buf, off, m.LimitSOC)
	off = WriteByte(buf, off, m.ErrorCode)
	off = WriteASCII(buf, off, m.AuthenticationNumber, 50)
	off = WriteByte(buf, off, m.TheScreenDisplayModeNumber)
	off = WriteByte(buf, off, m.FeeNum)
	for _, fi := range m.ListFee {
		off = WriteByte(buf, off, fi.Hour)
		off = WriteByte(buf, off, fi.Min)
		off += copy(buf[off:], uint32ToBytes3(fi.PowerFee))
		off += copy(buf[off:], uint32ToBytes3(fi.SvcFee))
		off = WriteByte(buf, off, fi.Type)
		off = WriteUint16LE(buf, off, fi.LimitedP)
	}
	off, _ = WriteBCD(buf, off, m.StopCode, 2)
	return buf[:off], nil
}
func (m *ChargerStartReply) Validate() []types.ValidationError { return nil }
func (m *ChargerStartReply) ToJSONMap() map[string]interface{} {
	fees := make([]map[string]interface{}, len(m.ListFee))
	for i, f := range m.ListFee {
		fees[i] = map[string]interface{}{
			"hour": f.Hour, "min": f.Min, "powerFee": f.PowerFee,
			"svcFee": f.SvcFee, "type": f.Type, "limitedP": f.LimitedP,
		}
	}
	return map[string]interface{}{
		"chargingOrderNumber": m.ChargingOrderNumber,
		"chargingPileOrderNumber": m.ChargingPileOrderNumber,
		"accountBalance": m.AccountBalance, "feeNum": m.FeeNum,
		"listFee": fees, "stopCode": m.StopCode,
	}
}

// ==================== 0x05 桩充电停止请求 ====================

// FeeModelItem 分时段累计信息
type FeeModelItem struct {
	EndTime        string `json:"endTime"`        // BCD[6]
	ElectricPrice  uint32 `json:"electricPrice"`  // BYTE[3]
	ServicePrice   uint32 `json:"servicePrice"`   // BYTE[3]
	ElectricQuantity uint32 `json:"electricQuantity"` // BYTE[4]
	ElectricFee    uint32 `json:"electricFee"`    // BYTE[4]
	ServiceFee     uint32 `json:"serviceFee"`     // BYTE[4]
	ElectricFlag   byte   `json:"electricFlag"`   // BYTE[1]
}

// ChargerStopUpload 充电桩上报充电停止
type ChargerStopUpload struct {
	ChargeOrderNo    string         `json:"chargeOrderNo"`    // BCD[10]
	DeviceOrderNo    string         `json:"deviceOrderNo"`    // BCD[10]
	Type             byte           `json:"type"`
	StartElectricMeter uint64        `json:"startElectricMeter"` // BYTE[5]
	StopElectricMeter  uint64        `json:"stopElectricMeter"`  // BYTE[5]
	StopReason       byte           `json:"stopReason"`
	ConcurrentCharge byte           `json:"concurrentCharge"`
	BmsStopReason    []byte         `json:"bmsStopReason"`    // BIT[1]
	BmsFaultReason   []byte         `json:"bmsFaultReason"`   // BIT[2]
	BmsErrorReason   byte           `json:"bmsErrorReason"`   // BIT[1]
	ChargerStopReason byte          `json:"chargerStopReason"` // BIT[1]
	ChargerFaultReason []byte       `json:"chargerFaultReason"` // BIT[2]
	ChargerErrorReason byte         `json:"chargerErrorReason"` // BIT[1]
	StopSoc          byte           `json:"stopSoc"`
	BatteryMinVoltage uint16        `json:"batteryMinVoltage"` // 0.01
	BatteryMaxVoltage uint16        `json:"batteryMaxVoltage"` // 0.01
	BatteryMinTemperature interface{} `json:"batteryMinTemperature"` // TEMP
	BatteryMaxTemperature interface{} `json:"batteryMaxTemperature"` // TEMP
	ChargeTime       uint16         `json:"chargeTime"`
	OutputEnergy     uint16         `json:"outputEnergy"`
	DeviceSerialNo   byte           `json:"deviceSerialNo"`
	BemTimeoutCheck  []byte         `json:"bemTimeoutCheck"`  // RAW[4]
	CemTimeoutCheck  []byte         `json:"cemTimeoutCheck"`  // RAW[4]
	MessageCount     byte           `json:"messageCount"`
	FeeModelList     []FeeModelItem `json:"feeModelList"`
	ChargeStartTime  string         `json:"chargeStartTime"`  // BCD[7]
	ChargeEndTime    string         `json:"chargeEndTime"`    // BCD[7]
}

func (m *ChargerStopUpload) Spec() types.MessageSpec {
	return MakeSpec(types.FuncChargerStop, types.DirectionUpload, "charger_stop_upload", false, true)
}
func (m *ChargerStopUpload) Decode(data []byte) error {
	off := 0
	m.ChargeOrderNo, off, _ = ReadBCD(data, off, 10)
	m.DeviceOrderNo, off, _ = ReadBCD(data, off, 10)
	m.Type, off, _ = ReadByte(data, off)
	se, off, _ := ReadBytes(data, off, 5); m.StartElectricMeter = bytesToUint64(se)
	ee, off, _ := ReadBytes(data, off, 5); m.StopElectricMeter = bytesToUint64(ee)
	m.StopReason, off, _ = ReadByte(data, off)
	m.ConcurrentCharge, off, _ = ReadByte(data, off)
	m.BmsStopReason, off, _ = ReadBytes(data, off, 1)
	m.BmsFaultReason, off, _ = ReadBytes(data, off, 2)
	m.BmsErrorReason, off, _ = ReadByte(data, off)
	m.ChargerStopReason, off, _ = ReadByte(data, off)
	m.ChargerFaultReason, off, _ = ReadBytes(data, off, 2)
	m.ChargerErrorReason, off, _ = ReadByte(data, off)
	m.StopSoc, off, _ = ReadByte(data, off)
	m.BatteryMinVoltage, off, _ = ReadUint16LE(data, off)
	m.BatteryMaxVoltage, off, _ = ReadUint16LE(data, off)
	// 温度字段暂用简单方式读取
	m.BatteryMinTemperature, off, _ = ReadByte(data, off)
	m.BatteryMaxTemperature, off, _ = ReadByte(data, off)
	m.ChargeTime, off, _ = ReadUint16LE(data, off)
	m.OutputEnergy, off, _ = ReadUint16LE(data, off)
	m.DeviceSerialNo, off, _ = ReadByte(data, off)
	m.BemTimeoutCheck, off, _ = ReadBytes(data, off, 4)
	m.CemTimeoutCheck, off, _ = ReadBytes(data, off, 4)
	m.MessageCount, off, _ = ReadByte(data, off)
	m.FeeModelList = make([]FeeModelItem, m.MessageCount)
	for i := 0; i < int(m.MessageCount); i++ {
		var fi FeeModelItem
		fi.EndTime, off, _ = ReadBCD(data, off, 6)
		ep, off, _ := ReadBytes(data, off, 3); fi.ElectricPrice = bytes3ToUint32(ep)
		sp, off, _ := ReadBytes(data, off, 3); fi.ServicePrice = bytes3ToUint32(sp)
		eq, off, _ := ReadBytes(data, off, 4); fi.ElectricQuantity = binary.LittleEndian.Uint32(eq)
		ef, off, _ := ReadBytes(data, off, 4); fi.ElectricFee = binary.LittleEndian.Uint32(ef)
		sf, off, _ := ReadBytes(data, off, 4); fi.ServiceFee = binary.LittleEndian.Uint32(sf)
		fi.ElectricFlag, off, _ = ReadByte(data, off)
		m.FeeModelList[i] = fi
	}
	m.ChargeStartTime, off, _ = ReadBCD(data, off, 7)
	m.ChargeEndTime, off, _ = ReadBCD(data, off, 7)
	return nil
}
func (m *ChargerStopUpload) Encode() ([]byte, error) { return nil, nil } // TODO
func (m *ChargerStopUpload) Validate() []types.ValidationError { return nil }
func (m *ChargerStopUpload) ToJSONMap() map[string]interface{} {
	fees := make([]map[string]interface{}, len(m.FeeModelList))
	for i, f := range m.FeeModelList {
		fees[i] = map[string]interface{}{
			"endTime": f.EndTime, "electricPrice": f.ElectricPrice,
			"servicePrice": f.ServicePrice, "electricQuantity": f.ElectricQuantity,
			"electricFee": f.ElectricFee, "serviceFee": f.ServiceFee,
			"electricFlag": f.ElectricFlag,
		}
	}
	return map[string]interface{}{
		"chargeOrderNo": m.ChargeOrderNo, "deviceOrderNo": m.DeviceOrderNo,
		"type": m.Type, "stopReason": m.StopReason,
		"stopSoc": m.StopSoc, "messageCount": m.MessageCount,
		"feeModelList": fees,
		"chargeStartTime": m.ChargeStartTime, "chargeEndTime": m.ChargeEndTime,
	}
}

// ChargerStopReply 平台回复0x05
type ChargerStopReply struct {
	ChargeOrderNo string `json:"chargeOrderNo"` // BCD[10]
	ResponseCode  byte   `json:"responseCode"`  // 00成功 01失败
}

func (m *ChargerStopReply) Spec() types.MessageSpec {
	return MakeSpec(types.FuncChargerStop, types.DirectionReply, "charger_stop_reply", false, false)
}
func (m *ChargerStopReply) Decode(data []byte) error {
	off := 0
	m.ChargeOrderNo, off, _ = ReadBCD(data, off, 10)
	m.ResponseCode, off, _ = ReadByte(data, off)
	return nil
}
func (m *ChargerStopReply) Encode() ([]byte, error) {
	buf := make([]byte, 11)
	off, _ := WriteBCD(buf, 0, m.ChargeOrderNo, 10)
	WriteByte(buf, off, m.ResponseCode)
	return buf, nil
}
func (m *ChargerStopReply) Validate() []types.ValidationError { return nil }
func (m *ChargerStopReply) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"chargeOrderNo": m.ChargeOrderNo, "responseCode": m.ResponseCode}
}

// ==================== 0x06 充电数据上报 ====================

// OverTimeAccumulateItem 分时段累计信息(0x06)
type OverTimeAccumulateItem struct {
	EndTime          string `json:"endTime"`          // BCD[6]
	ElectricityPrices uint32 `json:"electricityPrices"` // BYTE[3]
	ServiceChargePrice uint32 `json:"serviceChargePrice"` // BYTE[3]
	Electricity      uint32 `json:"electricity"`       // BYTE[4]
	ElectricityFee   uint32 `json:"electricityFee"`    // BYTE[4]
	ServiceCharge    uint32 `json:"serviceCharge"`     // BYTE[4]
	PeaksValleysFlag byte   `json:"peaksValleysFlag"`  // BYTE[1]
}

// ChargingDataUpload 充电数据上报
type ChargingDataUpload struct {
	ChargingOrderNumber string                  `json:"chargingOrderNumber"` // BCD[10]
	PileEndOrderNumber  string                  `json:"pileEndOrderNumber"`  // BCD[10]
	CurrentElec         uint64                  `json:"currentElec"`         // BYTE[5]
	CurrentSOC          byte                    `json:"currentSOC"`
	OutputChargingVoltage float64               `json:"outputChargingVoltage"` // BYTE[2] 1位小数
	OutputChargingCurrent float64               `json:"outputChargingCurrent"` // BYTE[2] 2位小数
	InputPhaseAChargeVoltage float64            `json:"inputPhaseAChargeVoltage"`
	InputAPhaseChargingCurrent float64          `json:"inputAPhaseChargingCurrent"`
	InputBPhaseChargingVoltage float64          `json:"inputBPhaseChargingVoltage"`
	InputBPhaseChargingCurrent float64          `json:"inputBPhaseChargingCurrent"`
	InputCPhaseChargingVoltage float64          `json:"inputCPhaseChargingVoltage"`
	InputCPhaseChargeCurrent float64            `json:"inputCPhaseChargeCurrent"`
	PileTemperature     interface{}             `json:"pileTemperature"`      // TEMP
	ChargerTemperature  interface{}             `json:"chargerTemperature"`   // TEMP
	CarTemperature      uint16                  `json:"carTemperature"`       // BYTE[2]
	DemandChargingVoltage float64               `json:"demandChargingVoltage"`
	DemandChargingCurrent float64               `json:"demandChargingCurrent"`
	TwoChargerTogether  byte                    `json:"twoChargerTogether"`
	OverTimeAccumulateInformationCount byte     `json:"overTimeAccumulateInformationCount"`
	OverTimeAccumulateInformationList []OverTimeAccumulateItem `json:"overTimeAccumulateInformationList"`
}

func (m *ChargingDataUpload) Spec() types.MessageSpec {
	return MakeSpec(types.FuncChargingData, types.DirectionUpload, "charging_data_upload", false, true)
}
func (m *ChargingDataUpload) Decode(data []byte) error {
	off := 0
	m.ChargingOrderNumber, off, _ = ReadBCD(data, off, 10)
	m.PileEndOrderNumber, off, _ = ReadBCD(data, off, 10)
	ce, off, _ := ReadBytes(data, off, 5); m.CurrentElec = bytesToUint64(ce)
	m.CurrentSOC, off, _ = ReadByte(data, off)
	ocv, off, _ := ReadUint16LE(data, off); m.OutputChargingVoltage = float64(ocv) / 10
	oci, off, _ := ReadUint16LE(data, off); m.OutputChargingCurrent = float64(oci) / 100
	iav, off, _ := ReadUint16LE(data, off); m.InputPhaseAChargeVoltage = float64(iav) / 10
	iai, off, _ := ReadUint16LE(data, off); m.InputAPhaseChargingCurrent = float64(iai) / 100
	ibv, off, _ := ReadUint16LE(data, off); m.InputBPhaseChargingVoltage = float64(ibv) / 10
	ibi, off, _ := ReadUint16LE(data, off); m.InputBPhaseChargingCurrent = float64(ibi) / 100
	icv, off, _ := ReadUint16LE(data, off); m.InputCPhaseChargingVoltage = float64(icv) / 10
	ici, off, _ := ReadUint16LE(data, off); m.InputCPhaseChargeCurrent = float64(ici) / 100
	m.PileTemperature, off, _ = ReadByte(data, off)
	m.ChargerTemperature, off, _ = ReadByte(data, off)
	m.CarTemperature, off, _ = ReadUint16LE(data, off)
	dcv, off, _ := ReadUint16LE(data, off); m.DemandChargingVoltage = float64(dcv) / 10
	dci, off, _ := ReadUint16LE(data, off); m.DemandChargingCurrent = float64(dci) / 100
	m.TwoChargerTogether, off, _ = ReadByte(data, off)
	m.OverTimeAccumulateInformationCount, off, _ = ReadByte(data, off)
	m.OverTimeAccumulateInformationList = make([]OverTimeAccumulateItem, m.OverTimeAccumulateInformationCount)
	for i := 0; i < int(m.OverTimeAccumulateInformationCount); i++ {
		var item OverTimeAccumulateItem
		item.EndTime, off, _ = ReadBCD(data, off, 6)
		ep, off, _ := ReadBytes(data, off, 3); item.ElectricityPrices = bytes3ToUint32(ep)
		sp, off, _ := ReadBytes(data, off, 3); item.ServiceChargePrice = bytes3ToUint32(sp)
		eq, off, _ := ReadBytes(data, off, 4); item.Electricity = binary.LittleEndian.Uint32(eq)
		ef, off, _ := ReadBytes(data, off, 4); item.ElectricityFee = binary.LittleEndian.Uint32(ef)
		sf, off, _ := ReadBytes(data, off, 4); item.ServiceCharge = binary.LittleEndian.Uint32(sf)
		item.PeaksValleysFlag, off, _ = ReadByte(data, off)
		m.OverTimeAccumulateInformationList[i] = item
	}
	return nil
}
func (m *ChargingDataUpload) Encode() ([]byte, error) { return nil, nil } // TODO
func (m *ChargingDataUpload) Validate() []types.ValidationError { return nil }
func (m *ChargingDataUpload) ToJSONMap() map[string]interface{} {
	items := make([]map[string]interface{}, len(m.OverTimeAccumulateInformationList))
	for i, item := range m.OverTimeAccumulateInformationList {
		items[i] = map[string]interface{}{
			"endTime": item.EndTime, "electricityPrices": item.ElectricityPrices,
			"serviceChargePrice": item.ServiceChargePrice, "electricity": item.Electricity,
			"electricityFee": item.ElectricityFee, "serviceCharge": item.ServiceCharge,
			"peaksValleysFlag": item.PeaksValleysFlag,
		}
	}
	return map[string]interface{}{
		"chargingOrderNumber": m.ChargingOrderNumber, "pileEndOrderNumber": m.PileEndOrderNumber,
		"currentElec": m.CurrentElec, "currentSOC": m.CurrentSOC,
		"outputChargingVoltage": m.OutputChargingVoltage, "outputChargingCurrent": m.OutputChargingCurrent,
		"pileTemperature": m.PileTemperature, "chargerTemperature": m.ChargerTemperature,
		"overTimeAccumulateInformationCount": m.OverTimeAccumulateInformationCount,
		"overTimeAccumulateInformationList": items,
	}
}

// ChargingDataReply 平台回复0x06
type ChargingDataReply struct {
	Confirm byte `json:"confirm"`
}

func (m *ChargingDataReply) Spec() types.MessageSpec {
	return MakeSpec(types.FuncChargingData, types.DirectionReply, "charging_data_reply", false, false)
}
func (m *ChargingDataReply) Decode(data []byte) error {
	if len(data) < 1 { return errInsufficientData(1, len(data)) }
	m.Confirm = data[0]; return nil
}
func (m *ChargingDataReply) Encode() ([]byte, error) { return []byte{m.Confirm}, nil }
func (m *ChargingDataReply) Validate() []types.ValidationError { return nil }
func (m *ChargingDataReply) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"confirm": m.Confirm}
}
