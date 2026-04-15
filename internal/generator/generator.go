// Package generator provides test data generation utilities for charging scenarios.
package generator

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var (
	orderCounter uint64
	orderMu      sync.Mutex
)

// GenerateOrderNo 生成订单编号（BCD[10]）
// 格式: yyyyMMddHHmmss + 2位序号
func GenerateOrderNo() string {
	orderMu.Lock()
	orderCounter++
	seq := orderCounter
	orderMu.Unlock()

	now := time.Now().UTC()
	return fmt.Sprintf("%s%02d",
		now.Format("20060102150405"),
		seq%100)
}

// GeneratePileOrderNo 生成桩端订单编号（BCD[10]）
func GeneratePileOrderNo() string {
	return GenerateOrderNo()
}

// GenerateStopCode 生成停止码（BCD[2]，4位数字）
func GenerateStopCode() string {
	code := rand.Intn(10000)
	return fmt.Sprintf("%04d", code)
}

// ==================== 计费数据生成 ====================

// FeeItem 计费项
type FeeItem struct {
	Hour     byte   // BCD[1] 小时
	Min      byte   // BCD[1] 分钟
	PowerFee uint32 // BYTE[3] 4位小数 (元/kWh)
	SvcFee   uint32 // BYTE[3] 4位小数 (元/kWh)
	Type     byte   // 尖峰谷平: 1尖2峰3平4谷
	LimitedP uint16 // 限制功率 kW
}

// PeakValleyType 峰谷类型
const (
	PeakValleySharp byte = 1 // 尖
	PeakValleyPeak  byte = 2 // 峰
	PeakValleyFlat  byte = 3 // 平
	PeakValleyValley byte = 4 // 谷
)

// PeakValleyNames 峰谷类型名称
var PeakValleyNames = map[byte]string{
	PeakValleySharp:  "sharp",
	PeakValleyPeak:   "peak",
	PeakValleyFlat:   "flat",
	PeakValleyValley: "valley",
}

// GenerateBillingRules 生成计费规则（24小时费率表）
func GenerateBillingRules() []FeeItem {
	rules := make([]FeeItem, 0, 24)

	hourTypes := [24]byte{
		4, 4, 4, 4, 4, 3, 3, 3, // 0-7点: 谷/平
		2, 2, 1, 1, 1, 1, 1, 1, // 8-15点: 峰/尖
		1, 1, 2, 2, 2, 3, 3, 4, // 16-23点: 尖/峰/平/谷
	}

	for hour := 0; hour < 24; hour++ {
		rules = append(rules, FeeItem{
			Hour:     byteToBCD(byte(hour)),
			Min:      0x00,
			PowerFee: priceForType(hourTypes[hour]),
			SvcFee:   serviceFeeForType(hourTypes[hour]),
			Type:     hourTypes[hour],
			LimitedP: 0,
		})
	}

	return rules
}

func priceForType(t byte) uint32 {
	switch t {
	case PeakValleySharp:
		return 15000 // 1.5000
	case PeakValleyPeak:
		return 10000 // 1.0000
	case PeakValleyFlat:
		return 6000
	case PeakValleyValley:
		return 3000
	default:
		return 6000
	}
}

func serviceFeeForType(_ byte) uint32 {
	return 800 // 0.0800
}

func byteToBCD(b byte) byte {
	return ((b / 10) << 4) | (b % 10)
}

// ==================== 充电参数 ====================

// ChargingLimits 充电限制参数
type ChargingLimits struct {
	MaxChargeAmount float64
	MaxChargeEnergy float64
	MaxChargeTime   uint16
	MaxSOC          byte
}

func DefaultChargingLimits() ChargingLimits {
	return ChargingLimits{
		MaxChargeAmount: 20000,
		MaxChargeEnergy: 10000,
		MaxChargeTime:   480,
		MaxSOC:          100,
	}
}

// ==================== 时间工具 ====================

// UTCTimeBCD 当前UTC时间BCD[6]: yy-mm-dd-hh-mm-ss (base 2000)
func UTCTimeBCD() []byte {
	now := time.Now().UTC()
	return []byte{
		byte(now.Year() - 2000),
		byte(now.Month()),
		byte(now.Day()),
		byte(now.Hour()),
		byte(now.Minute()),
		byte(now.Second()),
	}
}

// UTCTimeBCD7 当前UTC时间BCD[7]: yy-mm-dd-hh-mm-ss-00
func UTCTimeBCD7() []byte {
	now := time.Now().UTC()
	return []byte{
		byte(now.Year() - 2000),
		byte(now.Month()),
		byte(now.Day()),
		byte(now.Hour()),
		byte(now.Minute()),
		byte(now.Second()),
		0x00,
	}
}
