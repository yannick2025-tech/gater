package standard

import (
	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard/msg"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
)

// registerMessages 注册所有功能码消息
func (p *StandardProtocol) registerMessages() {
	r := p.registry

	// 0x0A 接入认证-随机数
	r.Register(types.FuncAuthRandom, types.DirectionUpload, func() types.Message { return &msg.Auth0AUpload{} })
	r.Register(types.FuncAuthRandom, types.DirectionReply, func() types.Message { return &msg.Auth0AReply{} })

	// 0x0B 接入认证-加密数据
	r.Register(types.FuncAuthEncrypted, types.DirectionUpload, func() types.Message { return &msg.Auth0BUpload{} })
	r.Register(types.FuncAuthEncrypted, types.DirectionReply, func() types.Message { return &msg.Auth0BReply{} })

	// 0x21 秘钥更新
	r.Register(types.FuncKeyUpdate, types.DirectionDownload, func() types.Message { return &msg.KeyUpdateDownload{} })
	r.Register(types.FuncKeyUpdate, types.DirectionReply, func() types.Message { return &msg.KeyUpdateReply{} })

	// 0x23 对时
	r.Register(types.FuncTimeSync, types.DirectionUpload, func() types.Message { return &msg.TimeSyncUpload{} })
	r.Register(types.FuncTimeSync, types.DirectionReply, func() types.Message { return &msg.TimeSyncReply{} })

	// 0x01 心跳
	r.Register(types.FuncHeartbeat, types.DirectionUpload, func() types.Message { return &msg.HeartbeatUpload{} })
	r.Register(types.FuncHeartbeat, types.DirectionReply, func() types.Message { return &msg.HeartbeatReply{} })

	// 0x22 分时段计费规则
	r.Register(types.FuncBillingRules, types.DirectionDownload, func() types.Message { return &msg.BillingRulesDownload{} })
	r.Register(types.FuncBillingRules, types.DirectionReply, func() types.Message { return &msg.BillingRulesReply{} })

	// 0x03 平台充电启动请求
	r.Register(types.FuncPlatformStart, types.DirectionDownload, func() types.Message { return &msg.PlatformStartDownload{} })

	// 0x04 桩充电启动请求
	r.Register(types.FuncChargerStart, types.DirectionUpload, func() types.Message { return &msg.ChargerStartUpload{} })
	r.Register(types.FuncChargerStart, types.DirectionReply, func() types.Message { return &msg.ChargerStartReply{} })

	// 0x05 桩充电停止请求
	r.Register(types.FuncChargerStop, types.DirectionUpload, func() types.Message { return &msg.ChargerStopUpload{} })
	r.Register(types.FuncChargerStop, types.DirectionReply, func() types.Message { return &msg.ChargerStopReply{} })

	// 0x06 充电数据上报
	r.Register(types.FuncChargingData, types.DirectionUpload, func() types.Message { return &msg.ChargingDataUpload{} })
	r.Register(types.FuncChargingData, types.DirectionReply, func() types.Message { return &msg.ChargingDataReply{} })

	// 0x07 预约充电
	r.Register(types.FuncReservation, types.DirectionDownload, func() types.Message { return &msg.ReservationDownload{} })
	r.Register(types.FuncReservation, types.DirectionReply, func() types.Message { return &msg.ReservationReply{} })

	// 0x08 平台充电停止请求
	r.Register(types.FuncPlatformStop, types.DirectionDownload, func() types.Message { return &msg.PlatformStopDownload{} })

	// 0x0C 设备参数查询
	r.Register(types.FuncDeviceQuery, types.DirectionDownload, func() types.Message { return &msg.DeviceQueryDownload{} })
	r.Register(types.FuncDeviceQuery, types.DirectionReply, func() types.Message { return &msg.DeviceQueryReply{} })

	// 0xC1 设备参数上报
	r.Register(types.FuncParamReport, types.DirectionUpload, func() types.Message { return &msg.ParamReportUpload{} })
	r.Register(types.FuncParamReport, types.DirectionReply, func() types.Message { return &msg.ParamReportReply{} })

	// 0xC2 配置信息下发
	r.Register(types.FuncConfigDownload, types.DirectionDownload, func() types.Message { return &msg.ConfigDownloadMsg{} })
	r.Register(types.FuncConfigDownload, types.DirectionReply, func() types.Message { return &msg.ConfigDownloadReply{} })

	// 0xF6 SFTP升级
	r.Register(types.FuncSFTPUpgrade, types.DirectionDownload, func() types.Message { return &msg.SFTPUpgradeDownload{} })
	r.Register(types.FuncSFTPUpgrade, types.DirectionReply, func() types.Message { return &msg.SFTPUpgradeReply{} })

	// 0xF7 升级进度
	r.Register(types.FuncUpgradeProgress, types.DirectionUpload, func() types.Message { return &msg.UpgradeProgressUpload{} })

	// 0x10 BMS静态数据
	r.Register(types.FuncBMSStatic, types.DirectionUpload, func() types.Message { return &msg.BMSStaticUpload{} })
	r.Register(types.FuncBMSStatic, types.DirectionReply, func() types.Message { return &msg.BMSStaticReply{} })

	// 0x24 充电中BMS数据
	r.Register(types.FuncBMSCharging, types.DirectionUpload, func() types.Message { return &msg.BMSChargingUpload{} })
	r.Register(types.FuncBMSCharging, types.DirectionReply, func() types.Message { return &msg.BMSChargingReply{} })

	// 0x25 BMV电压
	r.Register(types.FuncBMVVoltage, types.DirectionUpload, func() types.Message { return &msg.BMVVoltageUpload{} })
	r.Register(types.FuncBMVVoltage, types.DirectionReply, func() types.Message { return &msg.BMVVoltageReply{} })

	// 0x26 BMT温度
	r.Register(types.FuncBMTTemperature, types.DirectionUpload, func() types.Message { return &msg.BMTTemperatureUpload{} })
	r.Register(types.FuncBMTTemperature, types.DirectionReply, func() types.Message { return &msg.BMTTemperatureReply{} })

	// 0x27 BSP预留
	r.Register(types.FuncBSPReserved, types.DirectionUpload, func() types.Message { return &msg.BSPReservedUpload{} })
	r.Register(types.FuncBSPReserved, types.DirectionReply, func() types.Message { return &msg.BSPReservedReply{} })

	// 0x16 占位订单
	r.Register(types.FuncOccupancy, types.DirectionUpload, func() types.Message { return &msg.OccupancyUpload{} })
	r.Register(types.FuncOccupancy, types.DirectionReply, func() types.Message { return &msg.OccupancyReply{} })

	// 0x28 屏显模式
	r.Register(types.FuncScreenDisplay, types.DirectionDownload, func() types.Message { return &msg.ScreenDisplayDownload{} })
	r.Register(types.FuncScreenDisplay, types.DirectionReply, func() types.Message { return &msg.ScreenDisplayReply{} })
}
