// Package model provides database models for test reports and message archives.
package model

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 基础模型
type BaseModel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}

// TestReport 测试报告（场景级汇总）
type TestReport struct {
	BaseModel
	SessionID     string  `gorm:"uniqueIndex:idx_session_scenario;size:64;not null" json:"sessionId"`
	ScenarioID    string  `gorm:"uniqueIndex:idx_session_scenario;size:64;not null" json:"scenarioId"` // 每次startTest生成的UUID，同一session可有多个scenario
	ScenarioName  string  `gorm:"size:64" json:"scenarioName"`                                       // 场景名称
	PostNo        uint32  `gorm:"index;not null" json:"postNo"`
	ProtocolName  string  `gorm:"size:64" json:"protocolName"`
	ProtocolVer   string  `gorm:"size:16" json:"protocolVersion"`
	StartTime     time.Time `gorm:"index" json:"startTime"`
	EndTime       time.Time `json:"endTime"`
	DurationMs    int64   `json:"durationMs"`         // 持续时间(毫秒)
	TotalMessages int     `json:"totalMessages"`      // 总消息数
	SuccessTotal  int     `json:"successTotal"`       // 成功总数
	FailTotal     int     `json:"failTotal"`          // 失败总数
	SuccessRate   float64 `json:"successRate"`        // 成功率(%)
	IsPass        bool    `json:"isPass"`             // 测试是否通过
	PDFApath     string  `gorm:"size:512" json:"pdfPath,omitempty"` // PDF文件路径
	Status        string  `gorm:"size:32;default:running" json:"status"` // running/completed/archived
	AuthState     string  `gorm:"size:32;default:none" json:"authState"` // none/pending/authenticated (断开前的最终认证状态)
	// 场景级用例汇总
	TotalCases        int     `json:"totalCases"`
	PassedCases       int     `json:"passedCases"`
	FailedCases       int     `json:"failedCases"`
	SkippedCases      int     `json:"skippedCases"`
	DecodeFailCount   int     `json:"decodeFailCount"`
	InvalidFieldCount int     `json:"invalidFieldCount"`
	BusinessFailCount int     `json:"businessFailCount"`
}

// TableName 表名
func (TestReport) TableName() string {
	return "test_reports"
}

// FuncCodeStat 功能码统计
type FuncCodeStat struct {
	BaseModel
	SessionID     string  `gorm:"index;size:64;not null" json:"sessionId"`
	FuncCode      string  `gorm:"size:8;not null" json:"funcCode"`    // "0x0A"
	Direction     string  `gorm:"size:32;not null" json:"direction"`  // "充电桩→平台"/"平台→充电桩"/"回复"
	TotalMessages int     `json:"totalMessages"`
	SuccessCount  int     `json:"successCount"`
	DecodeFail    int     `json:"decodeFail"`
	InvalidField  int     `json:"invalidField"`
	MessageLoss   int     `json:"messageLoss"`
	SuccessRate   float64 `json:"successRate"`
	CaseID        string  `gorm:"size:32" json:"caseId"`              // 所属用例ID(空=场景级汇总)
	ScenarioID    string  `gorm:"size:64" json:"scenarioId"`          // 所属场景ID
	BusinessFail  int     `json:"businessFail"`                       // 业务校验失败数
}

// TableName 表名
func (FuncCodeStat) TableName() string {
	return "func_code_stats"
}

// MessageArchive 消息存档（用于报文查看）
type MessageArchive struct {
	BaseModel
	SessionID string `gorm:"index;size:64;not null" json:"sessionId"`
	CaseID    string `gorm:"size:32" json:"caseId"`                     // 所属用例ID
	FuncCode  string `gorm:"size:8;not null" json:"funcCode"`
	Direction string `gorm:"size:32;not null" json:"direction"`
	Status    string `gorm:"size:32;not null" json:"status"` // success/decode_fail/invalid_field/business_fail
	HexData   string `gorm:"type:text" json:"hexData"`       // 原始16进制
	JSONData  string `gorm:"type:text" json:"jsonData"`      // 解码后JSON
	ErrorMsg  string `gorm:"size:512" json:"errorMsg,omitempty"`
	Timestamp time.Time `gorm:"index" json:"timestamp"`
}

// TableName 表名
func (MessageArchive) TableName() string {
	return "message_archives"
}
