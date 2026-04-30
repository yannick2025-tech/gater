// Package handlers provides charging business handlers.
package handlers

import (
	"fmt"

	"github.com/yannick2025-tech/nts-gater/internal/dispatcher"
	"github.com/yannick2025-tech/nts-gater/internal/generator"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/standard/msg"
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
	"github.com/yannick2025-tech/nts-gater/internal/session"
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

	// 读取WEB端传入的充电参数
	balance, displayMode, targetSOC, energy := ctx.Session.GetChargingParams()

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
		// 存储下发的峰谷类型，供0x06校验使用
		peakTypes := make([]byte, len(msgFees))
		for i, f := range msgFees {
			peakTypes[i] = f.Type
		}
		ctx.Session.SetSentPeakTypes(peakTypes)
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
		AccountBalance:              uint32(balance * 100), // 前端余额(元)→协议值(分)
		LimitTheMaximumChargeCharge: limits.MaxChargeAmount,
		LimitTheChargingTime:        limits.MaxChargeTime,
		LimitTheAmountOfCharging:    func() float64 {
			if energy > 0 {
				return energy // 前端配置的充电电量(kWh)
			}
			return limits.MaxChargeEnergy
		}(),
		LimitChargingServiceFees:    5000,
		LimitChargingCharges:        5000,
		LimitSOC:                    func() byte {
			if targetSOC > 0 && targetSOC <= 100 {
				return targetSOC // 前端配置的SOC目标
			}
			return limits.MaxSOC
		}(),
		ErrorCode:                   0x00,
		AuthenticationNumber:        startMsg.AuthenticationNumber,
		TheScreenDisplayModeNumber:  displayMode, // 前端配置的屏显模式
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

	// 更新充电状态：充电桩充电开始/结束时间、停止SOC
	ctx.Session.SetChargingStopped(stopMsg.ChargeStartTime, stopMsg.ChargeEndTime, stopMsg.StopSoc)

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

	currentElecKWh := float64(dataMsg.CurrentElec) / 10000

	// 更新充电状态：电量、SOC、订单号
	ctx.Session.UpdateChargingData(
		currentElecKWh,
		dataMsg.CurrentSOC,
		dataMsg.ChargingOrderNumber,
		dataMsg.ToJSONMap(),
	)

	// ===== 0x06 校验 =====
	sid := ctx.Session.ID
	cs := ctx.Session.GetChargingState()
	if cs != nil && cs.ChargingDataCount > 1 {
		// 1. 电量递增校验
		if currentElecKWh < cs.LastElec {
			ctx.Session.AddValidationResult(0x06, "电量递增", false,
				fmt.Sprintf("currentElec=%.4f < lastElec=%.4f", currentElecKWh, cs.LastElec))
			h.logger.Warnf("[%s] [GATER] [0x06] 校验 电量递增 FAIL: current=%.4f < last=%.4f", sid, currentElecKWh, cs.LastElec)
		} else {
			ctx.Session.AddValidationResult(0x06, "电量递增", true,
				fmt.Sprintf("currentElec=%.4f >= lastElec=%.4f", currentElecKWh, cs.LastElec))
			h.logger.Infof("[%s] [GATER] [0x06] 校验 电量递增 PASS: current=%.4f >= last=%.4f", sid, currentElecKWh, cs.LastElec)
		}

		// 2. 分时段累计信息校验
		for i, item := range dataMsg.OverTimeAccumulateInformationList {
			// 2a. 计费校验: electricityPrices * electricity / 10000 ≈ electricityFee
			expectedElecFee := float64(item.ElectricityPrices) * float64(item.Electricity) / 10000
			actualElecFee := float64(item.ElectricityFee)
			if diff := expectedElecFee - actualElecFee; diff < -1 || diff > 1 {
				ctx.Session.AddValidationResult(0x06, fmt.Sprintf("电费计算[时段%d]", i), false,
					fmt.Sprintf("expected=%.2f actual=%.2f", expectedElecFee, actualElecFee))
				h.logger.Warnf("[%s] [GATER] [0x06] 校验 电费计算[时段%d] FAIL: expected=%.2f actual=%.2f", sid, i, expectedElecFee, actualElecFee)
			} else {
				ctx.Session.AddValidationResult(0x06, fmt.Sprintf("电费计算[时段%d]", i), true,
					fmt.Sprintf("expected=%.2f actual=%.2f", expectedElecFee, actualElecFee))
				h.logger.Infof("[%s] [GATER] [0x06] 校验 电费计算[时段%d] PASS: expected=%.2f actual=%.2f", sid, i, expectedElecFee, actualElecFee)
			}

			// 2b. 服务费校验: serviceChargePrice * electricity / 10000 ≈ serviceCharge
			expectedSvcFee := float64(item.ServiceChargePrice) * float64(item.Electricity) / 10000
			actualSvcFee := float64(item.ServiceCharge)
			if diff := expectedSvcFee - actualSvcFee; diff < -1 || diff > 1 {
				ctx.Session.AddValidationResult(0x06, fmt.Sprintf("服务费计算[时段%d]", i), false,
					fmt.Sprintf("expected=%.2f actual=%.2f", expectedSvcFee, actualSvcFee))
				h.logger.Warnf("[%s] [GATER] [0x06] 校验 服务费计算[时段%d] FAIL: expected=%.2f actual=%.2f", sid, i, expectedSvcFee, actualSvcFee)
			} else {
				ctx.Session.AddValidationResult(0x06, fmt.Sprintf("服务费计算[时段%d]", i), true,
					fmt.Sprintf("expected=%.2f actual=%.2f", expectedSvcFee, actualSvcFee))
				h.logger.Infof("[%s] [GATER] [0x06] 校验 服务费计算[时段%d] PASS: expected=%.2f actual=%.2f", sid, i, expectedSvcFee, actualSvcFee)
			}

			// 2c. 峰谷标识校验：根据上报时段EndTime的HHmm去session费率配置中查找对应类型
			prices := ctx.Session.GetPrices()
			if len(prices) > 0 {
				gaterType := h.findPeakTypeByEndTime(item.EndTime, prices)
				if gaterType != item.PeaksValleysFlag {
					ctx.Session.AddValidationResult(0x06, fmt.Sprintf("峰谷标识[时段%d]", i), false,
						fmt.Sprintf("gaterType=%d pileType=%d", gaterType, item.PeaksValleysFlag))
					h.logger.Warnf("[%s] [GATER] [0x06] 校验 峰谷标识[时段%d] FAIL: gater=%d pile=%d", sid, i, gaterType, item.PeaksValleysFlag)
				} else {
					ctx.Session.AddValidationResult(0x06, fmt.Sprintf("峰谷标识[时段%d]", i), true,
						fmt.Sprintf("gaterType=%d", gaterType))
					h.logger.Infof("[%s] [GATER] [0x06] 校验 峰谷标识[时段%d] PASS: gater=%d pile=%d", sid, i, gaterType, item.PeaksValleysFlag)
				}
			}
		}
	}

	// 回复确认
	reply := &msg.ChargingDataReply{Confirm: 0x00}
	return ctx.ReplyMessage(reply)
}

