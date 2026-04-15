// Package recorder provides per-session message recording and statistics.
package recorder

import (
	"fmt"
	"sync"
	"time"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
)

// MessageStatus 消息处理状态
type MessageStatus string

const (
	StatusSuccess      MessageStatus = "success"       // 解码成功且字段合法
	StatusDecodeFail   MessageStatus = "decode_fail"   // 解码失败
	StatusInvalidField MessageStatus = "invalid_field"  // 字段值非法
	StatusMessageLoss  MessageStatus = "message_loss"   // 消息丢失（期望但未收到）
)

// MessageRecord 单条消息记录
type MessageRecord struct {
	Timestamp time.Time     // 收发时间
	FuncCode  byte          // 功能码
	Direction types.Direction // 方向
	Status    MessageStatus // 处理状态
	HexData   string        // 原始16进制报文
	JSONData  string        // 解码后JSON
	ErrorMsg  string        // 错误信息（如有）
}

// FuncCodeStat 功能码统计
type FuncCodeStat struct {
	FuncCode       byte  // 功能码
	Direction      types.Direction // 方向
	TotalMessages  int   // 收到消息总数
	SuccessCount   int   // 成功数
	DecodeFail     int   // 解码失败数
	InvalidField   int   // 字段值非法数
	MessageLoss    int   // 消息丢失数
}

// SuccessRate 成功率
func (s *FuncCodeStat) SuccessRate() float64 {
	if s.TotalMessages == 0 {
		return 0
	}
	return float64(s.SuccessCount) / float64(s.TotalMessages) * 100
}

// SessionRecorder 会话消息记录器
// 在会话期间实时记录每个功能码的上下行消息，并按功能码统计
type SessionRecorder struct {
	sessionID  string
	postNo     uint32
	startTime  time.Time
	endTime    time.Time
	mu         sync.RWMutex
	records    []MessageRecord            // 所有消息记录
	stats      map[statKey]*FuncCodeStat  // 按功能码+方向统计
}

type statKey struct {
	FuncCode  byte
	Direction types.Direction
}

// NewSessionRecorder 创建会话记录器
func NewSessionRecorder(sessionID string, postNo uint32) *SessionRecorder {
	return &SessionRecorder{
		sessionID: sessionID,
		postNo:    postNo,
		startTime: time.Now(),
		records:   make([]MessageRecord, 0),
		stats:     make(map[statKey]*FuncCodeStat),
	}
}

// RecordRecv 记录收到的消息（充电桩→平台）
func (r *SessionRecorder) RecordRecv(funcCode byte, status MessageStatus, hexData string, jsonData string, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec := MessageRecord{
		Timestamp: time.Now(),
		FuncCode:  funcCode,
		Direction: types.DirectionUpload,
		Status:    status,
		HexData:   hexData,
		JSONData:  jsonData,
		ErrorMsg:  errMsg,
	}
	r.records = append(r.records, rec)
	r.updateStat(funcCode, types.DirectionUpload, status)
}

// RecordSend 记录发送的消息（平台→充电桩）
func (r *SessionRecorder) RecordSend(funcCode byte, status MessageStatus, hexData string, jsonData string, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec := MessageRecord{
		Timestamp: time.Now(),
		FuncCode:  funcCode,
		Direction: types.DirectionDownload,
		Status:    status,
		HexData:   hexData,
		JSONData:  jsonData,
		ErrorMsg:  errMsg,
	}
	r.records = append(r.records, rec)
	r.updateStat(funcCode, types.DirectionDownload, status)
}

// RecordReply 记录回复消息
func (r *SessionRecorder) RecordReply(funcCode byte, status MessageStatus, hexData string, jsonData string, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec := MessageRecord{
		Timestamp: time.Now(),
		FuncCode:  funcCode,
		Direction: types.DirectionReply,
		Status:    status,
		HexData:   hexData,
		JSONData:  jsonData,
		ErrorMsg:  errMsg,
	}
	r.records = append(r.records, rec)
	r.updateStat(funcCode, types.DirectionReply, status)
}

// updateStat 更新功能码统计
func (r *SessionRecorder) updateStat(funcCode byte, dir types.Direction, status MessageStatus) {
	key := statKey{FuncCode: funcCode, Direction: dir}
	stat, ok := r.stats[key]
	if !ok {
		stat = &FuncCodeStat{
			FuncCode:  funcCode,
			Direction: dir,
		}
		r.stats[key] = stat
	}

	stat.TotalMessages++

	switch status {
	case StatusSuccess:
		stat.SuccessCount++
	case StatusDecodeFail:
		stat.DecodeFail++
	case StatusInvalidField:
		stat.InvalidField++
	case StatusMessageLoss:
		stat.MessageLoss++
	}
}

