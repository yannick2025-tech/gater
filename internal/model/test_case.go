package model

import "time"

// TestCase 测试用例（场景→用例→报文明细 三级结构中的用例级）
type TestCase struct {
	BaseModel
	SessionID    string    `gorm:"index:idx_session_scenario;size:64;not null" json:"sessionId"`
	ScenarioID   string    `gorm:"index:idx_session_scenario;size:64;not null" json:"scenarioId"`
	ScenarioName string    `gorm:"size:64;not null" json:"scenarioName"`
	CaseID       string    `gorm:"index;size:32;not null" json:"caseId"`   // 用例编号: TC-BC-01
	CaseName     string    `gorm:"size:128;not null" json:"caseName"`      // 用例名: 单时段费率充电
	CaseType     string    `gorm:"size:32;not null" json:"caseType"`       // charging/config/upgrade
	Status       string    `gorm:"size:32;not null;default:pending" json:"status"` // pending/running/completed/skipped
	Result       string    `gorm:"size:16;not null" json:"result"`         // pass/fail/error/skipped
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
	DurationMs   int64     `json:"durationMs"`
	TotalMessages int      `json:"totalMessages"`
	SuccessCount int       `json:"successCount"`
	DecodeFail   int       `json:"decodeFail"`
	InvalidField int       `json:"invalidField"`
	BusinessFail int       `json:"businessFail"`
	SuccessRate  float64   `json:"successRate"`
	ErrorSummary string    `gorm:"size:512" json:"errorSummary"`
}

// TableName 表名
func (TestCase) TableName() string {
	return "test_cases"
}
