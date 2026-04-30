// Package report provides seed data for development/testing when no real chargers are available.
package report

import (
	"time"

	"github.com/yannick2025-tech/nts-gater/internal/database"
	"github.com/yannick2025-tech/nts-gater/internal/model"
)

// SeedTestData inserts mock completed sessions into the database for frontend testing.
// Call this after database initialization during development.
func SeedTestData() error {
	db := database.GetDB()
	if db == nil {
		return nil // skip if DB not ready
	}

	// Check if seed data already exists
	var count int64
	db.Model(&model.TestReport{}).Where("session_id IN ?", []string{
		"A1B2C3D4E5F6G7H8",
		"B2C3D4E5F6G7H8I9",
		"C3D4E5F6G7H8I9J0",
	}).Count(&count)
	if count > 0 {
		return nil // already seeded
	}

	now := time.Now()

	t1 := now.Add(-90 * time.Minute)
	t2 := now.Add(-4*time.Hour + 30*time.Minute)
	t3 := now.Add(-10 * time.Minute)

	reports := []model.TestReport{
		{
			SessionID:     "A1B2C3D4E5F6G7H8",
			PostNo:        1002003001,
			ProtocolName:  "XX标准协议",
			ProtocolVer:   "v1.6.0",
			StartTime:     now.Add(-2 * time.Hour),
			EndTime:       &t1,
			DurationMs:    int64(30 * time.Minute / time.Millisecond),
			TotalMessages: 24,
			SuccessTotal:  23,
			FailTotal:     1,
			SuccessRate:   95.83,
			IsPass:        true,
			Status:        "completed",
		},
		{
			SessionID:     "B2C3D4E5F6G7H8I9",
			PostNo:        1002003002,
			ProtocolName:  "XX标准协议",
			ProtocolVer:   "v1.6.0",
			StartTime:     now.Add(-5 * time.Hour),
			EndTime:       &t2,
			DurationMs:    int64(30 * time.Minute / time.Millisecond),
			TotalMessages: 18,
			SuccessTotal:  16,
			FailTotal:     2,
			SuccessRate:   88.89,
			IsPass:        false,
			Status:        "completed",
		},
		{
			SessionID:     "C3D4E5F6G7H8I9J0",
			PostNo:        1002003003,
			ProtocolName:  "XX标准协议",
			ProtocolVer:   "v1.6.0",
			StartTime:     now.Add(-1 * time.Hour),
			EndTime:       &t3,
			DurationMs:    int64(50 * time.Minute / time.Millisecond),
			TotalMessages: 42,
			SuccessTotal:  42,
			FailTotal:     0,
			SuccessRate:   100.0,
			IsPass:        true,
			Status:        "completed",
		},
	}

	for _, r := range reports {
		if err := db.Create(&r).Error; err != nil {
			return err
		}
	}

	// Session A1B2... : 基础充电测试（通过）- 完整充电流程
	statsA := []model.FuncCodeStat{
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x0A", Direction: "充电桩→平台", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x0B", Direction: "回复", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x21", Direction: "上传", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x01", Direction: "充电桩→平台", TotalMessages: 5, SuccessCount: 5, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x23", Direction: "充电桩→平台", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x03", Direction: "平台→充电桩", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x04", Direction: "充电桩→平台", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x04", Direction: "回复", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x10", Direction: "充电桩→平台", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x10", Direction: "回复", TotalMessages: 1, SuccessCount: 0, DecodeFail: 0, InvalidField: 1, MessageLoss: 0, SuccessRate: 0}, // 1个字段校验失败
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x06", Direction: "充电桩→平台", TotalMessages: 4, SuccessCount: 4, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x06", Direction: "回复", TotalMessages: 4, SuccessCount: 4, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x08", Direction: "平台→充电桩", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x05", Direction: "充电桩→平台", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x05", Direction: "回复", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
	}

	// Session B2C3... : SFTP升级测试（失败）- 中途断连
	statsB := []model.FuncCodeStat{
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0x0A", Direction: "充电桩→平台", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0x0B", Direction: "回复", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0x21", Direction: "上传", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0x01", Direction: "充电桩→平台", TotalMessages: 3, SuccessCount: 3, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0xF6", Direction: "平台→充电桩", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0xF6", Direction: "回复", TotalMessages: 1, SuccessCount: 0, DecodeFail: 1, InvalidField: 0, MessageLoss: 0, SuccessRate: 0}, // 解码失败
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0xF7", Direction: "充电桩→平台", TotalMessages: 2, SuccessCount: 1, DecodeFail: 0, InvalidField: 1, MessageLoss: 0, SuccessRate: 50},  // 1成功+1校验失败
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0xF7", Direction: "回复", TotalMessages: 2, SuccessCount: 2, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0x23", Direction: "充电桩→平台", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
	}

	// Session C3D4... : 基础充电测试（完美通过，数据量大）
	statsC := []model.FuncCodeStat{
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x0A", Direction: "充电桩→平台", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x0B", Direction: "回复", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x21", Direction: "上传", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x01", Direction: "充电桩→平台", TotalMessages: 12, SuccessCount: 12, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x23", Direction: "充电桩→平台", TotalMessages: 2, SuccessCount: 2, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x03", Direction: "平台→充电桩", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x04", Direction: "充电桩→平台", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x04", Direction: "回复", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x10", Direction: "充电桩→平台", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x10", Direction: "回复", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x06", Direction: "充电桩→平台", TotalMessages: 8, SuccessCount: 8, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x06", Direction: "回复", TotalMessages: 8, SuccessCount: 8, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x25", Direction: "充电桩→平台", TotalMessages: 2, SuccessCount: 2, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x26", Direction: "充电桩→平台", TotalMessages: 2, SuccessCount: 2, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x08", Direction: "平台→充电桩", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x05", Direction: "充电桩→平台", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x05", Direction: "回复", TotalMessages: 1, SuccessCount: 1, DecodeFail: 0, InvalidField: 0, MessageLoss: 0, SuccessRate: 100},
	}

	allStats := append(append(statsA, statsB...), statsC...)
	for _, s := range allStats {
		if err := db.Create(&s).Error; err != nil {
			return err
		}
	}

	// Insert message archives for each session (sample messages for viewing)
	baseTimeA := now.Add(-2 * time.Hour)
	archivesA := []model.MessageArchive{
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x0A", Direction: "充电桩→平台", Status: "success", HexData: "68 32 00 01 00 01 3A 98 05 D8 44 5D 40 00 00 00 00 13 00 01 A5 B3 C7 ...", JSONData: `{"randomKey":"A5B3C7..."}`, Timestamp: baseTimeA},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x0B", Direction: "回复", Status: "success", HexData: "68 32 00 01 00 02 3A 98 05 D8 44 5D 40 00 07 00 00 41 B2 C3 D4 E5 F6 ...", JSONData: `{"authResult":0,"time":"260417101500"}`, Timestamp: baseTimeA.Add(2 * time.Second)},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x03", Direction: "平台→充电桩", Status: "success", HexData: "68 20 00 01 80 03 3A 98 05 D8 44 5D 40 00 00 06 53 43 45 4E 41 52 49 4F ...", JSONData: `{"startupType":6,"authNumber":"SCENARIO-A1B2C3..."}`, Timestamp: baseTimeA.Add(30 * time.Second)},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x04", Direction: "充电桩→平台", Status: "success", HexData: "68 40 00 01 00 04 3A 98 05 D8 44 5D 40 00 30 01 00 4F 52 44 30 30 30 31 ...", JSONData: `{"deviceOrderNo":"ORD0001","startupType":1,"vin":"LSVA...","soc":20}`, Timestamp: baseTimeA.Add(35 * time.Second)},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x06", Direction: "充电桩→平台", Status: "success", HexData: "68 38 00 01 00 06 3A 98 05 D8 44 5D 40 00 28 4F 52 44 30 30 30 31 00 00 00 ...", JSONData: `{"chargeOrderNo":"ORD0001","currentElec":150.5,"currentSOC":45}`, Timestamp: baseTimeA.Add(60 * time.Second)},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x06", Direction: "充电桩→平台", Status: "success", HexData: "68 38 00 01 00 06 3A 98 05 D8 44 5D 40 00 28 4F 52 44 30 30 30 31 00 00 00 ...", JSONData: `{"chargeOrderNo":"ORD0001","currentElec":180.2,"currentSOC":62}`, Timestamp: baseTimeA.Add(120 * time.Second)},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x08", Direction: "平台→充电桩", Status: "success", HexData: "68 18 00 01 80 08 3A 98 05 D8 44 5D 40 00 00 08 4F 46 30 30 31 ...", JSONData: `{"platformOrderNo":"OF001","stopReason":0}`, Timestamp: baseTimeA.Add(25 * time.Minute)},
		{SessionID: "A1B2C3D4E5F6G7H8", FuncCode: "0x05", Direction: "充电桩→平台", Status: "success", HexData: "68 20 00 01 00 05 3A 98 05 D8 44 5D 40 00 10 4F 52 44 30 30 30 31 01 58 ...", JSONData: `{"chargeOrderNo":"ORD0001","stopReason":1,"soc":88}`, Timestamp: baseTimeA.Add(26 * time.Minute)},
	}

	baseTimeB := now.Add(-5 * time.Hour)
	archivesB := []model.MessageArchive{
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0x0A", Direction: "充电桩→平台", Status: "success", HexData: "68 32 00 01 00 0A B2 C3 D4 E5 F6 G7 H8 I9 00 00 00 00 13 00 01 X1 Y2 Z3 ...", JSONData: `{"randomKey":"X1Y2Z3..."}`, Timestamp: baseTimeB},
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0x0B", Direction: "回复", Status: "success", HexData: "68 32 00 01 00 02 B2 C3 D4 E5 F6 G7 H8 I9 00 07 00 00 F1 E2 D3 C4 B5 A6 ...", JSONData: `{"authResult":0,"time":"260417083000"}`, Timestamp: baseTimeB.Add(2 * time.Second)},
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0xF6", Direction: "平台→充电桩", Status: "success", HexData: "68 30 00 01 80 F6 B2 C3 D4 E5 F6 G7 H8 I9 00 00 20 66 69 72 6D 77 61 72 65 ...", JSONData: `{"firmwarePath":"/firmware/gater_v1.6.0.bin","md5":"abc123..."}`, Timestamp: baseTimeB.Add(10 * time.Second)},
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0xF6", Direction: "回复", Status: "decode_fail", HexData: "68 15 00 01 02 F6 B2 C3 D4 E5 F6 G7 H8 I9 00 00 03 FF 01", JSONData: ``, ErrorMsg: "decode failed: invalid frame length", Timestamp: baseTimeB.Add(15 * time.Second)},
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0xF7", Direction: "充电桩→平台", Status: "success", HexData: "68 18 00 01 00 F7 B2 C3 D4 E5 F6 G7 H8 I9 00 00 08 00 32 00 00 ...", JSONData: `{"progress":50,"status":0}`, Timestamp: baseTimeB.Add(20 * time.Second)},
		{SessionID: "B2C3D4E5F6G7H8I9", FuncCode: "0xF7", Direction: "充电桩→平台", Status: "invalid_field", HexData: "68 18 00 01 00 F7 B2 C3 D4 E5 F6 G7 H8 I9 00 00 08 FF 99 00 00 ...", JSONData: `{"progress":150,"status":255}`, ErrorMsg: "validation failed: progress out of range [0-100]", Timestamp: baseTimeB.Add(22 * time.Second)},
	}

	baseTimeC := now.Add(-1 * time.Hour)
	archivesC := []model.MessageArchive{
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x0A", Direction: "充电桩→平台", Status: "success", HexData: "68 32 00 01 00 0A C3 D4 E5 F6 G7 H8 I9 J0 00 00 00 00 13 00 01 M1 N2 O3 ...", JSONData: `{"randomKey":"M1N2O3..."}`, Timestamp: baseTimeC},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x0B", Direction: "回复", Status: "success", HexData: "68 32 00 01 00 02 C3 D4 E5 F6 G7 H8 I9 J0 00 07 00 00 G1 H2 I3 J4 K5 L6 ...", JSONData: `{"authResult":0,"time":"260417111500"}`, Timestamp: baseTimeC.Add(2 * time.Second)},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x21", Direction: "上传", Status: "success", HexData: "68 12 00 01 00 21 C3 D4 E5 F6 G7 H8 I9 J0 00 00 02 00 00", JSONData: `{"secretUpdateStatus":0}`, Timestamp: baseTimeC.Add(5 * time.Second)},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x03", Direction: "平台→充电桩", Status: "success", HexData: "68 20 00 01 80 03 C3 D4 E5 F6 G7 H8 I9 J0 00 00 06 53 43 45 4E 41 52 49 4F ...", JSONData: `{"startupType":6,"authNumber":"SCENARIO-C3D4E5..."}`, Timestamp: baseTimeC.Add(10 * time.Second)},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x04", Direction: "充电桩→平台", Status: "success", HexData: "68 40 00 01 00 04 C3 D4 E5 F6 G7 H8 I9 J0 00 30 01 00 4F 52 44 30 30 30 33 ...", JSONData: `{"deviceOrderNo":"ORD003","startupType":1,"vin":"LVGBU...","soc":15}`, Timestamp: baseTimeC.Add(12 * time.Second)},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x04", Direction: "回复", Status: "success", HexData: "68 48 00 01 02 04 C3 D4 E5 F6 G7 H8 I9 J0 00 38 4F 52 44 30 30 30 33 00 00 C3 50 ...", JSONData: `{"chargingOrderNo":"CO003","accountBalance":50000,...}`, Timestamp: baseTimeC.Add(13 * time.Second)},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x10", Direction: "充电桩→平台", Status: "success", HexData: "68 60 00 01 00 10 C3 D4 E5 F6 G7 H8 I9 J0 00 50 ...", JSONData: `{"bmsVersion":"V2.1","batteryCapacity":60,"ratedVoltage":380}`, Timestamp: baseTimeC.Add(15 * time.Second)},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x06", Direction: "充电桩→平台", Status: "success", HexData: "68 38 00 01 00 06 C3 D4 E5 F6 G7 H8 I9 J0 00 28 4F 52 44 30 30 30 33 ...", JSONData: `{"chargeOrderNo":"CO003","currentElec":120.0,"currentSOC":35}`, Timestamp: baseTimeC.Add(5 * time.Minute)},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x06", Direction: "充电桩→平台", Status: "success", HexData: "68 38 00 01 00 06 C3 D4 E5 F6 G7 H8 I9 J0 00 28 4F 52 44 30 30 30 33 ...", JSONData: `{"chargeOrderNo":"CO003","currentElec":250.5,"currentSOC":68}`, Timestamp: baseTimeC.Add(15 * time.Minute)},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x08", Direction: "平台→充电桩", Status: "success", HexData: "68 18 00 01 80 08 C3 D4 E5 F6 G7 H8 I9 J0 00 00 08 4F 46 30 30 33 ...", JSONData: `{"platformOrderNo":"OF003","stopReason":0}`, Timestamp: baseTimeC.Add(45 * time.Minute)},
		{SessionID: "C3D4E5F6G7H8I9J0", FuncCode: "0x05", Direction: "充电桩→平台", Status: "success", HexData: "68 20 00 01 00 05 C3 D4 E5 F6 G7 H8 I9 J0 00 10 4F 52 44 30 30 30 33 01 5A ...", JSONData: `{"chargeOrderNo":"CO003","stopReason":1,"soc":95}`, Timestamp: baseTimeC.Add(46 * time.Minute)},
	}

	allArchives := append(append(archivesA, archivesB...), archivesC...)
	for _, a := range allArchives {
		if err := db.Create(&a).Error; err != nil {
			return err
		}
	}

	return nil
}
