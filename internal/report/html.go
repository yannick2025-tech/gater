// Package report provides HTML test report generation.
package report

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/yannick2025-tech/nts-gater/internal/database"
	"github.com/yannick2025-tech/nts-gater/internal/model"
)

// HTMLReportData HTML 报告数据
type HTMLReportData struct {
	Session     model.Session
	Reports     []ScenarioReport
	GeneratedAt string
}

// ScenarioReport 场景级报告数据
type ScenarioReport struct {
	Report     model.TestReport
	TestCases  []CaseReport
}

// CaseReport 用例级报告数据
type CaseReport struct {
	TestCase     model.TestCase
	Messages     []model.MessageArchive
	Validations  []model.ValidationResult
}

// toInt64 将模板中可能的 int/int64 统一转为 int64
func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int64:
		return val
	case int32:
		return int64(val)
	default:
		return 0
	}
}

// GenerateHTML 生成 HTML 测试报告并打包为 ZIP
// ZIP 内包含带完整样式的 HTML 文件
// 返回 ZIP 文件路径
func GenerateHTML(sessionID string) (string, error) {
	db := database.GetDB()
	if db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	// 1. 查询 session（获取协议名称和版本号）
	var sess model.Session
	if err := db.Where("id = ?", sessionID).First(&sess).Error; err != nil {
		return "", fmt.Errorf("session not found: %w", err)
	}

	// 1.5 ClosedAt 兜底：如果异步 onRemove 回调尚未写入，取最后报文时间
	if sess.ClosedAt == nil || sess.ClosedAt.IsZero() {
		var lastMsg model.MessageArchive
		db.Where("session_id = ?", sessionID).Order("timestamp DESC").Limit(1).First(&lastMsg)
		if !lastMsg.Timestamp.IsZero() {
			sess.ClosedAt = &lastMsg.Timestamp
		}
	}

	// 2. 查询 test_reports
	var reports []model.TestReport
	db.Where("session_id = ?", sessionID).Order("start_time ASC").Find(&reports)

	// 3. 构建场景级数据
	scenarioReports := make([]ScenarioReport, 0, len(reports))
	for _, r := range reports {
		sr := ScenarioReport{Report: r}

		// 3.1 查询用例
		var cases []model.TestCase
		db.Where("session_id = ? AND scenario_id = ?", sessionID, r.ScenarioID).
			Order("created_at ASC").Find(&cases)

		caseReports := make([]CaseReport, 0, len(cases))
		for _, tc := range cases {
			cr := CaseReport{TestCase: tc}

			// 3.2 查询报文存档
			db.Where("session_id = ? AND case_id = ?", sessionID, tc.CaseID).
				Order("timestamp ASC").Find(&cr.Messages)

			// 3.3 查询校验结果
			db.Where("session_id = ? AND case_id = ?", sessionID, tc.CaseID).
				Order("created_at ASC").Find(&cr.Validations)

			caseReports = append(caseReports, cr)
		}

		// 无用例时也保留场景信息（展示未关联到具体用例的报文 → 报文检测）
		if len(cases) == 0 {
			var msgs []model.MessageArchive
			db.Where("session_id = ? AND (case_id = '' OR case_id IS NULL)", sessionID).
				Order("timestamp ASC").Find(&msgs)
			if len(msgs) > 0 {
				// 按状态类型分别统计（success / decode_fail / invalid_field / business_fail）
				totalMsgs := len(msgs)
				successCnt := 0
				decodeFailCnt := 0
				invalidFieldCnt := 0
				businessFailCnt := 0
				for _, m := range msgs {
					switch m.Status {
					case "success":
						successCnt++
					case "decode_fail":
						decodeFailCnt++
					case "invalid_field":
						invalidFieldCnt++
					case "business_fail":
						businessFailCnt++
					default:
						// 未知状态归入其他失败
						decodeFailCnt++
					}
				}
				successRate := 0.0
				if totalMsgs > 0 {
					successRate = float64(successCnt) / float64(totalMsgs) * 100
				}
				result := "pass"
				if decodeFailCnt+invalidFieldCnt+businessFailCnt > 0 {
					result = "fail"
				}

				caseReports = append(caseReports, CaseReport{
					TestCase: model.TestCase{
						CaseID:        "",
						CaseName:      "报文检测",
						Status:        "completed",
						Result:        result,
						TotalMessages: totalMsgs,
						SuccessCount:  successCnt,
						DecodeFail:    decodeFailCnt,
						InvalidField:  invalidFieldCnt,
						BusinessFail:  businessFailCnt,
						SuccessRate:   successRate,
					},
					Messages: msgs,
				})
			}
		}

		sr.TestCases = caseReports
		scenarioReports = append(scenarioReports, sr)
	}

	// 4. 渲染 HTML 到内存
	data := HTMLReportData{
		Session:     sess,
		Reports:     scenarioReports,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"fmtDate":    func(t time.Time) string { return t.Format("2006-01-02 15:04:05") },
		"fmtDateMs":  func(t time.Time) string { return t.Format("2006-01-02 15:04:05.000") },
		"fmtDatePtr": func(t *time.Time) string { if t == nil || t.IsZero() { return "--" }; return t.Format("2006-01-02 15:04:05") },
		"fmtRate":    func(r float64) string { return fmt.Sprintf("%.2f%%", r) },
		"fmtDur":     func(ms int64) string { return formatDuration(ms) },
		"add":        func(a, b interface{}) int64 { return toInt64(a) + toInt64(b) },
		"i64":        func(n interface{}) int64  { return toInt64(n) },
		"lenSlice":   func(slice interface{}) int64 {
			v := reflect.ValueOf(slice)
			if v.Kind() == reflect.Slice { return int64(v.Len()) }
			return 0
		},
		"mul":        func(a, b interface{}) int64 { return toInt64(a) * toInt64(b) },
		"div":        func(a, b interface{}) float64 {
			bv := toInt64(b)
			if bv == 0 { return 0 }
			return float64(toInt64(a)) / float64(bv)
		},
		"upper":      strings.ToUpper,
		"statusIcon": func(passed bool) string { if passed { return "✅" }; return "❌" },
		"resultClass": func(r string) string {
			switch r {
			case "pass":
				return "pass"
			case "fail":
				return "fail"
			default:
				return "skip"
			}
		},
	}).Parse(reportHTMLTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	// 将渲染结果写入内存 buffer
	var htmlBuf bytes.Buffer
	if err := tmpl.Execute(&htmlBuf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	// 5. 构建 ZIP 文件名: 协议名称-版本号-接入测试报告-yyyymmdd-hhmiss.zip
	now := time.Now()
	protocolName := strings.TrimSpace(sess.ProtocolName)
	if protocolName == "" {
		protocolName = "XX标准协议"
	}
	protocolVer := strings.TrimSpace(sess.ProtocolVer)
	if protocolVer == "" {
		protocolVer = "v1.6.0"
	}
	zipFileName := fmt.Sprintf("%s-%s-接入测试报告-%s-%s.zip",
		protocolName,
		protocolVer,
		now.Format("20060102"),
		now.Format("150405"),
	)

	outputDir := "reports"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create reports dir: %w", err)
	}
	zipPath := filepath.Join(outputDir, zipFileName)

	// 6. 创建 ZIP 文件，内含 report.html
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 在 ZIP 中创建 report.html
	htmlWriter, err := zipWriter.Create("report.html")
	if err != nil {
		return "", fmt.Errorf("create zip entry: %w", err)
	}

	if _, err := htmlWriter.Write(htmlBuf.Bytes()); err != nil {
		return "", fmt.Errorf("write html to zip: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		return "", fmt.Errorf("close zip writer: %w", err)
	}

	return zipPath, nil
}

// reportHTMLTemplate HTML报告模板 — 浅色主题（匹配gater管理平台配色）
const reportHTMLTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>测试报告 - {{.Session.ID}}</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&family=Noto+Sans+SC:wght@300;400;500;700;900&display=swap" rel="stylesheet">
<style>
:root{
  /* === gater管理平台同源配色 === */
  --bg-primary:#f5f6fa;--bg-secondary:#fff;--bg-card:#fff;
  --bg-card-hover:#fafbfc;--border:#e4e7ed;--border-light:#ebeef5;
  --border-accent:rgba(20,132,147,0.2);--text-primary:#303133;--text-secondary:#606266;
  --text-muted:#909399;--text-placeholder:#c0c4cc;--accent:#148493;--accent-hover:#0e6e7b;
  --accent-dim:rgba(20,132,147,0.06);--accent-border:rgba(20,132,147,0.15);
  --success:#52c41a;--success-dim:rgba(82,196,26,0.06);--success-border:rgba(82,196,26,0.15);
  --error:#ff4d4f;--error-dim:rgba(255,77,79,0.06);--error-border:rgba(255,77,79,0.15);
  --warning:#ea933f;--warning-dim:rgba(234,147,63,0.06);--warning-border:rgba(234,147,63,0.15);
  --info:#409eff;--info-dim:rgba(64,158,255,0.06);--info-border:rgba(64,158,255,0.15);
  --dir-up:#67c23a;--dir-down:#409eff;--dir-reply:#909399;
  --code-bg:#f0f2f5;--code-dark-bg:#1e1e1e;--code-text:#d4d4d4;
  --radius-sm:6px;--radius-md:8px;--radius-lg:12px;
  --shadow-sm:0 1px 3px rgba(0,0,0,0.06);--shadow:0 2px 8px rgba(0,0,0,0.08);
  --shadow-md:0 4px 16px rgba(0,0,0,0.1);--shadow-lg:0 8px 30px rgba(0,0,0,0.12);
}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'Noto Sans SC',-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;
  color:var(--text-primary);background:var(--bg-primary);line-height:1.65;min-height:100vh}

