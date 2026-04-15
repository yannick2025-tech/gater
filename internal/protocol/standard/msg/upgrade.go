// Package msg provides 0xF6/0xF7 SFTP upgrade message definitions.
package msg

import "github.com/yannick2025-tech/nts-gater/internal/protocol/types"

// ==================== 0xF6 SFTP升级 ====================

type SFTPUpgradeDownload struct {
	Seq         byte   `json:"seq"`         // 升级序号
	UpgradeType byte   `json:"upgradeType"` // 0升级 1图片/视频
	PackageType byte   `json:"packageType"` // 更新包类型
	Version     uint16 `json:"version"`     // 更新包版本
	Address     string `json:"address"`     // ASCII[100]
	Port        uint16 `json:"port"`        // SFTP端口
	UserName    string `json:"userName"`    // ASCII[20]
	Password    string `json:"password"`    // ASCII[20]
	FilePath    string `json:"filePath"`    // ASCII[120]
	CrcCode     uint16 `json:"crcCode"`     // CRC16
}

func (m *SFTPUpgradeDownload) Spec() types.MessageSpec { return MakeSpec(types.FuncSFTPUpgrade, types.DirectionDownload, "sftp_upgrade_download", false, true) }

func (m *SFTPUpgradeDownload) Decode(data []byte) error {
	off := 0; m.Seq, off, _ = ReadByte(data, off); m.UpgradeType, off, _ = ReadByte(data, off)
	m.PackageType, off, _ = ReadByte(data, off); m.Version, off, _ = ReadUint16LE(data, off)
	m.Address, off, _ = ReadASCII(data, off, 100); m.Port, off, _ = ReadUint16LE(data, off)
	m.UserName, off, _ = ReadASCII(data, off, 20); m.Password, off, _ = ReadASCII(data, off, 20)
	m.FilePath, off, _ = ReadASCII(data, off, 120); m.CrcCode, off, _ = ReadUint16LE(data, off)
	return nil
}

func (m *SFTPUpgradeDownload) Encode() ([]byte, error) {
	buf := make([]byte, 1+1+1+2+100+2+20+20+120+2)
	off := 0; off = WriteByte(buf, off, m.Seq); off = WriteByte(buf, off, m.UpgradeType)
	off = WriteByte(buf, off, m.PackageType); off = WriteUint16LE(buf, off, m.Version)
	off = WriteASCII(buf, off, m.Address, 100); off = WriteUint16LE(buf, off, m.Port)
	off = WriteASCII(buf, off, m.UserName, 20); off = WriteASCII(buf, off, m.Password, 20)
	off = WriteASCII(buf, off, m.FilePath, 120); off = WriteUint16LE(buf, off, m.CrcCode)
	return buf[:off], nil
}

func (m *SFTPUpgradeDownload) Validate() []types.ValidationError { return nil }

func (m *SFTPUpgradeDownload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"seq": m.Seq, "upgradeType": m.UpgradeType, "packageType": m.PackageType,
		"version": m.Version, "address": m.Address, "port": m.Port, "userName": m.UserName,
		"password": m.Password, "filePath": m.FilePath, "crcCode": m.CrcCode}
}

type SFTPUpgradeReply struct {
	Seq         byte   `json:"seq"`
	PackageType byte   `json:"packageType"`
	Version     uint16 `json:"version"`
	Result      byte   `json:"result"` // 00成功 01失败 02占用中
}

func (m *SFTPUpgradeReply) Spec() types.MessageSpec { return MakeSpec(types.FuncSFTPUpgrade, types.DirectionReply, "sftp_upgrade_reply", false, false) }

func (m *SFTPUpgradeReply) Decode(data []byte) error {
	off := 0; m.Seq, off, _ = ReadByte(data, off); m.PackageType, off, _ = ReadByte(data, off)
	m.Version, off, _ = ReadUint16LE(data, off); m.Result, off, _ = ReadByte(data, off); return nil
}

func (m *SFTPUpgradeReply) Encode() ([]byte, error) {
	buf := make([]byte, 5); off := 0; off = WriteByte(buf, off, m.Seq)
	off = WriteByte(buf, off, m.PackageType); off = WriteUint16LE(buf, off, m.Version)
	WriteByte(buf, off, m.Result); return buf, nil
}

func (m *SFTPUpgradeReply) Validate() []types.ValidationError { return nil }

func (m *SFTPUpgradeReply) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"seq": m.Seq, "packageType": m.PackageType, "version": m.Version, "result": m.Result}
}

// ==================== 0xF7 升级进度上报 ====================

type UpgradeProgressUpload struct {
	Seq          byte `json:"seq"`
	ProgressRate byte `json:"progressRate"` // 00下载中 01下载完成 02安装中 03安装完成
}

func (m *UpgradeProgressUpload) Spec() types.MessageSpec { return MakeSpec(types.FuncUpgradeProgress, types.DirectionUpload, "upgrade_progress_upload", false, false) }

func (m *UpgradeProgressUpload) Decode(data []byte) error {
	if len(data) < 2 { return errInsufficientData(2, len(data)) }
	m.Seq = data[0]; m.ProgressRate = data[1]; return nil
}

func (m *UpgradeProgressUpload) Encode() ([]byte, error) { return []byte{m.Seq, m.ProgressRate}, nil }

func (m *UpgradeProgressUpload) Validate() []types.ValidationError { return nil }

func (m *UpgradeProgressUpload) ToJSONMap() map[string]interface{} {
	return map[string]interface{}{"seq": m.Seq, "progressRate": m.ProgressRate}
}
