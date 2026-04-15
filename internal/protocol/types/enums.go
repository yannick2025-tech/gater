// Package types provides protocol type constants and direction/funcCode name mappings.
package types

// FuncCode 功能码常量
const (
	FuncHeartbeat       byte = 0x01 // 心跳
	FuncPlatformStart   byte = 0x03 // 平台充电启动请求
	FuncChargerStart    byte = 0x04 // 桩充电启动请求
	FuncChargerStop     byte = 0x05 // 桩充电停止请求
	FuncChargingData    byte = 0x06 // 充电数据上报
	FuncReservation     byte = 0x07 // 预约充电
	FuncPlatformStop    byte = 0x08 // 平台充电停止请求
	FuncAuthRandom      byte = 0x0A // 接入认证-随机数
	FuncAuthEncrypted   byte = 0x0B // 接入认证-加密数据
	FuncDeviceQuery     byte = 0x0C // 设备参数查询
	FuncBMSStatic       byte = 0x10 // BMS静态数据
	FuncKeyUpdate       byte = 0x21 // 秘钥更新
	FuncBillingRules    byte = 0x22 // 分时段计费规则下发
	FuncTimeSync        byte = 0x23 // 对时
	FuncBMSCharging     byte = 0x24 // 充电中BMS数据
	FuncBMVVoltage      byte = 0x25 // BMV单体电压
	FuncBMTTemperature  byte = 0x26 // BMT蓄电池温度
	FuncBSPReserved     byte = 0x27 // BSP预留报文
	FuncOccupancy       byte = 0x16 // 占位订单
	FuncScreenDisplay   byte = 0x28 // 屏显模式
	FuncParamReport     byte = 0xC1 // 设备参数上报
	FuncConfigDownload  byte = 0xC2 // 配置信息下发
	FuncSFTPUpgrade     byte = 0xF6 // SFTP升级
	FuncUpgradeProgress byte = 0xF7 // 升级进度上报
)

// DirectionNames 方向名称映射
var DirectionNames = map[Direction]string{
	DirectionUpload:   "upload",
	DirectionDownload: "download",
	DirectionReply:    "reply",
}

// FuncCodeNames 功能码名称映射
var FuncCodeNames = map[byte]string{
	FuncHeartbeat:       "heartbeat",
	FuncPlatformStart:   "platform_start_charge",
	FuncChargerStart:    "charger_start_charge",
	FuncChargerStop:     "charger_stop_charge",
	FuncChargingData:    "charging_data",
	FuncReservation:     "reservation",
	FuncPlatformStop:    "platform_stop_charge",
	FuncAuthRandom:      "access_auth_random",
	FuncAuthEncrypted:   "access_auth_encrypted",
	FuncDeviceQuery:     "device_param_query",
	FuncBMSStatic:       "bms_static_data",
	FuncKeyUpdate:       "key_update",
	FuncBillingRules:    "billing_rules",
	FuncTimeSync:        "time_sync",
	FuncBMSCharging:     "bms_charging_data",
	FuncBMVVoltage:      "bmv_voltage",
	FuncBMTTemperature:  "bmt_temperature",
	FuncBSPReserved:     "bsp_reserved",
	FuncOccupancy:       "occupancy_order",
	FuncScreenDisplay:   "screen_display",
	FuncParamReport:     "device_param_report",
	FuncConfigDownload:  "config_download",
	FuncSFTPUpgrade:     "sftp_upgrade",
	FuncUpgradeProgress: "upgrade_progress",
}
