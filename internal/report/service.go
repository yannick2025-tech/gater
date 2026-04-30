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
	"github.com/yannick2025-tech/nts-gater/internal/session"
)

// GenerateScenarioID 生成测试场景UUID（每次startTest调用一次）
func GenerateScenarioID() string {
	return uuid.New().String()
}

// ==================== Session 持久化 ====================

// SaveSession 持久化 Session 到 DB（实时）
func SaveSession(sess *session.Session) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	s := &model.Session{
		ID:           sess.ID,
		PostNo:       sess.PostNo,
		ConnID:       sess.ConnID,
		AuthState:    sess.GetAuthState().String(),
		IsOnline:     sess.IsConnected(),
		ProtocolName: "XX标准协议",
		ProtocolVer:  "v1.6.0",
	}

	// 使用 upsert：如果已存在则更新
	result := db.Where("id = ?", s.ID).Assign(s).FirstOrCreate(s)
	return result.Error
}

// UpdateSessionOnline 更新 Session 在线状态
func UpdateSessionOnline(sessionID string, isOnline bool) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	updates := map[string]interface{}{
		"is_online":  isOnline,
		"updated_at": time.Now(),
	}
	if !isOnline {
		now := time.Now()
		updates["closed_at"] = &now
	}
	return db.Model(&model.Session{}).Where("id = ?", sessionID).Updates(updates).Error
}

// UpdateSessionAuthState 更新 Session 认证状态
func UpdateSessionAuthState(sessionID string, authState string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	return db.Model(&model.Session{}).Where("id = ?", sessionID).
		Updates(map[string]interface{}{"auth_state": authState, "updated_at": time.Now()}).Error
}

// CleanStaleSessions 启动时清理僵尸 session（上次服务崩溃未正常关闭的）
func CleanStaleSessions() error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	now := time.Now()
	result := db.Model(&model.Session{}).Where("is_online = ?", true).
		Updates(map[string]interface{}{"is_online": false, "closed_at": &now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		fmt.Printf("[CleanStaleSessions] cleaned %d stale sessions\n", result.RowsAffected)
	}
	return nil
}

// GetSessionsFromDB 从 DB 获取所有 session（分页）
func GetSessionsFromDB(page int, pageSize int) ([]model.Session, int64, error) {
	db := database.GetDB()
	if db == nil {
		return nil, 0, fmt.Errorf("database not initialized")
	}

	var total int64
	db.Model(&model.Session{}).Count(&total)

	var sessions []model.Session
	offset := (page - 1) * pageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&sessions).Error; err != nil {
		return nil, 0, err
	}
	return sessions, total, nil
}

