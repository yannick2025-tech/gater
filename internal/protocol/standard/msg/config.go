// Package msg provides 0x0C/0xC1/0xC2 configuration message definitions.
package msg

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
)

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

// UnmarshalJSON 自定义反序列化：Value 字段兼容 JSON string 和 number
// 前端 JSON payload 中 value 可能是 number（如 8888）或 string（如 "192.168.1.100"），
// Go 的 json.Unmarshal 默认严格要求类型对应，number 无法反序列化到 string，
// 因此这里做兼容处理：number 转为字符串，string 直接赋值。
func (p *ParamItem) UnmarshalJSON(data []byte) error {
	// 用辅助结构避免递归调用 UnmarshalJSON
	type Alias struct {
		Seq        uint16          `json:"seq"`
		ValueBytes json.RawMessage `json:"valueBytes"`
		Value      json.RawMessage `json:"value"`
	}
	var a Alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	p.Seq = a.Seq

	// 解析 ValueBytes（可选字段）
	if len(a.ValueBytes) > 0 {
		_ = json.Unmarshal(a.ValueBytes, &p.ValueBytes)
	}

	// 解析 Value：兼容 string 和 number
	if len(a.Value) == 0 {
		p.Value = ""
	} else {
		// 先尝试 string
		var s string
		if err := json.Unmarshal(a.Value, &s); err == nil {
			p.Value = s
		} else {
			// 再尝试 number（int / float）
			var f float64
			if err := json.Unmarshal(a.Value, &f); err == nil {
				// 整数不加小数点，浮点数保留精度
				if f == float64(int64(f)) {
					p.Value = json.Number(a.Value).String()
				} else {
					p.Value = string(a.Value)
				}
			} else {
				// 其他类型（bool 等）转为字符串
				p.Value = string(a.Value)
			}
		}
	}
	return nil
}