.container{max-width:1280px;margin:0 auto;padding:28px 32px 48px}

/* ═════════ 封面区域 ═════════ */
.cover{
  position:relative;border-radius:var(--radius-lg);padding:36px 40px;margin-bottom:28px;overflow:hidden;
  background:linear-gradient(135deg,#148493 0%,#0e6e7b 50%,#0a5c68 100%);
  color:#fff;box-shadow:var(--shadow-md)
}
.cover::before{
  content:'';position:absolute;top:-80px;right:-60px;width:280px;height:280px;
  background:radial-gradient(circle,rgba(255,255,255,0.08),transparent 70%);pointer-events:none}
.cover::after{
  content:'';position:absolute;bottom:-40px;left:-30px;width:160px;height:160px;
  background:radial-gradient(circle,rgba(255,255,255,0.05),transparent 70%);pointer-events:none}
.cover h1{
  font-size:28px;font-weight:700;letter-spacing:-.3px;display:flex;align-items:center;gap:14px;
  position:relative;z-index:1
}
.cover h1 .icon-mark{
  display:inline-flex;width:42px;height:42px;background:rgba(255,255,255,0.15);
  border-radius:10px;align-items:center;justify-content:center;font-size:20px;
  border:1px solid rgba(255,255,255,0.2);flex-shrink:0;backdrop-filter:blur(4px)
}
.cover .meta-row{
  display:flex;flex-wrap:wrap;gap:8px 20px;margin-top:18px;position:relative;z-index:1}
.cover .meta-item{
  font-size:13px;color:rgba(255,255,255,0.85);display:flex;align-items:center;gap:7px;
  padding:5px 13px;background:rgba(255,255,255,0.08);border-radius:20px;border:1px solid rgba(255,255,255,0.1);
  backdrop-filter:blur(4px)
}
.cover .meta-item strong{color:#fff}
.cover .meta-item .dot{
  width:8px;height:8px;border-radius:50%;background:#5eecc4;box-shadow:0 0 8px rgba(94,236,196,0.5);flex-shrink:0}
.cover .meta-item.warn-dot .dot{background:#ffc96b;box-shadow:0 0 8px rgba(255,201,107,0.5)}

/* ═════════ 统计卡片区 ═════════ */
.stats-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:16px;margin-bottom:28px}
.stat-card{
  background:var(--bg-card);border:1px solid var(--border);border-radius:var(--radius-md);
  padding:22px 20px;position:relative;overflow:hidden;transition:all .3s ease;
  box-shadow:var(--shadow-sm)
}
.stat-card:hover{border-color:var(--border-accent);transform:translateY(-2px);box-shadow:var(--shadow-md)}
.stat-card::before{
  content:'';position:absolute;top:0;left:0;right:0;height:3px;
  background:linear-gradient(90deg,transparent,var(--accent),transparent);opacity:0;transition:opacity .3s}
.stat-card:hover::before{opacity:1}
.stat-card.danger::before{background:linear-gradient(90deg,transparent,var(--error),transparent);opacity:1}
.stat-card.warning::before{background:linear-gradient(90deg,transparent,var(--warning),transparent);opacity:1}
.stat-card .stat-label{font-size:12px;color:var(--text-muted);font-weight:500;margin-bottom:10px;
  text-transform:uppercase;letter-spacing:.8px}
.stat-card .stat-value{font-family:'JetBrains Mono',monospace;font-size:32px;font-weight:700;line-height:1.1;
  color:var(--text-primary);min-height:38px;display:flex;align-items:center}
.stat-card .stat-value.val-pass{color:var(--success)}
.stat-card .stat-value.val-fail{color:var(--error)}
.stat-card .stat-value.val-warn{color:var(--warning)}
.stat-card .stat-sub{font-size:12px;color:var(--text-muted);margin-top:6px}

/* 进度环容器 */
.progress-ring-wrap{position:absolute;right:16px;top:50%;transform:translateY(-50%);width:52px;height:52px;opacity:.1}
.progress-ring{transform:rotate(-90deg)}

/* ═════════ 锚点导航 ═════════ */
.nav-bar{
  position:sticky;top:0;z-index:100;padding:14px 0;margin-bottom:24px;
  background:rgba(255,255,255,0.92);backdrop-filter:blur(12px);
  border-bottom:1px solid var(--border);border-top:1px solid var(--border);
  border-radius:0 0 var(--radius-md) var(--radius-md);box-shadow:var(--shadow-sm)
}
.nav-bar .nav-inner{display:flex;gap:8px;flex-wrap:wrap;align-items:center}
.nav-bar a{
  display:inline-flex;align-items:center;gap:6px;padding:7px 16px;border-radius:20px;
  font-size:13px;color:var(--accent);text-decoration:none;font-weight:500;
  background:var(--accent-dim);border:1px solid var(--accent-border);
  transition:all .25s ease
}
.nav-bar a:hover{background:var(--accent);color:#fff;box-shadow:0 4px 14px rgba(20,132,147,0.25)}

/* ═════════ 场景区块 ═════════ */
.scenario{
  background:var(--bg-card);border:1px solid var(--border);border-radius:var(--radius-lg);
  margin-bottom:24px;overflow:hidden;transition:border-color .3s;box-shadow:var(--shadow-sm);
  animation:fadeSlideUp .5s ease both
}
.scenario:hover{border-color:var(--border-accent)}
.scenario-header{
  padding:18px 24px;display:flex;align-items:center;justify-content:space-between;
  border-bottom:1px solid var(--border);background:#fafbfc;
  flex-wrap:wrap;gap:12px
}
.scenario-header h2{font-size:18px;font-weight:600;display:flex;align-items:center;gap:12px;color:var(--text-primary)}
.scenario-header .badge-status{
  display:inline-flex;align-items:center;gap:5px;padding:5px 13px;border-radius:20px;
  font-size:12px;font-weight:600;letter-spacing:.3px
}
.badge-status.pass{background:var(--success-dim);color:var(--success);border:1px solid var(--success-border)}
.badge-status.fail{background:var(--error-dim);color:var(--error);border:1px solid var(--error-border)}
.scenario-id-tag{font-size:11.5px;color:var(--text-muted);font-family:'JetBrains Mono',monospace;
  background:var(--code-bg);padding:3px 10px;border-radius:4px;border:1px solid var(--border-light)}
.scenario-body{padding:22px 24px}

/* ═════════ 用例区块 ═════════ */
.case{
  border:1px solid var(--border);border-radius:var(--radius-md);margin-bottom:14px;
  overflow:hidden;transition:all .25s ease;animation:fadeSlideUp .45s ease both;
  box-shadow:var(--shadow-sm)
}
.case:hover{border-color:var(--border-accent);box-shadow:var(--shadow)}
.case-header{
  padding:14px 18px;background:#fafbfc;border-bottom:1px solid var(--border);
  display:flex;justify-content:space-between;align-items:center;cursor:pointer;
  transition:background .2s;flex-wrap:wrap;gap:10px
}
.case-header:hover{background:#f0f2f5}
.case-header h3{font-size:15px;font-weight:600;display:flex;align-items:center;gap:10px;color:var(--text-primary)}
.status-indicator{width:10px;height:10px;border-radius:50%;flex-shrink:0}
.status-indicator.pass{background:var(--success);box-shadow:0 0 8px rgba(82,196,26,0.35)}
.status-indicator.fail{background:var(--error);box-shadow:0 0 8px rgba(255,77,79,0.35)}
.status-indicator.running{background:var(--warning);box-shadow:0 0 8px rgba(234,147,63,0.35);
  animation:pulse 1.8s infinite}
.case-badge{
  font-size:12px;font-weight:600;padding:4px 12px;border-radius:14px;font-family:'JetBrains Mono',monospace;
  letter-spacing:.3px
}
.case-badge.pass{background:var(--success-dim);color:var(--success);border:1px solid var(--success-border)}
.case-badge.fail{background:var(--error-dim);color:var(--error);border:1px solid var(--error-border)}
.case-badge.skip{background:var(--code-bg);color:var(--text-muted);border:1px solid var(--border-light)}
.case-stats{font-size:12.5px;color:var(--text-secondary);font-family:'JetBrains Mono',monospace}
.case-body{padding:18px;display:none;border-top:1px solid var(--border);background:#fafbfc}
.case-body.expanded{display:block;animation:expandIn .3s ease}

@keyframes pulse{0%,100%{opacity:1}50%{opacity:.4}}
@keyframes fadeSlideUp{from{opacity:0;transform:translateY(16px)}to{opacity:1;transform:translateY(0)}}
@keyframes expandIn{from{opacity:0;transform:translateY(-8px)}to{opacity:1;transform:translateY(0)}}

/* ═════════ 数据表格 ═════════ */
.data-table{width:100%;border-collapse:separate;border-spacing:0;font-size:13px;
  margin:14px 0;border:1px solid var(--border);border-radius:var(--radius-sm);overflow:hidden}
.data-table thead th{
  padding:11px 16px;text-align:left;font-weight:600;font-size:12px;
  text-transform:uppercase;letter-spacing:.6px;color:var(--text-muted);
  background:#fafafa;border-bottom:1px solid var(--border)
}
.data-table tbody td{padding:11px 16px;border-bottom:1px solid #f5f5f5;color:var(--text-secondary)}
.data-table tbody tr:last-child td{border-bottom:none}
.data-table tbody tr:hover td{background:var(--accent-dim);color:var(--text-primary)}
.data-table .num-cell{font-family:'JetBrains Mono',monospace;font-weight:600;color:var(--text-primary)}

/* ═════════ 校验结果 ═════════ */
.validation{
  margin:8px 0;padding:11px 16px;border-radius:var(--radius-sm);font-size:13px;
  display:flex;align-items:flex-start;gap:10px;line-height:1.5;border:1px solid transparent;
  animation:fadeSlideUp .35s ease both
}
.validation.pass{background:var(--success-dim);color:#3da119;border-color:var(--success-border)}
.validation.fail{background:var(--error-dim);color:#d73435;border-color:var(--error-border)}
.validation .v-icon{flex-shrink:0;font-size:15px;margin-top:1px}
.validation small{color:var(--text-secondary);display:block;margin-top:3px;font-size:12px}

/* ═════════ 报文明细 ═════════ */
.msg-detail{margin:10px 0;border:1px solid var(--border);border-radius:var(--radius-md);overflow:hidden;
  transition:border-color .2s;box-shadow:var(--shadow-sm)}
.msg-detail:hover{border-color:var(--border-accent)}
.msg-hdr{
  padding:10px 16px;cursor:pointer;display:flex;justify-content:space-between;align-items:center;
  gap:12px;transition:background .2s;flex-wrap:wrap;
  background:#fafbfc;border-bottom:1px solid var(--border)
}
.msg-hdr:hover{background:#f0f2f5}
.msg-hdr-left{display:flex;align-items:center;gap:10px;min-width:0}
.msg-dir-tag{
  display:inline-block;padding:2px 9px;border-radius:4px;font-size:11px;font-weight:600;
  letter-spacing:.3px;font-family:'JetBrains Mono',monospace;flex-shrink:0
}
.msg-dir-tag.recv{background:rgba(103,194,58,0.08);color:var(--dir-up);border:1px solid rgba(103,194,58,0.18)}
.msg-dir-tag.send{background:var(--accent-dim);color:var(--accent);border:1px solid var(--accent-border)}
.msg-dir-tag.reply{background:rgba(144,147,153,0.08);color:var(--dir-reply);border:1px solid rgba(144,147,153,0.15)}
.msg-func-code{font-family:'JetBrains Mono',monospace;font-size:13px;font-weight:600;color:var(--warning)}
.msg-ts{font-family:'JetBrains Mono',monospace;font-size:11.5px;color:var(--text-muted);white-space:nowrap}
.msg-body{padding:16px;display:none;background:#f8f9fb;border-top:1px solid var(--border)}
.msg-body.expanded{display:block;animation:expandIn .25s ease}
.msg-meta-row{font-size:12px;color:var(--text-muted);margin-bottom:12px;display:flex;flex-wrap:wrap;gap:12px}
.msg-meta-row span{display:inline-flex;align-items:center;gap:4px}
.msg-error{color:var(--error);font-size:13px;margin-bottom:10px;padding:8px 12px;
  background:var(--error-dim);border-radius:var(--radius-sm);border:1px solid var(--error-border)}
.msg-section-title{font-size:11.5px;font-weight:600;text-transform:uppercase;letter-spacing:.8px;
  color:var(--text-muted);margin:10px 0 6px;display:flex;align-items:center;gap:8px}
.msg-section-title::after{content:'';flex:1;height:1px;background:var(--border)}

/* 代码块 — 与平台报文区保持一致的暗色风格 */
.code-block{
  background:var(--code-dark-bg);border:1px solid #383838;border-radius:var(--radius-sm);
  padding:14px 16px;overflow-x:auto;font-family:'JetBrains Mono',monospace;
  font-size:12px;line-height:1.7;white-space:pre-wrap;word-break:break-all;
  color:var(--code-text);max-height:380px;overflow-y:auto
}
.code-block::-webkit-scrollbar{width:6px;height:6px}
.code-block::-webkit-scrollbar-track{background:transparent}
.code-block::-webkit-scrollbar-thumb{background:rgba(136,136,136,.35);border-radius:3px}

/* ═════════ 报文明细标题区 ═════════ */
.detail-section{margin-bottom:24px;animation:fadeSlideUp .5s ease both}
.detail-section-title{
  font-size:16px;font-weight:600;margin:18px 0 12px;padding:12px 18px;
  display:flex;align-items:center;gap:10px;color:var(--text-primary);
  background:var(--bg-card);border-radius:var(--radius-md);
  border:1px solid var(--border);border-left:4px solid var(--accent);
  box-shadow:var(--shadow-sm)
}
.detail-section-title .ds-icon{font-size:18px}

/* 错误摘要 */
.error-summary{
  margin:10px 0;padding:12px 16px;background:var(--error-dim);border-radius:var(--radius-sm);
  border:1px solid var(--error-border);font-size:13px;color:var(--error);
  display:flex;align-items:flex-start;gap:8px;line-height:1.5
}
.error-summary .es-icon{flex-shrink:0}

/* ═════════ 页脚 ═════════ */
.page-footer{
  text-align:center;padding:24px;color:var(--text-muted);font-size:12px;
  margin-top:32px;border-top:1px solid var(--border);letter-spacing:.3px
}
.page-footer span{opacity:.7}

/* ═════════ 空状态 ═════════ */
.empty-state{text-align:center;padding:48px 20px;color:var(--text-muted)}
.empty-state .es-icon{font-size:40px;margin-bottom:12px;opacity:.35}

/* ═════════ 锚点偏移 ═════════ */
[id]{scroll-margin-top:76px}

/* 查看报文明细链接 */
.detail-link{
  display:inline-flex;align-items:center;gap:6px;margin-top:14px;padding:8px 16px;
  font-size:13px;color:var(--accent);text-decoration:none;font-weight:500;
  background:var(--accent-dim);border:1px solid var(--accent-border);border-radius:20px;
  transition:all .25s ease;cursor:pointer
}
.detail-link:hover{background:var(--accent);color:#fff;box-shadow:0 4px 14px rgba(20,132,147,0.25)}

/* ═════════ 打印样式 ═════════ */
@media print{
  body{background:#fff!important;color:#222!important}
  .nav-bar{display:none!important;box-shadow:none!important}
  .container{padding:0!important;max-width:100%!important}
  .cover{background:linear-gradient(135deg,#148493 0%,#0e6e7b 100%)!important;color:#fff!important;border-radius:0;box-shadow:none!important}
  .scenario,.case,.msg-detail{border-color:#ddd!important;break-inside:avoid;box-shadow:none!important}
  .case-body,.msg-body{display:block!important}
  .stat-card{border-color:#ccc!important;page-break-inside:avoid;box-shadow:none!important}
  .code-block{background:#f5f5f5!important;color:#333!important;border-color:#ddd!important}
  .validation.pass{background:#f6ffed!important;color:#389e0d!important;border-color:#b7eb8f!important}
  .validation.fail{background:#fff2f0!important;color:#cf1322!important;border-color:#ffa39e!important}
  .detail-link{display:none!important}
}

/* 响应式 */
@media(max-width:768px){
  .container{padding:16px}
  .cover{padding:24px 20px}
  .cover h1{font-size:22px}
  .stats-grid{grid-template-columns:repeat(2,1fr)}
  .scenario-body,.case-body{padding:16px}
  .msg-meta-row{flex-direction:column;gap:4px}
}
</style>
</head>
<body>
<div class="container">

<!-- ══ 封面 ══ -->
<div class="cover">
  <h1><span class="icon-mark">⚡</span> 充电桩协议接入测试报告</h1>
  <div class="meta-row">
    <div class="meta-item"><span class="dot"></span>桩号: <strong>{{.Session.PostNo}}</strong></div>
    <div class="meta-item"><span class="dot"></span>Session: <strong style="font-family:'JetBrains Mono',monospace">{{.Session.ID}}</strong></div>
    <div class="meta-item">协议: <strong>{{.Session.ProtocolName}} {{.Session.ProtocolVer}}</strong></div>
    <div class="meta-item">连接: <strong>{{fmtDate .Session.CreatedAt}}</strong></div>
    <div class="meta-item warn-dot{{if .Session.ClosedAt}}{{else}} warn-dot{{end}}">
      <span class="dot"></span>断开: <strong>{{fmtDatePtr .Session.ClosedAt}}</strong>
    </div>
  </div>
</div>

<!-- ══ 全局统计概览 ══ -->
{{$totalScenarios := lenSlice .Reports}}
{{$totalCases := i64 0}}{{$totalPass := i64 0}}{{$totalFail := i64 0}}{{$totalMsgs := i64 0}}
{{range $_,$sr := .Reports}}
  {{$tcLen := lenSlice $sr.TestCases}}
  {{$totalCases = add $totalCases $tcLen}}
  {{$tPass := i64 0}}{{$tFail := i64 0}}
  {{range $_,$cr := $sr.TestCases}}
    {{if eq $cr.TestCase.Result "pass"}}
      {{$tPass = add $tPass (i64 1)}}
    {{else if eq $cr.TestCase.Result "fail"}}
      {{$tFail = add $tFail (i64 1)}}
    {{end}}
    {{$totalMsgs = add $totalMsgs $cr.TestCase.TotalMessages}}
  {{end}}
  {{$totalPass = add $totalPass $tPass}}
  {{$totalFail = add $totalFail $tFail}}
{{end}}

<div class="stats-grid">
  <div class="stat-card">
    <div class="stat-label">场景数</div>
    <div class="stat-value">{{$totalScenarios}}</div>
  </div>
  <div class="stat-card">
    <div class="stat-label">总用例</div>
    <div class="stat-value">{{$totalCases}}</div>
    <div class="stat-sub">{{if gt $totalCases (i64 0)}}{{printf "%.2f" (div (mul $totalPass (i64 100)) $totalCases)}}% 通过率{{else}}--{{end}}</div>
  </div>
  <div class="stat-card">
    <div class="stat-label">通过</div>
    <div class="stat-value val-pass">{{$totalPass}}</div>
  </div>
  <div class="stat-card {{if gt $totalFail (i64 0)}}danger{{end}}">
    <div class="stat-label">失败</div>
    <div class="stat-value {{if gt $totalFail (i64 0)}}val-fail{{else}}val-warn{{end}}">{{$totalFail}}</div>
  </div>
  <div class="stat-card">
    <div class="stat-label">报文总量</div>
    <div class="stat-value">{{$totalMsgs}}</div>
  </div>
</div>

<!-- ══ 锚点导航 ══ -->
{{if gt $totalScenarios (i64 1)}}
<div class="nav-bar">
  <div class="nav-inner">
  {{$navIdx := i64 0}}{{range $_,$sr := .Reports}}
    <a href="#sc-{{$sr.Report.ScenarioID}}">#{{$navIdx}} {{$sr.Report.ScenarioName}}</a>
    {{$navIdx = add $navIdx (i64 1)}}
  {{end}}
  </div>
</div>
{{end}}

<!-- ══ 场景列表 ══ -->
{{if not .Reports}}
<div class="empty-state">
  <div class="es-icon">📭</div>
  <p>无测试场景执行记录</p>
</div>
{{end}}

{{$sci := i64 0}}{{range $_,$sr := .Reports}}
<div class="scenario" id="sc-{{$sr.Report.ScenarioID}}" style="animation-delay:{{$sci}}ms">
  <div class="scenario-header">
    <h2>
      <span class="status-indicator {{if $sr.Report.IsPass}}pass{{else}}fail{{end}}"></span>
      场景{{$sci}} · {{$sr.Report.ScenarioName}}
      <span class="scenario-id-tag">{{$sr.Report.ScenarioID}}</span>
    </h2>
    <span class="badge-status {{if $sr.Report.IsPass}}pass{{else}}fail{{end}}">
      {{if $sr.Report.IsPass}}✓ PASS{{else}}✗ FAIL{{end}}
    </span>
  </div>

  <div class="scenario-body">
    <!-- 场景级统计 -->
    <div class="stats-grid" style="margin-bottom:24px">
      {{$scTC := lenSlice $sr.TestCases}}
      {{$scPass := i64 0}}{{$scFail := i64 0}}
      {{range $_,$c := $sr.TestCases}}
        {{if eq $c.TestCase.Result "pass"}}{{$scPass = add $scPass (i64 1)}}{{end}}
        {{if eq $c.TestCase.Result "fail"}}{{$scFail = add $scFail (i64 1)}}{{end}}
      {{end}}
      <div class="stat-card">
        <div class="stat-label">用例总数</div>
        <div class="stat-value">{{$scTC}}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">通过</div>
        <div class="stat-value val-pass">{{$scPass}}</div>
      </div>
      <div class="stat-card {{if gt $scFail (i64 0)}}danger{{end}}">
        <div class="stat-label">失败</div>
        <div class="stat-value {{if gt $scFail (i64 0)}}val-fail{{else}}val-warn{{end}}">{{$scFail}}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">消息数</div>
        <div class="stat-value">{{$sr.Report.TotalMessages}}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">成功率</div>
        <div class="stat-value {{if ge $sr.Report.SuccessRate 100.0}}val-pass{{else if ge $sr.Report.SuccessRate 70.0}}val-warn{{else}}val-fail{{end}}">{{fmtRate $sr.Report.SuccessRate}}</div>
      </div>
    </div>

    {{if not $sr.TestCases}}
    <div class="empty-state">
      <div class="es-icon">📋</div>
      <p>无测试用例执行</p>
    </div>
    {{end}}

    <!-- 用例列表 -->
    {{range $cr := $sr.TestCases}}
    <div class="case" id="tc-{{$cr.TestCase.CaseID}}" style="animation-delay:{{add $sci 100}}ms">
      <div class="case-header" onclick="this.nextElementSibling.classList.toggle('expanded')">
        <h3>
          <span class="status-indicator {{if eq $cr.TestCase.Result "pass"}}pass{{else if eq $cr.TestCase.Result "fail"}}fail{{else}}running{{end}}"></span>
          {{$cr.TestCase.CaseName}}
          {{if $cr.TestCase.CaseID}}<span style="color:var(--text-muted);font-size:11.5px;font-weight:400;font-family:'JetBrains Mono',monospace">{{$cr.TestCase.CaseID}}</span>{{end}}
        </h3>
        <div style="display:flex;align-items:center;gap:10px">
          <span class="case-badge {{resultClass $cr.TestCase.Result}}">{{upper $cr.TestCase.Result}}</span>
          <span class="case-stats">{{$cr.TestCase.TotalMessages}} 条报文 · {{fmtRate $cr.TestCase.SuccessRate}}</span>
        </div>
      </div>
      <div class="case-body">
        {{if $cr.TestCase.TotalMessages}}
        <table class="data-table">
          <thead>
            <tr><th>指标</th><th style="text-align:center">✓ 成功</th><th style="text-align:center">✗ 解码失败</th><th style="text-align:center">⚠ 字段非法</th><th style="text-align:center">✕ 业务失败</th></tr>
          </thead>
          <tbody>
            <tr>
              <td>数量</td>
              <td class="num-cell" style="color:var(--accent)">{{$cr.TestCase.SuccessCount}}</td>
              <td class="num-cell">{{$cr.TestCase.DecodeFail}}</td>
              <td class="num-cell" style="color:var(--warning)">{{$cr.TestCase.InvalidField}}</td>
              <td class="num-cell" style="color:var(--danger)">{{$cr.TestCase.BusinessFail}}</td>
            </tr>
          </tbody>
        </table>
        {{end}}

        {{range $v := $cr.Validations}}
        <div class="validation {{if $v.Passed}}pass{{else}}fail{{end}}">
          <span class="v-icon">{{if $v.Passed}}✓{{else}}✗{{end}}</span>
          <div>
            <strong>[{{$v.FuncCode}}]</strong> {{$v.RuleName}}
            {{if $v.DetailMsg}}<small>{{$v.DetailMsg}}</small>{{end}}
          </div>
        </div>
        {{end}}

        {{if $cr.TestCase.ErrorSummary}}
        <div class="error-summary">
          <span class="es-icon">⚠</span>
          {{$cr.TestCase.ErrorSummary}}
        </div>
        {{end}}

        <a class="detail-link" onclick="document.getElementById('detail-{{$cr.TestCase.CaseID}}').scrollIntoView({behavior:'smooth'});return false;">
          📎 查看报文明细 ↓
        </a>
      </div>
    </div>
    {{end}}
  </div>
</div>
{{$sci = add $sci 80}}
{{end}}

<!-- ══ 报文明细区 ══ -->
{{$di := i64 0}}{{range $_,$sr := .Reports}}
{{range $cr := $sr.TestCases}}
{{if $cr.Messages}}
<div class="detail-section" id="detail-{{$cr.TestCase.CaseID}}" style="animation-delay:{{$di}}ms">
  <div class="detail-section-title">
    <span class="ds-icon">📨</span> 报文明细 · {{$cr.TestCase.CaseName}}
    {{if $cr.TestCase.CaseID}}<span style="color:var(--text-muted);font-size:13px;font-weight:400;font-family:'JetBrains Mono',monospace">{{$cr.TestCase.CaseID}}</span>{{end}}
  </div>

  {{range $m := $cr.Messages}}
  <div class="msg-detail">
    <div class="msg-hdr" onclick="this.nextElementSibling.classList.toggle('expanded')">
      <div class="msg-hdr-left">
        {{if eq $m.Status "success"}}<span class="status-indicator pass" style="width:7px;height:7px"></span>
        {{else}}<span class="status-indicator fail" style="width:7px;height:7px"></span>{{end}}
        <span class="msg-dir-tag {{if eq $m.Direction "发送"}}send{{else if eq $m.Direction "回复"}}reply{{else}}recv{{end}}">{{$m.Direction}}</span>
        <span class="msg-func-code">{{$m.FuncCode}}</span>
        <span style="color:var(--text-muted);font-size:12px">{{$m.Status}}</span>
      </div>
      <span class="msg-ts">{{fmtDateMs $m.Timestamp}}</span>
    </div>
    <div class="msg-body">
      <div class="msg-meta-row">
        <span>📋 Case: <code style="color:var(--accent)">{{$m.CaseID}}</code></span>
        <span>🔢 FuncCode: <strong>{{$m.FuncCode}}</strong></span>
        <span>↔️ 方向: <strong>{{$m.Direction}}</strong></span>
        <span>📊 状态: <strong style="{{if eq $m.Status "success"}}color:var(--accent){{else}}color:var(--danger){{end}}">{{$m.Status}}</strong></span>
      </div>

      {{if $m.ErrorMsg}}
      <div class="msg-error"><strong>错误信息:</strong> {{$m.ErrorMsg}}</div>
      {{end}}

      {{if $m.HexData}}
      <div class="msg-section-title">HEX 原始数据</div>
      <pre class="code-block">{{$m.HexData}}</pre>
      {{end}}

      {{if $m.JSONData}}
      <div class="msg-section-title">JSON 解析结果</div>
      <pre class="code-block">{{$m.JSONData}}</pre>
      {{end}}
    </div>
  </div>
  {{end}}
</div>
{{end}}
{{end}}
{{$di = add $di 60}}
{{end}}

<!-- ══ 页脚 ══ -->
<div class="page-footer">
  报告生成于 <strong>{{.GeneratedAt}}</strong> · <span>NTS-Gater 充电桩协议测试平台</span>
</div>

</div>

<script>
(function(){
  // 自动展开含失败的用例
  document.querySelectorAll('.case-header').forEach(function(h){
    var s=h.querySelector('.case-badge');
    if(s&&(s.classList.contains('fail')||s.textContent.includes('FAIL')||s.textContent.includes('fail'))){
      h.nextElementSibling.classList.add('expanded');
    }
  });

  // 平滑滚动增强
  document.querySelectorAll('.detail-link,a[href^="#"]').forEach(function(a){
    a.addEventListener('click',function(e){
      var t=this.getAttribute('href');
      if(t&&t.startsWith('#')){
        e.preventDefault();
        var el=document.querySelector(t);
        if(el){el.scrollIntoView({behavior:'smooth'})}
      }
    });
  });
})();
</script>
</body>
</html>`