// GetAllSessionsFromDB 从 DB 获取所有 session
func GetAllSessionsFromDB() ([]model.Session, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var sessions []model.Session
	if err := db.Order("created_at DESC").Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// ==================== TestReport 持久化 ====================

// CreateRunningReport 在测试开始时插入一条 running 占位记录到数据库
func CreateRunningReport(sessionID string, postNo uint32, testCase string, scenarioID string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	now := time.Now()

	protocolName := "XX标准协议"
	scenarioName := ""
	switch testCase {
	case "basic_charging":
		protocolName = "XX标准协议-业务场景测试"
		scenarioName = "业务场景测试"
	case "sftp_upgrade":
		protocolName = "XX标准协议-SFTP升级测试"
		scenarioName = "SFTP升级测试"
	case "config_download":
		protocolName = "XX标准协议-配置下发测试"
		scenarioName = "配置下发测试"
	}

	report := &model.TestReport{
		SessionID:    sessionID,
		ScenarioID:   scenarioID,
		ScenarioName: scenarioName,
		PostNo:       postNo,
		ProtocolName: protocolName,
		ProtocolVer:  "v1.6.0",
		StartTime:    now,
		// EndTime: nil （测试运行中，结束时间为 NULL）
		Status:       "running",
		AuthState:    "",
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
	endTime := summary.EndTime // time.Time → *time.Time for the model
	updates := map[string]interface{}{
		"end_time":       &endTime,
		"duration_ms":    summary.Duration.Milliseconds(),
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

	// ★ 兜底：如果该 session 没有 running 记录可更新（例如 CreateRunningReport 之前失败），
	// 则直接创建一条 completed 记录，确保 test_reports 表不缺失
	if result.RowsAffected == 0 {
		endTime := summary.EndTime
		fallback := &model.TestReport{
			SessionID:     summary.SessionID,
			ScenarioID:    "",
			ScenarioName:  "",
			PostNo:        summary.PostNo,
			ProtocolName:  protocolName,
			ProtocolVer:   protocolVer,
			StartTime:     summary.StartTime,
			EndTime:       &endTime,
			DurationMs:    summary.Duration.Milliseconds(),
			TotalMessages: summary.TotalMessages,
			SuccessTotal:  summary.SuccessTotal,
			FailTotal:     summary.FailTotal,
			SuccessRate:   summary.SuccessRate,
			IsPass:        summary.IsPass,
			Status:        "completed",
			AuthState:     authState,
		}
		if err := db.Create(fallback).Error; err != nil {
			fmt.Printf("[SaveReport] create fallback report warning: %v\n", err)
			// 不返回错误，因为 func_code_stats 和 message_archives 已写入成功
		} else {
			fmt.Printf("[SaveReport] created fallback report for session %s\n", summary.SessionID)
		}
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
			BusinessFail:  stat.BusinessFail,
			SuccessRate:   stat.SuccessRate(),
		}
		if err := db.Create(fcStat).Error; err != nil {
			return fmt.Errorf("save func code stat failed: %w", err)
		}
	}

	// 3. 批量刷写 Recorder 的所有报文记录到 message_archives（兜底补偿）
	if rec != nil {
		records := rec.GetAllRecords()
		if len(records) > 0 {
			archives := make([]model.MessageArchive, len(records))
			for i, r := range records {
				archives[i] = model.MessageArchive{
					SessionID: summary.SessionID,
					CaseID:    r.CaseID,
					FuncCode:  recorder.FormatFuncCode(r.FuncCode),
					Direction: recorder.FormatDirection(r.Direction),
					Status:    string(r.Status),
					HexData:   r.HexData,
					JSONData:  r.JSONData,
					ErrorMsg:  r.ErrorMsg,
					Timestamp: r.Timestamp,
				}
			}
			if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&archives).Error; err != nil {
				fmt.Printf("[SaveReport] batch flush message archives warning: %v\n", err)
			} else {
				fmt.Printf("[SaveReport] batch flushed %d message archives for session %s\n", len(archives), summary.SessionID)
			}
		}
	}

	// 4. 聚合用例级统计 → 更新 test_cases 表
	aggregateTestCases(db, summary.SessionID)

	// 5. 聚合场景级统计 → 更新 test_reports 表
	aggregateTestReports(db, summary.SessionID)

	// 6. 同步写入 ClosedAt（不依赖 sessMgr.Remove 的异步 onRemove 回调，
	//    确保报告生成时 ClosedAt 已落盘）
	now := time.Now()
	db.Model(&model.Session{}).Where("id = ? AND (closed_at IS NULL OR closed_at < ?)", summary.SessionID, now).
		Update("closed_at", &now)

	return nil
}

// aggregateTestCases 聚合用例级统计
func aggregateTestCases(db *gorm.DB, sessionID string) {
	// 获取该 session 下所有 test_cases
	var cases []model.TestCase
	if err := db.Where("session_id = ?", sessionID).Find(&cases).Error; err != nil {
		return
	}

	for _, tc := range cases {
		// 按 case_id 统计报文
		var msgStats struct {
			Total int
			Ok    int
			DF    int
			IF    int
			BF    int
		}
		db.Model(&model.MessageArchive{}).
			Where("session_id = ? AND case_id = ?", sessionID, tc.CaseID).
			Select("COUNT(*) as total").
			Select("SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as ok").
			Select("SUM(CASE WHEN status = 'decode_fail' THEN 1 ELSE 0 END) as df").
			Select("SUM(CASE WHEN status = 'invalid_field' THEN 1 ELSE 0 END) as if").
			Select("SUM(CASE WHEN status = 'business_fail' THEN 1 ELSE 0 END) as bf").
			Scan(&msgStats)

		successRate := 0.0
		if msgStats.Total > 0 {
			successRate = float64(msgStats.Ok) / float64(msgStats.Total) * 100
		}

		result := "pass"
		if msgStats.Total > 0 && successRate < 100 {
			result = "fail"
		}

		db.Model(&model.TestCase{}).Where("id = ?", tc.ID).Updates(map[string]interface{}{
			"total_messages": msgStats.Total,
			"success_count":  msgStats.Ok,
			"decode_fail":    msgStats.DF,
			"invalid_field":  msgStats.IF,
			"business_fail":  msgStats.BF,
			"success_rate":   successRate,
			"result":         result,
			"status":         "completed",
			"end_time":       time.Now(),
		})
	}
}

// aggregateTestReports 聚合场景级统计
func aggregateTestReports(db *gorm.DB, sessionID string) {
	var reports []model.TestReport
	if err := db.Where("session_id = ?", sessionID).Find(&reports).Error; err != nil {
		return
	}

	for _, r := range reports {
		var caseStats struct {
			Total   int
			Passed  int
			Failed  int
			Skipped int
		}
		db.Model(&model.TestCase{}).
			Where("session_id = ? AND scenario_id = ?", sessionID, r.ScenarioID).
			Select("COUNT(*) as total").
			Select("SUM(CASE WHEN result = 'pass' THEN 1 ELSE 0 END) as passed").
			Select("SUM(CASE WHEN result = 'fail' THEN 1 ELSE 0 END) as failed").
			Select("SUM(CASE WHEN result = 'skipped' THEN 1 ELSE 0 END) as skipped").
			Scan(&caseStats)

		var failStats struct {
			DF int
			IF int
			BF int
		}
		db.Model(&model.MessageArchive{}).
			Where("session_id = ?", sessionID).
			Select("SUM(CASE WHEN status = 'decode_fail' THEN 1 ELSE 0 END) as df").
			Select("SUM(CASE WHEN status = 'invalid_field' THEN 1 ELSE 0 END) as if").
			Select("SUM(CASE WHEN status = 'business_fail' THEN 1 ELSE 0 END) as bf").
			Scan(&failStats)

		db.Model(&model.TestReport{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
			"total_cases":         caseStats.Total,
			"passed_cases":        caseStats.Passed,
			"failed_cases":        caseStats.Failed,
			"skipped_cases":       caseStats.Skipped,
			"decode_fail_count":   failStats.DF,
			"invalid_field_count": failStats.IF,
			"business_fail_count": failStats.BF,
		})
	}
}

// ==================== TestCase 持久化 ====================

// SaveTestCase 创建或更新用例记录（实时）
func SaveTestCase(tc *model.TestCase) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// 按 (session_id, case_id) 查找已有记录，存在则更新
	var existing model.TestCase
	result := db.Where("session_id = ? AND case_id = ?", tc.SessionID, tc.CaseID).First(&existing)
	if result.Error == nil {
		// 已存在，更新
		return db.Model(&existing).Updates(map[string]interface{}{
			"scenario_id":   tc.ScenarioID,
			"scenario_name": tc.ScenarioName,
			"case_name":     tc.CaseName,
			"case_type":     tc.CaseType,
			"status":        tc.Status,
			"start_time":    tc.StartTime,
		}).Error
	}
	// 不存在，创建
	return db.Create(tc).Error
}

