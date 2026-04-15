// Package report provides test report persistence and query services.
package report

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/yannick2025-tech/nts-gater/internal/database"
	"github.com/yannick2025-tech/nts-gater/internal/model"
	"github.com/yannick2025-tech/nts-gater/internal/recorder"
)

// SaveReport 保存测试报告到数据库（会话结束时调用）
func SaveReport(summary *recorder.SessionSummary, protocolName string, protocolVer string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// 1. 保存测试报告
	report := &model.TestReport{
		SessionID:     summary.SessionID,
		PostNo:        summary.PostNo,
		ProtocolName:  protocolName,
		ProtocolVer:   protocolVer,
		StartTime:     summary.StartTime,
		EndTime:       summary.EndTime,
		DurationMs:    summary.Duration.Milliseconds(),
		TotalMessages: summary.TotalMessages,
		SuccessTotal:  summary.SuccessTotal,
		FailTotal:     summary.FailTotal,
		SuccessRate:   summary.SuccessRate,
		IsPass:        summary.IsPass,
		Status:        "completed",
	}

	if err := db.Create(report).Error; err != nil {
		return fmt.Errorf("save test report failed: %w", err)
	}

	// 2. 保存功能码统计
	for _, stat := range summary.FuncCodeStats {
		fcStat := &model.FuncCodeStat{
			SessionID:     summary.SessionID,
			FuncCode:      recorder.FormatFuncCode(stat.FuncCode),
			Direction:     recorder.FormatDirection(stat.Direction),
			TotalMessages: stat.TotalMessages,
			SuccessCount:  stat.SuccessCount,
			DecodeFail:    stat.DecodeFail,
			InvalidField:  stat.InvalidField,
			MessageLoss:   stat.MessageLoss,
			SuccessRate:   stat.SuccessRate(),
		}
		if err := db.Create(fcStat).Error; err != nil {
			return fmt.Errorf("save func code stat failed: %w", err)
		}
	}

	return nil
}

// SaveMessageArchive 保存单条消息存档
func SaveMessageArchive(sessionID string, rec recorder.MessageRecord) error {
	db := database.GetDB()
	if db == nil {
		return nil // 数据库未初始化时不报错
	}

	archive := &model.MessageArchive{
		SessionID: sessionID,
		FuncCode:  recorder.FormatFuncCode(rec.FuncCode),
		Direction: recorder.FormatDirection(rec.Direction),
		Status:    string(rec.Status),
		HexData:   rec.HexData,
		JSONData:  rec.JSONData,
		ErrorMsg:  rec.ErrorMsg,
		Timestamp: rec.Timestamp,
	}

	return db.Create(archive).Error
}

// GetTestReports 查询测试报告列表
func GetTestReports(page int, pageSize int, startTime *time.Time, endTime *time.Time) ([]model.TestReport, int64, error) {
	db := database.GetDB()
	if db == nil {
		return nil, 0, fmt.Errorf("database not initialized")
	}

	query := db.Model(&model.TestReport{})
	if startTime != nil {
		query = query.Where("start_time >= ?", startTime)
	}
	if endTime != nil {
		query = query.Where("start_time <= ?", endTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var reports []model.TestReport
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&reports).Error; err != nil {
		return nil, 0, err
	}

	return reports, total, nil
}

// GetTestReportBySessionID 根据SessionID获取测试报告
func GetTestReportBySessionID(sessionID string) (*model.TestReport, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var report model.TestReport
	if err := db.Where("session_id = ?", sessionID).First(&report).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

// GetFuncCodeStatsBySessionID 根据SessionID获取功能码统计
func GetFuncCodeStatsBySessionID(sessionID string) ([]model.FuncCodeStat, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var stats []model.FuncCodeStat
	if err := db.Where("session_id = ?", sessionID).Find(&stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

// GetMessageArchivesBySessionID 根据SessionID获取消息存档
func GetMessageArchivesBySessionID(sessionID string, funcCode string, status string) ([]model.MessageArchive, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := db.Where("session_id = ?", sessionID)
	if funcCode != "" {
		query = query.Where("func_code = ?", funcCode)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var archives []model.MessageArchive
	if err := query.Order("timestamp ASC").Find(&archives).Error; err != nil {
		return nil, err
	}
	return archives, nil
}

// GetDeviceOnlineStatus 获取设备在线状态
func GetDeviceOnlineStatus(postNo uint32) bool {
	// 通过数据库查找最近报告判断是否在线
	db := database.GetDB()
	if db == nil {
		return false
	}

	var report model.TestReport
	err := db.Where("post_no = ? AND status = 'active'", postNo).First(&report).Error
	return err == nil
}

// UpdateReportPDFPath 更新报告PDF路径
func UpdateReportPDFPath(sessionID string, pdfPath string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	return db.Model(&model.TestReport{}).Where("session_id = ?", sessionID).
		Update("pdf_path", pdfPath).Error
}

// Ensure interface compliance
var _ = (*gorm.DB)(nil)
