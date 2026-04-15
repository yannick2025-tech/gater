// Package msg provides BMS-related message definitions (0x10, 0x24-0x27).
package msg

import "github.com/yannick2025-tech/nts-gater/internal/protocol/types"

// ==================== 0x10 BMS静态数据 ====================

type BMSStaticUpload struct {
	TheCurrentOrderNumber                    string  `json:"theCurrentOrderNumber"` // BCD[10]
	BmsCommunicationProtocolVersionNumber   []byte  `json:"bmsCommunicationProtocolVersionNumber"` // BYTE[3]
	BatteryType                              byte    `json:"batteryType"`
	RatedCapacityOfVehicleBatteryPack        uint16  `json:"ratedCapacityOfVehicleBatteryPack"`
	RatedVoltageOfTheVehicleBatteryPack      uint16  `json:"ratedVoltageOfTheVehicleBatteryPack"`
	BatteryManufacturer                      []byte  `json:"batteryManufacturer"` // BYTE[4]
	BatteryPackSerialNumber                  []byte  `json:"batteryPackSerialNumber"` // BYTE[4]
	BatteryPackDateOfManufacture             []byte  `json:"batteryPackDateOfManufacture"` // BYTE[3]
	BatteryPackNumberOfCharges               []byte  `json:"batteryPackNumberOfCharges"` // BYTE[3]
	BatteryPackTitleMarking                  byte    `json:"batteryPackTitleMarking"`
	Vin                                      string  `json:"vin"` // ASCII[17]
	TheMaximumVoltageOfASingleCell           uint16  `json:"theMaximumVoltageOfASingleCell"`
	TheHighestCurrentOfASingleBattery        uint16  `json:"theHighestCurrentOfASingleBattery"`
	PowerBatteryNominalTotalElectricalEnergy uint16  `json:"powerBatteryNominalTotalElectricalEnergy"`
	MaximumAllowableTotalChargingVoltage     float64 `json:"maximumAllowableTotalChargingVoltage"`
	MaximumPermissibleTemperature            interface{} `json:"maximumPermissibleTemperature"`
	TheStateOfChargeOfTheVehiclePowerBattery uint16  `json:"theStateOfChargeOfTheVehiclePowerBattery"`
	VehiclePowerBatteryCurrentBatteryVoltage float64 `json:"vehiclePowerBatteryCurrentBatteryVoltage"`
	MaximumOutputVoltage                     float64 `json:"maximumOutputVoltage"`
	MinimumOutputVoltage                     float64 `json:"minimumOutputVoltage"`
	MaximumOutputCurrent                     float64 `json:"maximumOutputCurrent"`
	BatteryReadiness                         interface{} `json:"batteryReadiness"`
	BmsSoftwareVersion                       []byte  `json:"bmsSoftwareVersion"` // BYTE[8]
	ChargerCommunicationProtocolVersionNumber []byte `json:"chargerCommunicationProtocolVersionNumber"` // BYTE[3]
	CrmIdentificationResults                 byte    `json:"crmIdentificationResults"`
	TheMinimumOutputCurrentOfTheCharger      uint16  `json:"theMinimumOutputCurrentOfTheCharger"`
	ChargerNumber                            []byte  `json:"chargerNumber"` // BYTE[4]
}

func (m *BMSStaticUpload) Spec() types.MessageSpec { return MakeSpec(types.FuncBMSStatic, types.DirectionUpload, "bms_static_upload", false, true) }