// UpdateTestCaseStatus 更新用例状态
func UpdateTestCaseStatus(sessionID string, caseID string, status string, result string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	updates := map[string]interface{}{"status": status, "updated_at": time.Now()}
	if result != "" {
		updates["result"] = result
	}
	if status == "completed" {
		updates["end_time"] = time.Now()
	}
	return db.Model(&model.TestCase{}).
		Where("session_id = ? AND case_id = ?", sessionID, caseID).
		Updates(updates).Error
}

// GetTestCasesBySessionID 获取某 session 下所有用例
func GetTestCasesBySessionID(sessionID string) ([]model.TestCase, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var cases []model.TestCase
	if err := db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&cases).Error; err != nil {
		return nil, err
	}
	return cases, nil
}

// GetTestCasesByScenarioID 获取某场景下所有用例
func GetTestCasesByScenarioID(sessionID string, scenarioID string) ([]model.TestCase, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var cases []model.TestCase
	if err := db.Where("session_id = ? AND scenario_id = ?", sessionID, scenarioID).Order("created_at ASC").Find(&cases).Error; err != nil {
		return nil, err
	}
	return cases, nil
}

// ==================== ValidationResult 持久化 ====================

// SaveValidationResult 保存校验结果（实时）
func SaveValidationResult(vr *model.ValidationResult) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	return db.Create(vr).Error
}

