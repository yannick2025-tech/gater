package model

import "time"

// Session 充电桩TCP连接会话（持久化到数据库，解决重启丢数据问题）
type Session struct {
	ID           string     `gorm:"primaryKey;size:64" json:"id"`
	PostNo       uint32     `gorm:"index;not null" json:"postNo"`
	ConnID       string     `gorm:"size:64" json:"connId"`
	AuthState    string     `gorm:"size:32;default:none" json:"authState"` // none/pending/authenticated
	IsOnline     bool       `gorm:"default:true;index" json:"isOnline"`
	ProtocolName string     `gorm:"size:64" json:"protocolName"`
	ProtocolVer  string     `gorm:"size:16" json:"protocolVersion"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
	ClosedAt     *time.Time `json:"closedAt,omitempty"`
}

// TableName 表名
func (Session) TableName() string {
	return "sessions"
}