func (m *BMSStaticUpload) Decode(data []byte) error {
	off := 0
	m.TheCurrentOrderNumber, off, _ = ReadBCD(data, off, 10)
	m.BmsCommunicationProtocolVersionNumber, off, _ = ReadBytes(data, off, 3)
	m.BatteryType, off, _ = ReadByte(data, off)
	m.RatedCapacityOfVehicleBatteryPack, off, _ = ReadUint16LE(data, off)
	m.RatedVoltageOfTheVehicleBatteryPack, off, _ = ReadUint16LE(data, off)
	m.BatteryManufacturer, off, _ = ReadBytes(data, off, 4)
	m.BatteryPackSerialNumber, off, _ = ReadBytes(data, off, 4)
	m.BatteryPackDateOfManufacture, off, _ = ReadBytes(data, off, 3)
	m.BatteryPackNumberOfCharges, off, _ = ReadBytes(data, off, 3)
	m.BatteryPackTitleMarking, off, _ = ReadByte(data, off)
	m.Vin, off, _ = ReadASCII(data, off, 17)
	m.TheMaximumVoltageOfASingleCell, off, _ = ReadUint16LE(data, off)
	m.TheHighestCurrentOfASingleBattery, off, _ = ReadUint16LE(data, off)
	m.PowerBatteryNominalTotalElectricalEnergy, off, _ = ReadUint16LE(data, off)
	matv, off, _ := ReadUint16LE(data, off); m.MaximumAllowableTotalChargingVoltage = float64(matv) / 10
	m.MaximumPermissibleTemperature, off, _ = ReadByte(data, off) // TEMP简化
	m.TheStateOfChargeOfTheVehiclePowerBattery, off, _ = ReadUint16LE(data, off)
	vcv, off, _ := ReadUint16LE(data, off); m.VehiclePowerBatteryCurrentBatteryVoltage = float64(vcv) / 10
	mxov, off, _ := ReadUint16LE(data, off); m.MaximumOutputVoltage = float64(mxov) / 10
	mnov, off, _ := ReadUint16LE(data, off); m.MinimumOutputVoltage = float64(mnov) / 10
	mxoc, off, _ := ReadUint16LE(data, off); m.MaximumOutputCurrent = float64(mxoc) / 10
	m.BatteryReadiness, off, _ = ReadByte(data, off)
	m.BmsSoftwareVersion, off, _ = ReadBytes(data, off, 8)
	m.ChargerCommunicationProtocolVersionNumber, off, _ = ReadBytes(data, off, 3)
	m.CrmIdentificationResults, off, _ = ReadByte(data, off)
	m.TheMinimumOutputCurrentOfTheCharger, off, _ = ReadUint16LE(data, off)
	m.ChargerNumber, off, _ = ReadBytes(data, off, 4)
	return nil
}

func (m *BMSStaticUpload) Encode() ([]byte, error) { return nil, nil } // TODO
func (m *BMSStaticUpload) Validate() []types.ValidationError { return nil }

func (m *BMSStaticUpload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{
		"theCurrentOrderNumber": m.TheCurrentOrderNumber,
		"bmsCommunicationProtocolVersionNumber": m.BmsCommunicationProtocolVersionNumber,
		"batteryType": m.BatteryType, "maximumAllowableTotalChargingVoltage": m.MaximumAllowableTotalChargingVoltage,
		"maximumOutputVoltage": m.MaximumOutputVoltage, "minimumOutputVoltage": m.MinimumOutputVoltage,
		"maximumOutputCurrent": m.MaximumOutputCurrent,
	}
}

type BMSStaticReply struct{ Reply byte `json:"reply"` }

func (m *BMSStaticReply) Spec() types.MessageSpec { return MakeSpec(types.FuncBMSStatic, types.DirectionReply, "bms_static_reply", false, false) }

func (m *BMSStaticReply) Decode(data []byte) error { if len(data) < 1 { return errInsufficientData(1, len(data)) }; m.Reply = data[0]; return nil }

func (m *BMSStaticReply) Encode() ([]byte, error) { return []byte{m.Reply}, nil }

func (m *BMSStaticReply) Validate() []types.ValidationError { return nil }

func (m *BMSStaticReply) ToJSONMap() map[string]interface{} { return map[string]interface{}{"reply": m.Reply} }

// ==================== 0x24 充电中BMS数据 ====================

