package model

import "time"

// ValidationResult 业务校验结果（绑定到用例的报文明细）
type ValidationResult struct {
	BaseModel
	SessionID string    `gorm:"index:idx_session_case;size:64;not null" json:"sessionId"`
	CaseID    string    `gorm:"index:idx_session_case;size:32;not null" json:"caseId"`
	FuncCode  string    `gorm:"index:idx_func_rule;size:8;not null" json:"funcCode"`
	RuleName  string    `gorm:"index:idx_func_rule;size:128;not null" json:"ruleName"`
	Passed    bool      `gorm:"not null" json:"passed"`
	DetailMsg string    `gorm:"type:text" json:"detailMsg"`
	HexData   string    `gorm:"type:text" json:"hexData"`
	JSONData  string    `gorm:"type:text" json:"jsonData"`
	Timestamp time.Time `json:"timestamp"`
}

// TableName 表名
func (ValidationResult) TableName() string {
	return "validation_results"
}