// findPeakTypeByEndTime 根据0x06上报时段的EndTime(HHmm)在session费率配置中精确匹配峰谷类型
//
// 0x06上报的EndTime格式: BCD[6]解码后的字符串如 "202604282359"(YYMMDDHHmmSS)
// 取HHmm部分(如 "2359")，去session prices中找 EndTime=="23:59" 的规则，返回其 PeakValleyType
func (h *ChargingHandler) findPeakTypeByEndTime(endTimeBCD string, prices []session.PriceConfig) byte {
	h.logger.Debugf("[charging] findPeakTypeByEndTime: endTimeBCD=%q(len=%d) prices=%d",
		endTimeBCD, len(endTimeBCD), len(prices))

	if len(endTimeBCD) < 12 {
		h.logger.Warnf("[charging] findPeakTypeByEndTime: endTimeBCD too short len=%d val=%q", len(endTimeBCD), endTimeBCD)
		return 0
	}
	hhmm := endTimeBCD[8:12] // "HHmm"

	// 将 "HHmm" 转为 "HH:mm" 用于匹配 PriceConfig.EndTime
	endTimeFormatted := hhmm[:2] + ":" + hhmm[2:]
	h.logger.Debugf("[charging]   hhmm=%s formatted=%s", hhmm, endTimeFormatted)

	for i, p := range prices {
		matched := p.EndTime == endTimeFormatted
		h.logger.Debugf("[charging]   prices[%d]: EndTime=%q PeakValleyType=%d matched=%v",
			i, p.EndTime, p.PeakValleyType, matched)
		if matched {
			return p.PeakValleyType
		}
	}

	h.logger.Warnf("[charging] findPeakTypeByEndTime: NO match for endTimeBCD=%s hhmm=%s, returning 0",
		endTimeBCD, hhmm)
	return 0
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