type BMSChargingUpload struct {
	TheCurrentOrderNumber                     string      `json:"theCurrentOrderNumber"` // BCD[10]
	VoltageDemand                             float64     `json:"voltageDemand"`
	CurrentDemand                             float64     `json:"currentDemand"`
	ChargingMode                              byte        `json:"chargingMode"`
	ChargingVoltageMeasurements               float64     `json:"chargingVoltageMeasurements"`
	ChargeCurrentMeasurements                 float64     `json:"chargeCurrentMeasurements"`
	MaximumMonomerPowerBatteryVoltage         float64     `json:"maximumMonomerPowerBatteryVoltage"`
	TheHighestSinglePowerBatteryVoltageGroupNumber byte   `json:"theHighestSinglePowerBatteryVoltageGroupNumber"`
	CurrentStateOfCharge                      byte        `json:"currentStateOfCharge"`
	EstimateTheTimeRemaining                  uint16      `json:"estimateTheTimeRemaining"`
	VoltageOutputValue                        uint16      `json:"voltageOutputValue"`
	CurrentOutputValue                        uint16      `json:"currentOutputValue"`
	CumulativeChargingTime                    uint16      `json:"cumulativeChargingTime"`
	TheHighestSinglePowerBatteryVoltageNumber byte        `json:"theHighestSinglePowerBatteryVoltageNumber"`
	MaximumPowerBatteryTemperature            interface{} `json:"maximumPowerBatteryTemperature"`
	MaximumTemperatureMonitoringPointNumber   byte        `json:"maximumTemperatureMonitoringPointNumber"`
	MinimumPowerBatteryTemperature            interface{} `json:"minimumPowerBatteryTemperature"`
	MinimumPowerBatteryTemperatureMonitoringPointNumber byte `json:"minimumPowerBatteryTemperatureMonitoringPointNumber"`
	SinglePowerBatteryVoltage                 byte        `json:"singlePowerBatteryVoltage"`
	VehiclePowerBatteryStateOfChargeSOC       byte        `json:"vehiclePowerBatteryStateOfChargeSOC"`
	PowerBatteryChargingOvercurrent           byte        `json:"powerBatteryChargingOvercurrent"`
	ThePowerBatteryIsChargedOverTemperature   byte        `json:"thePowerBatteryIsChargedOverTemperature"`
	PowerBatteryInsulationState               byte        `json:"powerBatteryInsulationState"`
	PowerBatteryPackOutputConnectorConnection byte        `json:"powerBatteryPackOutputConnectorConnection"`
	ChargingAllowed                           byte        `json:"chargingAllowed"`
	TheChargerOutputsAReadyToUseMessage       byte        `json:"theChargerOutputsAReadyToUseMessage"`
}

func (m *BMSChargingUpload) Spec() types.MessageSpec { return MakeSpec(types.FuncBMSCharging, types.DirectionUpload, "bms_charging_upload", false, true) }

