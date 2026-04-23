// Package handlers provides charging business handlers.
package handlers

import (
	"github.com/yannick2025-tech/nts-gater/internal/dispatcher"
	"github.com/yannick2025-tech/nts-gater/internal/generator"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard/msg"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	logging "github.com/yannick2025-tech/gwc-logging"
)

// ChargingHandler 充电业务处理器
type ChargingHandler struct {
	logger logging.Logger
}

// NewChargingHandler 创建充电处理器
func NewChargingHandler(logger logging.Logger) *ChargingHandler {
	return &ChargingHandler{logger: logger}
}

// HandleChargerStartUpload 处理充电桩上报0x04充电启动请求
// 当前为平台模式：收到桩启动请求后，自动回复允许充电
func (h *ChargingHandler) HandleChargerStartUpload(ctx *dispatcher.Context) error {
	startMsg, ok := ctx.Message.(*msg.ChargerStartUpload)
	if !ok {
		return nil
	}

	ctx.Logger.Infof("[charging] charger start request: postNo=%d deviceOrderNo=%s startupType=%d",
		ctx.PostNo, startMsg.DeviceOrderNo, startMsg.StartupType)

	// 构建回复
	limits := generator.DefaultChargingLimits()

	// 费率列表：优先从session获取WEB端传入的prices，否则使用默认值
	var msgFees []msg.FeeItem
	prices := ctx.Session.GetPrices()
	if len(prices) > 0 {
		// 将WEB端prices转为24小时费率列表（金额已放大10000倍）
		billingRules := generator.PricesToFeeItems(prices)
		msgFees = make([]msg.FeeItem, len(billingRules))
		for i, f := range billingRules {
			msgFees[i] = msg.FeeItem{
				Hour:     f.Hour,
				Min:      f.Min,
				PowerFee: f.PowerFee,
				SvcFee:   f.SvcFee,
				Type:     f.Type,
				LimitedP: f.LimitedP,
			}
		}
		h.logger.Infof("[charging] using WEB prices: %d rules from session", len(msgFees))
	} else {
		// 无WEB端配置时使用默认费率
		billingRules := generator.GenerateBillingRules()
		msgFees = make([]msg.FeeItem, len(billingRules))
		for i, f := range billingRules {
			msgFees[i] = msg.FeeItem{
				Hour:     f.Hour,
				Min:      f.Min,
				PowerFee: f.PowerFee,
				SvcFee:   f.SvcFee,
				Type:     f.Type,
				LimitedP: f.LimitedP,
			}
		}
		h.logger.Infof("[charging] using default billing rules: %d items", len(msgFees))
	}

	reply := &msg.ChargerStartReply{
		ChargingOrderNumber:         generator.GenerateOrderNo(),
		ChargingPileOrderNumber:     startMsg.DeviceOrderNo,
		AccountBalance:              50000, // 500.00元
		LimitTheMaximumChargeCharge: limits.MaxChargeAmount,
		LimitTheChargingTime:        limits.MaxChargeTime,
		LimitTheAmountOfCharging:    limits.MaxChargeEnergy,
		LimitChargingServiceFees:    5000,
		LimitChargingCharges:        5000,
		LimitSOC:                    limits.MaxSOC,
		ErrorCode:                   0x00,
		AuthenticationNumber:        startMsg.AuthenticationNumber,
		FeeNum:                      byte(len(msgFees)),
		ListFee:                     msgFees,
		StopCode:                    generator.GenerateStopCode(),
	}

	return ctx.ReplyMessage(reply)
}

// HandleChargerStopUpload 处理充电桩上报0x05充电停止
func (h *ChargingHandler) HandleChargerStopUpload(ctx *dispatcher.Context) error {
	stopMsg, ok := ctx.Message.(*msg.ChargerStopUpload)
	if !ok {
		return nil
	}

	ctx.Logger.Infof("[charging] charger stop: postNo=%d orderNo=%s reason=%d soc=%d",
		ctx.PostNo, stopMsg.ChargeOrderNo, stopMsg.StopReason, stopMsg.StopSoc)

	reply := &msg.ChargerStopReply{
		ChargeOrderNo: stopMsg.ChargeOrderNo,
		ResponseCode:  0x00, // 成功
	}
	return ctx.ReplyMessage(reply)
}

// HandleChargingDataUpload 处理充电桩上报0x06充电数据
func (h *ChargingHandler) HandleChargingDataUpload(ctx *dispatcher.Context) error {
	dataMsg, ok := ctx.Message.(*msg.ChargingDataUpload)
	if !ok {
		return nil
	}

	ctx.Logger.Debugf("[charging] charging data: postNo=%d orderNo=%s elec=%.4f soc=%d%%",
		ctx.PostNo, dataMsg.ChargingOrderNumber, dataMsg.CurrentElec, dataMsg.CurrentSOC)

	// 回复确认
	reply := &msg.ChargingDataReply{Confirm: 0x00}
	return ctx.ReplyMessage(reply)
}

// HandlePlatformStartDownload 处理平台下发0x03（下载方向，由业务API触发）
func (h *ChargingHandler) HandlePlatformStartDownload(ctx *dispatcher.Context) error {
	ctx.Logger.Infof("[charging] platform start download to postNo=%d", ctx.PostNo)
	return nil
}

// HandlePlatformStopDownload 处理平台下发0x08停止充电
func (h *ChargingHandler) HandlePlatformStopDownload(ctx *dispatcher.Context) error {
	ctx.Logger.Infof("[charging] platform stop download to postNo=%d", ctx.PostNo)
	return nil
}

// HandleReservationReply 处理充电桩回复0x07预约
func (h *ChargingHandler) HandleReservationReply(ctx *dispatcher.Context) error {
	replyMsg, ok := ctx.Message.(*msg.ReservationReply)
	if !ok {
		return nil
	}
	ctx.Logger.Infof("[charging] reservation reply: postNo=%d result=%d", ctx.PostNo, replyMsg.ResultCode)
	return nil
}

// registerChargingHandlers 注册充电相关处理器
func registerChargingHandlers(dp *dispatcher.Dispatcher, logger logging.Logger) {
	ch := NewChargingHandler(logger)

	// 0x03 平台充电启动（下载方向，由业务API触发）
	dp.RegisterFunc(types.FuncPlatformStart, types.DirectionDownload, ch.HandlePlatformStartDownload)

	// 0x04 桩充电启动请求（上传）+ 回复
	dp.RegisterFunc(types.FuncChargerStart, types.DirectionUpload, ch.HandleChargerStartUpload)

	// 0x05 桩充电停止请求（上传）+ 回复
	dp.RegisterFunc(types.FuncChargerStop, types.DirectionUpload, ch.HandleChargerStopUpload)

	// 0x06 充电数据上报（上传）+ 确认回复
	dp.RegisterFunc(types.FuncChargingData, types.DirectionUpload, ch.HandleChargingDataUpload)

	// 0x07 预约回复（充电桩回复方向）
	dp.RegisterFunc(types.FuncReservation, types.DirectionReply, ch.HandleReservationReply)

	// 0x08 平台充电停止（下载方向，由业务API触发）
	dp.RegisterFunc(types.FuncPlatformStop, types.DirectionDownload, ch.HandlePlatformStopDownload)
}
