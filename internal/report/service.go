// Package report provides test report persistence and query services.
package report

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yannick2025-tech/nts-gater/internal/database"
	"github.com/yannick2025-tech/nts-gater/internal/model"
	"github.com/yannick2025-tech/nts-gater/internal/recorder"
)

// GenerateScenarioID 生成测试场景UUID（每次startTest调用一次）
func GenerateScenarioID() string {
	return uuid.New().String()
}

// CreateRunningReport 在测试开始时插入一条 running 占位记录到数据库
func CreateRunningReport(sessionID string, postNo uint32, testCase string, scenarioID string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	now := time.Now()

	protocolName := "XX标准协议"
	switch testCase {
	case "basic_charging":
		protocolName = "XX标准协议-基础充电测试"
	case "sftp_upgrade":
		protocolName = "XX标准协议-SFTP升级测试"
	case "config_download":
		protocolName = "XX标准协议-配置下发测试"
	}

	report := &model.TestReport{
		SessionID:     sessionID,
		ScenarioID:    scenarioID,
		PostNo:        postNo,
		ProtocolName:  protocolName,
		ProtocolVer:   "v1.6.0",
		StartTime:     now,
		EndTime:       time.Time{},
		Status:        "running",
		AuthState:     "",
	}
	if err := db.Create(report).Error; err != nil {
		return fmt.Errorf("create running report failed: %w", err)
	}
	return nil
}

// UpdateScenarioStatus 更新指定测试场景的状态（如 stopTest 时标记为 completed）
func UpdateScenarioStatus(scenarioID string, status string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	now := time.Now()
	updates := map[string]interface{}{"status": status}
	if status == "completed" {
		updates["end_time"] = now
	}
	result := db.Model(&model.TestReport{}).Where("scenario_id = ?", scenarioID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update scenario status failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("scenario not found: %s", scenarioID)
	}
	return nil
}

// GetTestReportsBySessionID 获取某会话下的所有测试场景报告（一个会话可有多个场景）
func GetTestReportsBySessionID(sessionID string) ([]model.TestReport, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var reports []model.TestReport
	if err := db.Where("session_id = ?", sessionID).Order("start_time ASC").Find(&reports).Error; err != nil {
		return nil, err
	}
	return reports, nil
}
// SaveReport 保存/更新测试报告到数据库（会话断开时调用）
// 将该会话下所有 running 状态的场景报告更新为 completed
// rec 参数可选：传入时将 Recorder 中所有报文记录批量刷写到 message_archives 表（兜底补偿）
func SaveReport(summary *recorder.SessionSummary, protocolName string, protocolVer string, authState string, rec *recorder.SessionRecorder) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// 1. 批量更新该会话下所有 running 状态的报告为 completed
	updates := map[string]interface{}{
		"end_time":     summary.EndTime,
		"duration_ms":  summary.Duration.Milliseconds(),
		"total_messages": summary.TotalMessages,
		"success_total":  summary.SuccessTotal,
		"fail_total":     summary.FailTotal,
		"success_rate":   summary.SuccessRate,
		"is_pass":        summary.IsPass,
		"status":         "completed",
		"auth_state":     authState,
	}
	result := db.Model(&model.TestReport{}).
		Where("session_id = ? AND status = 'running'", summary.SessionID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("save test report failed: %w", result.Error)
	}
	fmt.Printf("[SaveReport] updated %d scenario records for session %s to completed\n", result.RowsAffected, summary.SessionID)

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

	// 3. 批量刷写 Recorder 的所有报文记录到 message_archives（兜底补偿）
	// 运行时的实时异步写入可能因竞态/时序丢失部分数据，此处确保完整性
	if rec != nil {
		records := rec.GetAllRecords()
		if len(records) > 0 {
			archives := make([]model.MessageArchive, len(records))
			for i, r := range records {
				archives[i] = model.MessageArchive{
					SessionID: summary.SessionID,
					FuncCode:  recorder.FormatFuncCode(r.FuncCode),
					Direction: recorder.FormatDirection(r.Direction),
					Status:    string(r.Status),
					HexData:   r.HexData,
					JSONData:  r.JSONData,
					ErrorMsg:  r.ErrorMsg,
					Timestamp: r.Timestamp,
				}
			}
			// 批量插入（使用 Clauses ON DUPLICATE KEY IGNORE 避免与实时写入冲突）
			if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&archives).Error; err != nil {
				fmt.Printf("[SaveReport] batch flush message archives warning: %v\n", err)
				// 不返回错误，报告本身已保存成功
			} else {
				fmt.Printf("[SaveReport] batch flushed %d message archives for session %s\n", len(archives), summary.SessionID)
			}
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

// GetAllSessionSummaries 获取所有会话摘要（用于会话列表展示，合并活跃+历史）
func GetAllSessionSummaries() ([]model.TestReport, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var reports []model.TestReport
	if err := db.Order("start_time DESC").Find(&reports).Error; err != nil {
		return nil, err
	}
	return reports, nil
}

// Ensure interface compliance
var _ = (*gorm.DB)(nil)
