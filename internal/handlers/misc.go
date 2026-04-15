package handlers

import (
	"github.com/yannick2025-tech/nts-gater/internal/dispatcher"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard/msg"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	logging "github.com/yannick2025-tech/gwc-logging"
)

// MiscHandler 其他业务处理器（配置/计费/升级/BMS等）
type MiscHandler struct {
	logger logging.Logger
}

// NewMiscHandler 创建其他业务处理器
func NewMiscHandler(logger logging.Logger) *MiscHandler {
	return &MiscHandler{logger: logger}
}

// ==================== 0x0C 设备参数 ====================

// HandleConfigUpload 处理充电桩回复0x0C配置查询结果
func (h *MiscHandler) HandleConfigUpload(ctx *dispatcher.Context) error {
	replyMsg, ok := ctx.Message.(*msg.DeviceQueryReply)
	if !ok {
		return nil
	}
	ctx.Logger.Infof("[misc] config reply: postNo=%d result=%d cmdCode=%d",
		ctx.PostNo, replyMsg.ResultCode, replyMsg.CmdCode)
	return nil
}

// ==================== 0x22 计费规则 ====================

// HandleBillingReply 处理充电桩回复0x22计费规则
func (h *MiscHandler) HandleBillingReply(ctx *dispatcher.Context) error {
	ctx.Logger.Infof("[misc] billing reply: postNo=%d", ctx.PostNo)
	return nil
}

// ==================== 0x10 BMS静态数据 ====================

// HandleBMSStaticUpload 处理充电桩上报0x10 BMS静态数据
func (h *MiscHandler) HandleBMSStaticUpload(ctx *dispatcher.Context) error {
	ctx.Logger.Debugf("[misc] BMS static data: postNo=%d", ctx.PostNo)
	reply := &msg.BMSStaticReply{Reply: 0x00}
	return ctx.ReplyMessage(reply)
}

// ==================== 0x24 BMS充电中数据 ====================

// HandleBMSChargingUpload 处理0x24
func (h *MiscHandler) HandleBMSChargingUpload(ctx *dispatcher.Context) error {
	ctx.Logger.Debugf("[misc] BMS charging data: postNo=%d", ctx.PostNo)
	reply := &msg.BMSChargingReply{Reply: 0x00}
	return ctx.ReplyMessage(reply)
}

// ==================== 0x25 BMV单体电压 ====================

// HandleBMVVoltageUpload 处理0x25
func (h *MiscHandler) HandleBMVVoltageUpload(ctx *dispatcher.Context) error {
	ctx.Logger.Debugf("[misc] BMV voltage: postNo=%d", ctx.PostNo)
	reply := &msg.BMVVoltageReply{Reply: 0x00}
	return ctx.ReplyMessage(reply)
}

// ==================== 0x26 BMT蓄电池温度 ====================

// HandleBMTTemperatureUpload 处理0x26
func (h *MiscHandler) HandleBMTTemperatureUpload(ctx *dispatcher.Context) error {
	ctx.Logger.Debugf("[misc] BMT temperature: postNo=%d", ctx.PostNo)
	reply := &msg.BMTTemperatureReply{Reply: 0x00}
	return ctx.ReplyMessage(reply)
}

// ==================== 0x27 BSP预留 ====================

// HandleBSPReservedUpload 处理0x27
func (h *MiscHandler) HandleBSPReservedUpload(ctx *dispatcher.Context) error {
	ctx.Logger.Debugf("[misc] BSP reserved: postNo=%d", ctx.PostNo)
	return nil
}

// ==================== 0x16 占位订单 ====================

// HandlePlaceholderUpload 处理0x16
func (h *MiscHandler) HandlePlaceholderUpload(ctx *dispatcher.Context) error {
	occMsg, _ := ctx.Message.(*msg.OccupancyUpload)
	ctx.Logger.Infof("[misc] placeholder order: postNo=%d orderNo=%s", ctx.PostNo, occMsg.ChargeOrderNo)
	reply := &msg.OccupancyReply{
		ChargeOrderNo: occMsg.ChargeOrderNo,
		ResponseCode:  0x00,
	}
	return ctx.ReplyMessage(reply)
}

// ==================== 0x28 屏显模式 ====================

// HandleScreenDisplayUpload 处理0x28
func (h *MiscHandler) HandleScreenDisplayUpload(ctx *dispatcher.Context) error {
	ctx.Logger.Infof("[misc] screen display: postNo=%d", ctx.PostNo)
	return nil
}

// ==================== 0xF6 SFTP升级 ====================

// HandleUpgradeUpload 处理0xF6充电桩回复升级
func (h *MiscHandler) HandleUpgradeUpload(ctx *dispatcher.Context) error {
	ctx.Logger.Infof("[misc] upgrade reply: postNo=%d", ctx.PostNo)
	return nil
}

// ==================== 0xF7 升级进度 ====================

// HandleUpgradeProgressUpload 处理0xF7升级进度上报
func (h *MiscHandler) HandleUpgradeProgressUpload(ctx *dispatcher.Context) error {
	ctx.Logger.Infof("[misc] upgrade progress: postNo=%d", ctx.PostNo)
	return nil
}

// ==================== 0xC1 设备参数上报 ====================

// HandleDeviceParamUpload 处理0xC1
func (h *MiscHandler) HandleDeviceParamUpload(ctx *dispatcher.Context) error {
	ctx.Logger.Infof("[misc] device param upload: postNo=%d", ctx.PostNo)
	return nil
}

// ==================== 0xC2 配置信息下发回复 ====================

// HandleConfigInfoReply 处理0xC2回复
func (h *MiscHandler) HandleConfigInfoReply(ctx *dispatcher.Context) error {
	ctx.Logger.Infof("[misc] config info reply: postNo=%d", ctx.PostNo)
	return nil
}

// registerMiscHandlers 注册其他业务处理器
func registerMiscHandlers(dp *dispatcher.Dispatcher, logger logging.Logger) {
	mh := NewMiscHandler(logger)

	dp.RegisterFunc(types.FuncDeviceQuery, types.DirectionReply, mh.HandleConfigUpload)
	dp.RegisterFunc(types.FuncBillingRules, types.DirectionReply, mh.HandleBillingReply)
	dp.RegisterFunc(types.FuncBMSStatic, types.DirectionUpload, mh.HandleBMSStaticUpload)
	dp.RegisterFunc(types.FuncBMSCharging, types.DirectionUpload, mh.HandleBMSChargingUpload)
	dp.RegisterFunc(types.FuncBMVVoltage, types.DirectionUpload, mh.HandleBMVVoltageUpload)
	dp.RegisterFunc(types.FuncBMTTemperature, types.DirectionUpload, mh.HandleBMTTemperatureUpload)
	dp.RegisterFunc(types.FuncBSPReserved, types.DirectionUpload, mh.HandleBSPReservedUpload)
	dp.RegisterFunc(types.FuncOccupancy, types.DirectionUpload, mh.HandlePlaceholderUpload)
	dp.RegisterFunc(types.FuncScreenDisplay, types.DirectionUpload, mh.HandleScreenDisplayUpload)
	dp.RegisterFunc(types.FuncSFTPUpgrade, types.DirectionReply, mh.HandleUpgradeUpload)
	dp.RegisterFunc(types.FuncUpgradeProgress, types.DirectionUpload, mh.HandleUpgradeProgressUpload)
	dp.RegisterFunc(types.FuncParamReport, types.DirectionUpload, mh.HandleDeviceParamUpload)
	dp.RegisterFunc(types.FuncConfigDownload, types.DirectionReply, mh.HandleConfigInfoReply)
}