func (m *BMSChargingUpload) Decode(data []byte) error {
	off := 0
	m.TheCurrentOrderNumber, off, _ = ReadBCD(data, off, 10)
	vd, off, _ := ReadUint16LE(data, off); m.VoltageDemand = float64(vd) / 10
	cd, off, _ := ReadUint16LE(data, off); m.CurrentDemand = float64(cd) / 100
	m.ChargingMode, off, _ = ReadByte(data, off)
	cvm, off, _ := ReadUint16LE(data, off); m.ChargingVoltageMeasurements = float64(cvm) / 10
	ccm, off, _ := ReadUint16LE(data, off); m.ChargeCurrentMeasurements = float64(ccm) / 10
	mmpv, off, _ := ReadUint16LE(data, off); m.MaximumMonomerPowerBatteryVoltage = float64(mmpv) / 100
	m.TheHighestSinglePowerBatteryVoltageGroupNumber, off, _ = ReadByte(data, off)
	m.CurrentStateOfCharge, off, _ = ReadByte(data, off)
	m.EstimateTheTimeRemaining, off, _ = ReadUint16LE(data, off)
	m.VoltageOutputValue, off, _ = ReadUint16LE(data, off)
	m.CurrentOutputValue, off, _ = ReadUint16LE(data, off)
	m.CumulativeChargingTime, off, _ = ReadUint16LE(data, off)
	m.TheHighestSinglePowerBatteryVoltageNumber, off, _ = ReadByte(data, off)
	m.MaximumPowerBatteryTemperature, off, _ = ReadByte(data, off)
	m.MaximumTemperatureMonitoringPointNumber, off, _ = ReadByte(data, off)
	m.MinimumPowerBatteryTemperature, off, _ = ReadByte(data, off)
	m.MinimumPowerBatteryTemperatureMonitoringPointNumber, off, _ = ReadByte(data, off)
	m.SinglePowerBatteryVoltage, off, _ = ReadByte(data, off)
	m.VehiclePowerBatteryStateOfChargeSOC, off, _ = ReadByte(data, off)
	m.PowerBatteryChargingOvercurrent, off, _ = ReadByte(data, off)
	m.ThePowerBatteryIsChargedOverTemperature, off, _ = ReadByte(data, off)
	m.PowerBatteryInsulationState, off, _ = ReadByte(data, off)
	m.PowerBatteryPackOutputConnectorConnection, off, _ = ReadByte(data, off)
	m.ChargingAllowed, off, _ = ReadByte(data, off)
	m.TheChargerOutputsAReadyToUseMessage, off, _ = ReadByte(data, off)
	return nil
}

func (m *BMSChargingUpload) Encode() ([]byte, error) { return nil, nil } // TODO
func (m *BMSChargingUpload) Validate() []types.ValidationError { return nil }

func (m *BMSChargingUpload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{
		"theCurrentOrderNumber": m.TheCurrentOrderNumber, "voltageDemand": m.VoltageDemand,
		"currentDemand": m.CurrentDemand, "chargingMode": m.ChargingMode,
		"chargingAllowed": m.ChargingAllowed,
	}
}

type BMSChargingReply struct{ Reply byte `json:"reply"` }

func (m *BMSChargingReply) Spec() types.MessageSpec { return MakeSpec(types.FuncBMSCharging, types.DirectionReply, "bms_charging_reply", false, false) }

func (m *BMSChargingReply) Decode(data []byte) error { if len(data) < 1 { return errInsufficientData(1, len(data)) }; m.Reply = data[0]; return nil }

func (m *BMSChargingReply) Encode() ([]byte, error) { return []byte{m.Reply}, nil }

func (m *BMSChargingReply) Validate() []types.ValidationError { return nil }

func (m *BMSChargingReply) ToJSONMap() map[string]interface{} { return map[string]interface{}{"reply": m.Reply} }

// ==================== 0x25/0x26/0x27 简化实现 ====================

type BMVVoltageUpload struct {
	TheCurrentOrderNumber string `json:"theCurrentOrderNumber"` // BCD[10]
	RawData               []byte `json:"rawData"`               // 剩余数据
}

func (m *BMVVoltageUpload) Spec() types.MessageSpec { return MakeSpec(types.FuncBMVVoltage, types.DirectionUpload, "bmv_voltage_upload", false, true) }

func (m *BMVVoltageUpload) Decode(data []byte) error {
	if len(data) >= 10 { m.TheCurrentOrderNumber, _, _ = ReadBCD(data, 0, 10) }
	if len(data) > 10 { m.RawData = data[10:] }
	return nil
}

func (m *BMVVoltageUpload) Encode() ([]byte, error) { return nil, nil }

func (m *BMVVoltageUpload) Validate() []types.ValidationError { return nil }

func (m *BMVVoltageUpload) ToJSONMap() map[string]interface{} { return map[string]interface{}{"theCurrentOrderNumber": m.TheCurrentOrderNumber} }

type BMVVoltageReply struct{ Reply byte `json:"reply"` }

func (m *BMVVoltageReply) Spec() types.MessageSpec { return MakeSpec(types.FuncBMVVoltage, types.DirectionReply, "bmv_voltage_reply", false, false) }

