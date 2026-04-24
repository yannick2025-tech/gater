// Package msg provides 0x0C/0xC1/0xC2 configuration message definitions.
package msg

import "github.com/yannick2025-tech/nts-gater/internal/protocol/types"

// ==================== 0x0C 设备参数查询 ====================

type DeviceQueryDownload struct {
	CmdCode byte   `json:"cmdCode"` // 命令码
	Data    []byte `json:"data"`    // 数据内容
}

func (m *DeviceQueryDownload) Spec() types.MessageSpec { return MakeSpec(types.FuncDeviceQuery, types.DirectionDownload, "device_query_download", false, true) }

func (m *DeviceQueryDownload) Decode(data []byte) error {
	if len(data) < 1 { return errInsufficientData(1, len(data)) }
	m.CmdCode = data[0]
	if len(data) > 1 { m.Data = data[1:] }
	return nil
}

func (m *DeviceQueryDownload) Encode() ([]byte, error) {
	buf := make([]byte, 1+len(m.Data))
	buf[0] = m.CmdCode; copy(buf[1:], m.Data); return buf, nil
}

func (m *DeviceQueryDownload) Validate() []types.ValidationError { return nil }

func (m *DeviceQueryDownload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"cmdCode": m.CmdCode}
}

type DeviceQueryReply struct {
	ResultCode byte   `json:"resultCode"` // 00成功 01异常
	CmdCode    byte   `json:"cmdCode"`
	Result     []byte `json:"result"`
}

func (m *DeviceQueryReply) Spec() types.MessageSpec { return MakeSpec(types.FuncDeviceQuery, types.DirectionReply, "device_query_reply", false, false) }

func (m *DeviceQueryReply) Decode(data []byte) error {
	if len(data) < 2 { return errInsufficientData(2, len(data)) }
	m.ResultCode = data[0]; m.CmdCode = data[1]
	if len(data) > 2 { m.Result = data[2:] }
	return nil
}

func (m *DeviceQueryReply) Encode() ([]byte, error) {
	buf := make([]byte, 2+len(m.Result))
	buf[0] = m.ResultCode; buf[1] = m.CmdCode; copy(buf[2:], m.Result); return buf, nil
}

func (m *DeviceQueryReply) Validate() []types.ValidationError { return nil }

func (m *DeviceQueryReply) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"resultCode": m.ResultCode, "cmdCode": m.CmdCode, "result": m.Result}
}

// ==================== 0xC1 设备参数上报 ====================

type ParamItem struct {
	Seq        uint16 `json:"seq"`        // 地址序号
	ValueBytes []byte `json:"valueBytes"` // 配置内容
	Value      string `json:"value"`      // 可读值
}

type ParamReportUpload struct {
	ParamCount uint16      `json:"paramCount"`
	ParamList  []ParamItem `json:"paramList"`
}

func (m *ParamReportUpload) Spec() types.MessageSpec { return MakeSpec(types.FuncParamReport, types.DirectionUpload, "param_report_upload", false, true) }

func (m *ParamReportUpload) Decode(data []byte) error {
	off := 0
	m.ParamCount, off, _ = ReadUint16LE(data, off)
	m.ParamList = make([]ParamItem, m.ParamCount)
	for i := 0; i < int(m.ParamCount); i++ {
		var item ParamItem
		var vlen byte
		item.Seq, off, _ = ReadUint16LE(data, off)
		vlen, off, _ = ReadByte(data, off)
		item.ValueBytes, off, _ = ReadBytes(data, off, int(vlen))
		m.ParamList[i] = item
	}
	return nil
}

func (m *ParamReportUpload) Encode() ([]byte, error) { return nil, nil } // TODO
func (m *ParamReportUpload) Validate() []types.ValidationError { return nil }

func (m *ParamReportUpload) ToJSONMap() map[string]interface{} {
	items := make([]map[string]interface{}, len(m.ParamList))
	for i, p := range m.ParamList {
		items[i] = map[string]interface{}{"seq": p.Seq, "valueBytes": p.ValueBytes}
	}
	return map[string]interface{}{"paramCount": m.ParamCount, "paramList": items}
}

type ParamReportReply struct {
	ResponseCode byte `json:"responseCode"` // 00成功 01失败
}

func (m *ParamReportReply) Spec() types.MessageSpec { return MakeSpec(types.FuncParamReport, types.DirectionReply, "param_report_reply", false, false) }

func (m *ParamReportReply) Decode(data []byte) error {
	if len(data) < 1 { return errInsufficientData(1, len(data)) }
	m.ResponseCode = data[0]; return nil
}

func (m *ParamReportReply) Encode() ([]byte, error) { return []byte{m.ResponseCode}, nil }

func (m *ParamReportReply) Validate() []types.ValidationError { return nil }

func (m *ParamReportReply) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"responseCode": m.ResponseCode}
}

// ==================== 0xC2 配置信息下发 ====================

type ConfigDownloadMsg struct {
	ParamList []ParamItem `json:"paramList"`
}

func (m *ConfigDownloadMsg) Spec() types.MessageSpec { return MakeSpec(types.FuncConfigDownload, types.DirectionDownload, "config_download", false, true) }

func (m *ConfigDownloadMsg) Decode(data []byte) error {
	off := 0
	for off < len(data) {
		var item ParamItem
		var vlen byte
		item.Seq, off, _ = ReadUint16LE(data, off)
		vlen, off, _ = ReadByte(data, off)
		item.ValueBytes, off, _ = ReadBytes(data, off, int(vlen))
		m.ParamList = append(m.ParamList, item)
	}
	return nil
}

func (m *ConfigDownloadMsg) Encode() ([]byte, error) { return nil, nil } // TODO
func (m *ConfigDownloadMsg) Validate() []types.ValidationError { return nil }

func (m *ConfigDownloadMsg) ToJSONMap() map[string]interface{} {
	items := make([]map[string]interface{}, len(m.ParamList))
	for i, p := range m.ParamList {
		items[i] = map[string]interface{}{"seq": p.Seq, "valueBytes": p.ValueBytes}
	}
	return map[string]interface{}{"paramList": items}
}

type ConfigResultItem struct {
	Seq        uint16 `json:"seq"`
	ResultCode byte   `json:"resultCode"` // 00成功 01失败
}

type ConfigDownloadReply struct {
	ParamCount uint16             `json:"paramCount"`
	ResultList []ConfigResultItem `json:"resultList"`
}

func (m *ConfigDownloadReply) Spec() types.MessageSpec { return MakeSpec(types.FuncConfigDownload, types.DirectionReply, "config_download_reply", false, false) }

func (m *ConfigDownloadReply) Decode(data []byte) error {
	off := 0; m.ParamCount, off, _ = ReadUint16LE(data, off)
	m.ResultList = make([]ConfigResultItem, m.ParamCount)
	for i := 0; i < int(m.ParamCount); i++ {
		m.ResultList[i].Seq, off, _ = ReadUint16LE(data, off)
		m.ResultList[i].ResultCode, off, _ = ReadByte(data, off)
	}
	return nil
}

func (m *ConfigDownloadReply) Encode() ([]byte, error) { return nil, nil } // TODO
func (m *ConfigDownloadReply) Validate() []types.ValidationError { return nil }

func (m *ConfigDownloadReply) ToJSONMap() map[string]interface{} {
	items := make([]map[string]interface{}, len(m.ResultList))
	for i, r := range m.ResultList {
		items[i] = map[string]interface{}{"seq": r.Seq, "resultCode": r.ResultCode}
	}
	return map[string]interface{}{"paramCount": m.ParamCount, "resultList": items}
}