// Close 关闭记录器（会话结束时调用）
func (r *SessionRecorder) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.endTime = time.Now()
}

// GetStats 获取所有功能码统计
func (r *SessionRecorder) GetStats() []FuncCodeStat {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]FuncCodeStat, 0, len(r.stats))
	for _, stat := range r.stats {
		result = append(result, *stat)
	}
	return result
}

// GetRecordsByFuncCode 获取指定功能码的消息记录
func (r *SessionRecorder) GetRecordsByFuncCode(funcCode byte, status MessageStatus) []MessageRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []MessageRecord
	for _, rec := range r.records {
		if rec.FuncCode == funcCode && (status == "" || rec.Status == status) {
			result = append(result, rec)
		}
	}
	return result
}

// GetAllRecords 获取所有消息记录
func (r *SessionRecorder) GetAllRecords() []MessageRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]MessageRecord, len(r.records))
	copy(result, r.records)
	return result
}

// SessionID 返回会话ID
func (r *SessionRecorder) SessionID() string {
	return r.sessionID
}

// PostNo 返回充电桩编号
func (r *SessionRecorder) PostNo() uint32 {
	return r.postNo
}

// StartTime 返回会话开始时间
func (r *SessionRecorder) StartTime() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.startTime
}

// EndTime 返回会话结束时间
func (r *SessionRecorder) EndTime() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.endTime
}

// Duration 返回会话持续时间
func (r *SessionRecorder) Duration() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	end := r.endTime
	if end.IsZero() {
		end = time.Now()
	}
	return end.Sub(r.startTime)
}

// TotalMessages 返回总消息数
func (r *SessionRecorder) TotalMessages() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.records)
}

// SuccessTotal 返回成功总数
func (r *SessionRecorder) SuccessTotal() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := 0
	for _, stat := range r.stats {
		total += stat.SuccessCount
	}
	return total
}

// FailTotal 返回失败总数
func (r *SessionRecorder) FailTotal() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := 0
	for _, stat := range r.stats {
		total += stat.DecodeFail + stat.InvalidField + stat.MessageLoss
	}
	return total
}

// OverallSuccessRate 返回总体成功率
func (r *SessionRecorder) OverallSuccessRate() float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := 0
	success := 0
	for _, stat := range r.stats {
		total += stat.TotalMessages
		success += stat.SuccessCount
	}
	if total == 0 {
		return 0
	}
	return float64(success) / float64(total) * 100
}

// IsTestPass 判断测试是否通过（所有消息成功率100%）
func (r *SessionRecorder) IsTestPass() bool {
	return r.OverallSuccessRate() == 100.0
}

// Summary 返回会话摘要（用于报告生成）
func (r *SessionRecorder) Summary() *SessionSummary {
	r.mu.RLock()
	defer r.mu.RUnlock()

	end := r.endTime
	if end.IsZero() {
		end = time.Now()
	}

	stats := make([]FuncCodeStat, 0, len(r.stats))
	for _, stat := range r.stats {
		stats = append(stats, *stat)
	}

	return &SessionSummary{
		SessionID:       r.sessionID,
		PostNo:          r.postNo,
		StartTime:       r.startTime,
		EndTime:         end,
		Duration:        end.Sub(r.startTime),
		TotalMessages:   len(r.records),
		SuccessTotal:    r.SuccessTotal(),
		FailTotal:       r.FailTotal(),
		SuccessRate:     r.OverallSuccessRate(),
		IsPass:          r.IsTestPass(),
		FuncCodeStats:   stats,
	}
}

// SessionSummary 会话摘要
type SessionSummary struct {
	SessionID     string
	PostNo        uint32
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	TotalMessages int
	SuccessTotal  int
	FailTotal     int
	SuccessRate   float64
	IsPass        bool
	FuncCodeStats []FuncCodeStat
}

// FormatFuncCode 格式化功能码为0xXX
func FormatFuncCode(code byte) string {
	return fmt.Sprintf("0x%02X", code)
}

// FormatDirection 格式化方向
func FormatDirection(dir types.Direction) string {
	switch dir {
	case types.DirectionUpload:
		return "充电桩→平台"
	case types.DirectionDownload:
		return "平台→充电桩"
	case types.DirectionReply:
		return "回复"
	default:
		return "未知"
	}
}