// EncodeValue 根据地址序号(seq)将可读 Value 字符串编码为二进制 ValueBytes。
// 编码规则参照协议文档"附录-配置项列表"：
//   - ASCII 类型：ASCII 字符串，右侧 0x00 填充到协议定义的固定长度
//   - IP 类型（seq=8,10,11,12）: 4 字节 IP 地址
//   - BYTE[n] 数值类型（seq=0,2,3,7,9 等）: 按 n 字节小端整数编码
//   - BCD 类型（seq=14,23,24）: BCD 编码
//
// 如果 ValueBytes 已有值（前端直接传入），优先使用 ValueBytes。
func (p *ParamItem) EncodeValue() error {
	// 如果前端已直接提供 ValueBytes，直接使用
	if len(p.ValueBytes) > 0 {
		return nil
	}

	if p.Value == "" {
		return fmt.Errorf("param seq %d: value is empty", p.Seq)
	}

	switch p.Seq {
	// ASCII 类型（必须右填充 0x00 到协议定义的固定长度）
	case 1: // 设备串号 ASCII[20]
		p.ValueBytes = padRight([]byte(p.Value), 20)
	case 4: // 软件版本号 ASCII[10]
		p.ValueBytes = padRight([]byte(p.Value), 10)
	case 5: // 硬件版本号 ASCII[10]
		p.ValueBytes = padRight([]byte(p.Value), 10)
	case 6: // 服务器地址 ASCII[30]
		p.ValueBytes = padRight([]byte(p.Value), 30)
	case 25: // 二维码 ASCII[100]
		p.ValueBytes = padRight([]byte(p.Value), 100)
	case 52: // A枪桩体号 ASCII[20]
		p.ValueBytes = padRight([]byte(p.Value), 20)
	case 53: // B枪桩体号 ASCII[20]
		p.ValueBytes = padRight([]byte(p.Value), 20)
	case 77: // 用户支付二维码 ASCII[50]
		p.ValueBytes = padRight([]byte(p.Value), 50)
	case 79: // 左枪二维码文字描述 ASCII[100]
		p.ValueBytes = padRight([]byte(p.Value), 100)
	case 80: // 左枪二维码 ASCII[100]
		p.ValueBytes = padRight([]byte(p.Value), 100)
	case 81: // 右枪二维码文字描述 ASCII[100]
		p.ValueBytes = padRight([]byte(p.Value), 100)
	case 82: // 右枪二维码 ASCII[100]
		p.ValueBytes = padRight([]byte(p.Value), 100)
	case 93: // 运营商ID ASCII[32]
		p.ValueBytes = padRight([]byte(p.Value), 32)

	// IP 地址类型 (4 字节)
	case 8, 10, 11, 12, 13:
		ip := net.ParseIP(p.Value)
		if ip == nil {
			return fmt.Errorf("param seq %d: invalid IP address %q", p.Seq, p.Value)
		}
		ip4 := ip.To4()
		if ip4 == nil {
			return fmt.Errorf("param seq %d: not an IPv4 address %q", p.Seq, p.Value)
		}
		p.ValueBytes = ip4

	// 端口号 / BYTE[2] 数值类型
	case 3, 7, 9, 22, 26, 27, 28, 29, 30, 31, 35, 36, 37, 38, 44, 45, 46, 47, 48, 49, 50, 51, 57, 62, 64, 65, 95:
		v, err := strconv.ParseUint(p.Value, 10, 16)
		if err != nil {
			return fmt.Errorf("param seq %d: invalid uint16 value %q: %v", p.Seq, p.Value, err)
		}
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(v))
		p.ValueBytes = buf

	// BYTE[1] 数值类型
	case 0, 15, 16, 17, 18, 19, 20, 21, 32, 33, 34, 39, 40, 41, 42, 43, 54, 56, 58, 59, 60, 61, 63, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 78, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 94, 96:
		v, err := strconv.ParseUint(p.Value, 10, 8)
		if err != nil {
			return fmt.Errorf("param seq %d: invalid byte value %q: %v", p.Seq, p.Value, err)
		}
		p.ValueBytes = []byte{byte(v)}

	// BYTE[4] 数值类型
	case 2:
		v, err := strconv.ParseUint(p.Value, 10, 32)
		if err != nil {
			return fmt.Errorf("param seq %d: invalid uint32 value %q: %v", p.Seq, p.Value, err)
		}
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(v))
		p.ValueBytes = buf

	// BCD[7] 类型
	case 14, 23, 24:
		off, err := WriteBCD(make([]byte, 7), 0, p.Value, 7)
		if err != nil {
			return fmt.Errorf("param seq %d: invalid BCD value %q: %v", p.Seq, p.Value, err)
		}
		p.ValueBytes = make([]byte, 7)
		WriteBCD(p.ValueBytes, 0, p.Value, 7)
		_ = off

	default:
		// 未知 seq，尝试作为 ASCII 字符串处理
		p.ValueBytes = []byte(p.Value)
	}

	return nil
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

func (m *ConfigDownloadMsg) Encode() ([]byte, error) {
	// 0xC2 编码格式：seq(2字节LE) + valueLen(1字节) + valueBytes(valueLen字节)，逐项拼接，无 paramCount 前缀
	var buf []byte
	for i, item := range m.ParamList {
		if err := item.EncodeValue(); err != nil {
			return nil, fmt.Errorf("paramList[%d]: %v", i, err)
		}
		if len(item.ValueBytes) > 255 {
			return nil, fmt.Errorf("paramList[%d]: valueBytes too long (%d bytes, max 255)", i, len(item.ValueBytes))
		}
		// seq(2字节LE)
		seqBuf := make([]byte, 2)
		binary.LittleEndian.PutUint16(seqBuf, item.Seq)
		buf = append(buf, seqBuf...)
		// valueLen(1字节)
		buf = append(buf, byte(len(item.ValueBytes)))
		// valueBytes
		buf = append(buf, item.ValueBytes...)
	}
	return buf, nil
}
func (m *ConfigDownloadMsg) Validate() []types.ValidationError {
	if len(m.ParamList) == 0 {
		return []types.ValidationError{{Field: "paramList", Message: "paramList is required"}}
	}
	return nil
}

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

// padRight 将字节切片右侧填充 0x00 到指定长度（协议固定长度字段）
func padRight(data []byte, length int) []byte {
	if len(data) >= length {
		return data[:length] // 截断超长数据
	}
	padded := make([]byte, length)
	copy(padded, data)
	return padded
}