func (m *BMVVoltageReply) Decode(data []byte) error { if len(data) < 1 { return errInsufficientData(1, len(data)) }; m.Reply = data[0]; return nil }

func (m *BMVVoltageReply) Encode() ([]byte, error) { return []byte{m.Reply}, nil }

func (m *BMVVoltageReply) Validate() []types.ValidationError { return nil }

func (m *BMVVoltageReply) ToJSONMap() map[string]interface{} { return map[string]interface{}{"reply": m.Reply} }

type BMTTemperatureUpload struct {
	TheCurrentOrderNumber string `json:"theCurrentOrderNumber"` // BCD[10]
	RawData               []byte `json:"rawData"`
}

func (m *BMTTemperatureUpload) Spec() types.MessageSpec { return MakeSpec(types.FuncBMTTemperature, types.DirectionUpload, "bmt_temperature_upload", false, true) }

func (m *BMTTemperatureUpload) Decode(data []byte) error {
	if len(data) >= 10 { m.TheCurrentOrderNumber, _, _ = ReadBCD(data, 0, 10) }
	if len(data) > 10 { m.RawData = data[10:] }
	return nil
}

func (m *BMTTemperatureUpload) Encode() ([]byte, error) { return nil, nil }

func (m *BMTTemperatureUpload) Validate() []types.ValidationError { return nil }

func (m *BMTTemperatureUpload) ToJSONMap() map[string]interface{} { return map[string]interface{}{"theCurrentOrderNumber": m.TheCurrentOrderNumber} }

type BMTTemperatureReply struct{ Reply byte `json:"reply"` }

func (m *BMTTemperatureReply) Spec() types.MessageSpec { return MakeSpec(types.FuncBMTTemperature, types.DirectionReply, "bmt_temperature_reply", false, false) }

func (m *BMTTemperatureReply) Decode(data []byte) error { if len(data) < 1 { return errInsufficientData(1, len(data)) }; m.Reply = data[0]; return nil }

func (m *BMTTemperatureReply) Encode() ([]byte, error) { return []byte{m.Reply}, nil }

func (m *BMTTemperatureReply) Validate() []types.ValidationError { return nil }

func (m *BMTTemperatureReply) ToJSONMap() map[string]interface{} { return map[string]interface{}{"reply": m.Reply} }

type BSPReservedUpload struct {
	TheCurrentOrderNumber string `json:"theCurrentOrderNumber"` // BCD[10]
	RawData               []byte `json:"rawData"`
}

func (m *BSPReservedUpload) Spec() types.MessageSpec { return MakeSpec(types.FuncBSPReserved, types.DirectionUpload, "bsp_reserved_upload", false, true) }

func (m *BSPReservedUpload) Decode(data []byte) error {
	if len(data) >= 10 { m.TheCurrentOrderNumber, _, _ = ReadBCD(data, 0, 10) }
	if len(data) > 10 { m.RawData = data[10:] }
	return nil
}

func (m *BSPReservedUpload) Encode() ([]byte, error) { return nil, nil }

func (m *BSPReservedUpload) Validate() []types.ValidationError { return nil }

func (m *BSPReservedUpload) ToJSONMap() map[string]interface{} { return map[string]interface{}{"theCurrentOrderNumber": m.TheCurrentOrderNumber} }

type BSPReservedReply struct{ Reply byte `json:"reply"` }

func (m *BSPReservedReply) Spec() types.MessageSpec { return MakeSpec(types.FuncBSPReserved, types.DirectionReply, "bsp_reserved_reply", false, false) }

func (m *BSPReservedReply) Decode(data []byte) error { if len(data) < 1 { return errInsufficientData(1, len(data)) }; m.Reply = data[0]; return nil }

func (m *BSPReservedReply) Encode() ([]byte, error) { return []byte{m.Reply}, nil }

func (m *BSPReservedReply) Validate() []types.ValidationError { return nil }

func (m *BSPReservedReply) ToJSONMap() map[string]interface{} { return map[string]interface{}{"reply": m.Reply} }