// GetValidationResultsBySessionID 获取某 session 下所有校验结果
func GetValidationResultsBySessionID(sessionID string) ([]model.ValidationResult, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var results []model.ValidationResult
	if err := db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// GetValidationResultsByCaseID 获取某用例下所有校验结果
func GetValidationResultsByCaseID(sessionID string, caseID string) ([]model.ValidationResult, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var results []model.ValidationResult
	if err := db.Where("session_id = ? AND case_id = ?", sessionID, caseID).Order("created_at ASC").Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// ==================== MessageArchive 持久化 ====================

// SaveMessageArchive 保存单条消息存档（去重写入）
// 规则：同 (session_id, case_id, func_code, direction, status) 只保留最新一条
func SaveMessageArchive(sessionID string, rec recorder.MessageRecord) error {
	db := database.GetDB()
	if db == nil {
		return nil // 数据库未初始化时不报错
	}

	archive := &model.MessageArchive{
		SessionID: sessionID,
		CaseID:    rec.CaseID,
		FuncCode:  recorder.FormatFuncCode(rec.FuncCode),
		Direction: recorder.FormatDirection(rec.Direction),
		Status:    string(rec.Status),
		HexData:   rec.HexData,
		JSONData:  rec.JSONData,
		ErrorMsg:  rec.ErrorMsg,
		Timestamp: rec.Timestamp,
	}

	return UpsertMessageArchive(archive)
}

// UpsertMessageArchive 去重写入报文存档
// 规则：同一 (session_id, case_id, func_code, direction, status) 只保留最新一条
func UpsertMessageArchive(archive *model.MessageArchive) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// 先删后插（简单可靠）
	db.Where(
		"session_id = ? AND case_id = ? AND func_code = ? AND direction = ? AND status = ?",
		archive.SessionID, archive.CaseID, archive.FuncCode, archive.Direction, archive.Status,
	).Delete(&model.MessageArchive{})

	return db.Create(archive).Error
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

// GetMessageArchivesByCaseID 根据 caseID 获取消息存档
func GetMessageArchivesByCaseID(sessionID string, caseID string) ([]model.MessageArchive, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var archives []model.MessageArchive
	if err := db.Where("session_id = ? AND case_id = ?", sessionID, caseID).
		Order("timestamp ASC").Find(&archives).Error; err != nil {
		return nil, err
	}
	return archives, nil
}

// ==================== FuncCodeStat 持久化 ====================

// UpsertFuncCodeStat 创建或更新功能码统计（实时）
func UpsertFuncCodeStat(stat *model.FuncCodeStat) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// 按 (session_id, func_code, direction, case_id) 查找，存在则更新
	var existing model.FuncCodeStat
	result := db.Where("session_id = ? AND func_code = ? AND direction = ? AND (case_id = ? OR (case_id = '' AND ? = ''))",
		stat.SessionID, stat.FuncCode, stat.Direction, stat.CaseID, stat.CaseID).First(&existing)

	if result.Error == nil {
		// 已存在，更新
		return db.Model(&existing).Updates(map[string]interface{}{
			"total_messages": stat.TotalMessages,
			"success_count":  stat.SuccessCount,
			"decode_fail":    stat.DecodeFail,
			"invalid_field":  stat.InvalidField,
			"message_loss":   stat.MessageLoss,
			"business_fail":  stat.BusinessFail,
			"success_rate":   stat.SuccessRate,
		}).Error
	}
	// 不存在，创建
	return db.Create(stat).Error
}

// ==================== 通用查询 ====================

// GetTestReports 查询测试报告列表
func GetTestReports(page int, pageSize int, startTime *time.Time, endTime *time.Time, sessionID string) ([]model.TestReport, int64, error) {
	db := database.GetDB()
	if db == nil {
		return nil, 0, fmt.Errorf("database not initialized")
	}

	query := db.Model(&model.TestReport{})
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
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

// GetDeviceOnlineStatus 获取设备在线状态
func GetDeviceOnlineStatus(postNo uint32) bool {
	db := database.GetDB()
	if db == nil {
		return false
	}

	var s model.Session
	err := db.Where("post_no = ? AND is_online = ?", postNo, true).First(&s).Error
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

// GetAllSessionSummaries 获取所有会话摘要（从 sessions 表读取）
func GetAllSessionSummaries() ([]model.Session, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var sessions []model.Session
	if err := db.Order("created_at DESC").Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// Ensure interface compliance
var _ = (*gorm.DB)(nil)
