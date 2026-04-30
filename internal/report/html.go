// Package report provides HTML test report generation.
package report

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
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
		"fmtRate":    func(r float64) string { return fmt.Sprintf("%.1f%%", r) },
		"fmtDur":     func(ms int64) string { return formatDuration(ms) },
		"add":        func(a, b int) int { return a + b },
		"mul":        func(a, b int) int64 { return int64(a) * int64(b) },
		"div":        func(a, b int64) float64 {
			if b == 0 { return 0 }
			return float64(a) / float64(b)
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

// reportHTMLTemplate HTML报告模板 — 工业科技仪表盘风格（深色主题）
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
  --bg-primary:#0a0e17;--bg-secondary:#111827;--bg-card:rgba(17,24,39,0.8);
  --bg-card-hover:rgba(24,34,56,0.9);--border:rgba(56,75,108,0.35);
  --border-glow:rgba(0,212,170,0.25);--text-primary:#e8ecf4;--text-secondary:#8b95a8;
  --text-muted:#525d73;--accent:#00d4aa;--accent-dim:rgba(0,212,170,0.12);
  --accent-glow:rgba(0,212,170,0.3);--danger:#ff4d6a;--danger-dim:rgba(255,77,106,0.12);
  --warning:#ffb84d;--warning-dim:rgba(255,184,77,0.12);--info:#4dabf7;
  --radius-sm:6px;--radius-md:10px;--radius-lg:16px;
  --shadow:0 4px 24px rgba(0,0,0,0.4);--shadow-lg:0 8px 40px rgba(0,0,0,0.55);
}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'Noto Sans SC',-apple-system,BlinkMacSystemFont,sans-serif;color:var(--text-primary);
  background:var(--bg-primary);line-height:1.65;min-height:100vh;
  background-image:
    radial-gradient(ellipse 80% 50% at 50% -20%,rgba(0,212,170,0.06),transparent),
    linear-gradient(180deg,var(--bg-primary) 0%,#0d1220 100%);
  background-attachment:fixed}

/* === 背景网格 === */
body::before{
  content:'';position:fixed;inset:0;z-index:0;
  background-image:linear-gradient(var(--border) 1px,transparent 1px),linear-gradient(90deg,var(--border) 1px,transparent 1px);
  background-size:60px 60px;opacity:.18;pointer-events:none}

.container{max-width:1280px;margin:0 auto;padding:28px 32px 48px;position:relative;z-index:1}

/* ═════════ 封面区域 ═════════ */
.cover{
  position:relative;border-radius:var(--radius-lg);padding:42px 44px;margin-bottom:32px;overflow:hidden;
  background:linear-gradient(135deg,rgba(0,212,170,0.08) 0%,rgba(0,100,180,0.06) 50%,rgba(10,14,23,0.95) 100%);
  border:1px solid var(--border-glow);backdrop-filter:blur(12px)
}
.cover::before{
  content:'';position:absolute;top:-120px;right:-80px;width:320px;height:320px;
  background:radial-gradient(circle,var(--accent-glow),transparent 70%);opacity:.35;pointer-events:none}
.cover::after{
  content:'';position:absolute;bottom:-60px;left:-40px;width:200px;height:200px;
  background:radial-gradient(circle,rgba(77,171,247,0.15),transparent 70%);pointer-events:none}
.cover h1{
  font-size:30px;font-weight:900;letter-spacing:-.5px;display:flex;align-items:center;gap:14px;
  position:relative;z-index:1;text-shadow:0 0 36px var(--accent-glow)
}
.cover h1 .icon-mark{
  display:inline-flex;width:44px;height:44px;background:var(--accent-dim);
  border-radius:12px;align-items:center;justify-content:center;font-size:22px;
  border:1px solid var(--border-glow);flex-shrink:0
}
.cover .meta-row{
  display:flex;flex-wrap:wrap;gap:8px 24px;margin-top:20px;position:relative;z-index:1}
.cover .meta-item{
  font-size:13px;color:var(--text-secondary);display:flex;align-items:center;gap:7px;
  padding:6px 14px;background:rgba(255,255,255,.03);border-radius:20px;border:1px solid rgba(255,255,255,.05)}
.cover .meta-item .dot{
  width:8px;height:8px;border-radius:50%;background:var(--accent);box-shadow:0 0 8px var(--accent);flex-shrink:0}
.cover .meta-item.warn-dot .dot{background:var(--warning);box-shadow:0 0 8px var(--warning)}

/* ═════════ 统计卡片区 ═════════ */
.stats-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:16px;margin-bottom:32px}
.stat-card{
  background:var(--bg-card);border:1px solid var(--border);border-radius:var(--radius-md);
  padding:22px 20px;position:relative;overflow:hidden;transition:all .3s ease;
  backdrop-filter:blur(8px)
}
.stat-card:hover{border-color:var(--border-glow);transform:translateY(-2px);box-shadow:var(--shadow)}
.stat-card::before{
  content:'';position:absolute;top:0;left:0;right:0;height:3px;
  background:linear-gradient(90deg,transparent,var(--accent),transparent);opacity:0;transition:opacity .3s}
.stat-card:hover::before{opacity:1}
.stat-card.danger::before{background:linear-gradient(90deg,transparent,var(--danger),transparent);opacity:1}
.stat-card.warning::before{background:linear-gradient(90deg,transparent,var(--warning),transparent);opacity:1}
.stat-card .stat-label{font-size:11.5px;text-transform:uppercase;letter-spacing:1.2px;color:var(--text-muted);font-weight:600;margin-bottom:8px}
.stat-card .stat-value{font-family:'JetBrains Mono',monospace;font-size:32px;font-weight:700;line-height:1.1;
  background:linear-gradient(135deg,var(--text-primary),var(--text-secondary));
  -webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text}
.stat-card .stat-value.val-pass{background:linear-gradient(135deg,#00e6b8,#00d4aa);-webkit-background-clip:text;background-clip:text}
.stat-card .stat-value.val-fail{background:linear-gradient(135deg,#ff6b8a,var(--danger));-webkit-background-clip:text;background-clip:text}
.stat-card .stat-value.val-warn{background:linear-gradient(135deg,#ffc96b,var(--warning));-webkit-background-clip:text;background-clip:text}
.stat-card .stat-sub{font-size:12px;color:var(--text-muted);margin-top:6px}

/* 进度环容器 */
.progress-ring-wrap{position:absolute;right:16px;top:50%;transform:translateY(-50%);width:52px;height:52px;opacity:.15}
.progress-ring{transform:rotate(-90deg)}

/* ═════════ 锚点导航 ═════════ */
.nav-bar{
  position:sticky;top:0;z-index:100;padding:14px 0;margin-bottom:28px;
  background:rgba(10,14,23,0.85);backdrop-filter:blur(16px);
  border-bottom:1px solid var(--border);border-top:1px solid var(--border);
  border-radius:0 0 var(--radius-md) var(--radius-md)
}
.nav-bar .nav-inner{display:flex;gap:8px;flex-wrap:wrap;align-items:center}
.nav-bar a{
  display:inline-flex;align-items:center;gap:6px;padding:7px 16px;border-radius:20px;
  font-size:13px;color:var(--accent);text-decoration:none;font-weight:500;
  background:var(--accent-dim);border:1px solid var(--border-glow);
  transition:all .25s ease
}
.nav-bar a:hover{background:var(--accent);color:var(--bg-primary);box-shadow:0 0 20px var(--accent-glow)}

/* ═════════ 场景区块 ═════════ */
.scenario{
  background:var(--bg-card);border:1px solid var(--border);border-radius:var(--radius-lg);
  margin-bottom:28px;overflow:hidden;backdrop-filter:blur(8px);transition:border-color .3s;
  animation:fadeSlideUp .5s ease both
}
.scenario:hover{border-color:rgba(86,110,155,0.45)}
.scenario-header{
  padding:20px 26px;display:flex;align-items:center;justify-content:space-between;
  border-bottom:1px solid var(--border);background:rgba(255,255,255,.015);
  flex-wrap:wrap;gap:12px
}
.scenario-header h2{font-size:19px;font-weight:700;display:flex;align-items:center;gap:12px}
.scenario-header .badge-status{
  display:inline-flex;align-items:center;gap:6px;padding:5px 13px;border-radius:20px;
  font-size:12px;font-weight:600;letter-spacing:.3px
}
.badge-status.pass{background:var(--accent-dim);color:var(--accent);border:1px solid var(--border-glow)}
.badge-status.fail{background:var(--danger-dim);color:var(--danger);border:1px solid rgba(255,77,106,0.25)}
.scenario-id-tag{font-size:11.5px;color:var(--text-muted);font-family:'JetBrains Mono',monospace;
  background:rgba(255,255,255,.04);padding:3px 10px;border-radius:4px;border:1px solid rgba(255,255,255,.05)}
.scenario-body{padding:24px 26px}

/* ═════════ 用例区块 ═════════ */
.case{
  border:1px solid var(--border);border-radius:var(--radius-md);margin-bottom:16px;
  overflow:hidden;transition:all .25s ease;animation:fadeSlideUp .45s ease both
}
.case:hover{border-color:rgba(86,110,155,0.4)}
.case-header{
  padding:15px 20px;background:rgba(255,255,255,.02);border-bottom:1px solid var(--border);
  display:flex;justify-content:space-between;align-items:center;cursor:pointer;
  transition:background .2s;flex-wrap:wrap;gap:10px
}
.case-header:hover{background:rgba(255,255,255,.045)}
.case-header h3{font-size:15px;font-weight:600;display:flex;align-items:center;gap:10px}
.status-indicator{width:10px;height:10px;border-radius:50%;flex-shrink:0}
.status-indicator.pass{background:var(--accent);box-shadow:0 0 10px var(--accent-glow)}
.status-indicator.fail{background:var(--danger);box-shadow:0 0 10px rgba(255,77,106,.35)}
.status-indicator.running{background:var(--warning);box-shadow:0 0 10px rgba(255,184,77,.35);
  animation:pulse 1.8s infinite}
.case-badge{
  font-size:12.5px;font-weight:600;padding:4px 12px;border-radius:14px;font-family:'JetBrains Mono',monospace;
  letter-spacing:.3px
}
.case-badge.pass{background:var(--accent-dim);color:var(--accent);border:1px solid var(--border-glow)}
.case-badge.fail{background:var(--danger-dim);color:var(--danger);border:1px solid rgba(255,77,106,0.2)}
.case-badge.skip{background:rgba(148,153,153,0.1);color:var(--text-muted);border:1px solid rgba(148,153,153,0.15)}
.case-stats{font-size:12.5px;color:var(--text-secondary);font-family:'JetBrains Mono',monospace}
.case-body{padding:20px;display:none;border-top:1px solid var(--border)}
.case-body.expanded{display:block;animation:expandIn .3s ease}

@keyframes pulse{0%,100%{opacity:1}50%{opacity:.35}}
@keyframes fadeSlideUp{from{opacity:0;transform:translateY(16px)}to{opacity:1;transform:translateY(0)}}
@keyframes expandIn{from{opacity:0;transform:translateY(-8px)}to{opacity:1;transform:translateY(0)}}

/* ═════════ 数据表格 ═════════ */
.data-table{width:100%;border-collapse:separate;border-spacing:0;font-size:13px;
  margin:14px 0;border:1px solid var(--border);border-radius:var(--radius-sm);overflow:hidden}
.data-table thead th{
  padding:11px 16px;text-align:left;font-weight:600;font-size:11.5px;
  text-transform:uppercase;letter-spacing:.8px;color:var(--text-muted);
  background:rgba(255,255,255,.03);border-bottom:1px solid var(--border)
}
.data-table tbody td{padding:11px 16px;border-bottom:1px solid rgba(56,75,108,.18);color:var(--text-secondary)}
.data-table tbody tr:last-child td{border-bottom:none}
.data-table tbody tr:hover td{background:rgba(0,212,170,0.04);color:var(--text-primary)}
.data-table .num-cell{font-family:'JetBrains Mono',monospace;font-weight:600;color:var(--text-primary)}

/* ═════════ 校验结果 ═════════ */
.validation{
  margin:8px 0;padding:11px 16px;border-radius:var(--radius-sm);font-size:13px;
  display:flex;align-items:flex-start;gap:10px;line-height:1.5;border:1px solid transparent;
  animation:fadeSlideUp .35s ease both
}
.validation.pass{background:var(--accent-dim);color:#5eecc4;border-color:rgba(0,212,170,.15)}
.validation.fail{background:var(--danger-dim);color:#ff8a9e;border-color:rgba(255,77,106,.15)}
.validation .v-icon{flex-shrink:0;font-size:15px;margin-top:1px}
.validation small{color:var(--text-secondary);display:block;margin-top:3px;font-size:12px}

/* ═════════ 报文明细 ═════════ */
.msg-detail{margin:10px 0;border:1px solid var(--border);border-radius:var(--radius-md);overflow:hidden;
  transition:border-color .2s}
.msg-detail:hover{border-color:rgba(86,110,155,.4)}
.msg-hdr{
  padding:10px 16px;cursor:pointer;display:flex;justify-content:space-between;align-items:center;
  gap:12px;transition:background .2s;flex-wrap:wrap;
  background:rgba(255,255,255,.02);border-bottom:1px solid var(--border)
}
.msg-hdr:hover{background:rgba(255,255,255,.05)}
.msg-hdr-left{display:flex;align-items:center;gap:10px;min-width:0}
.msg-dir-tag{
  display:inline-block;padding:2px 9px;border-radius:4px;font-size:11px;font-weight:600;
  text-transform:uppercase;letter-spacing:.5px;font-family:'JetBrains Mono',monospace;flex-shrink:0
}
.msg-dir-tag.recv{background:rgba(77,171,247,.12);color:var(--info);border:1px solid rgba(77,171,247,.18)}
.msg-dir-tag.send{background:var(--accent-dim);color:var(--accent);border:1px solid var(--border-glow)}
.msg-dir-tag.reply{background:rgba(168,119,255,.12);color:#a877ff;border:1px solid rgba(168,119,255,.18)}
.msg-func-code{font-family:'JetBrains Mono',monospace;font-size:13px;font-weight:600;color:var(--text-primary)}
.msg-ts{font-family:'JetBrains Mono',monospace;font-size:11.5px;color:var(--text-muted);white-space:nowrap}
.msg-body{padding:16px;display:none;background:rgba(0,0,0,.15);border-top:1px solid var(--border)}
.msg-body.expanded{display:block;animation:expandIn .25s ease}
.msg-meta-row{font-size:12px;color:var(--text-muted);margin-bottom:12px;display:flex;flex-wrap:wrap;gap:12px}
.msg-meta-row span{display:inline-flex;align-items:center;gap:4px}
.msg-error{color:var(--danger);font-size:13px;margin-bottom:10px;padding:8px 12px;
  background:var(--danger-dim);border-radius:var(--radius-sm);border:1px solid rgba(255,77,106,.15)}
.msg-section-title{font-size:11.5px;font-weight:700;text-transform:uppercase;letter-spacing:1px;
  color:var(--text-muted);margin:10px 0 6px;display:flex;align-items:center;gap:8px}
.msg-section-title::after{content:'';flex:1;height:1px;background:var(--border)}

/* 代码块 */
.code-block{
  background:#0d1117;border:1px solid rgba(56,75,108,.25);border-radius:var(--radius-sm);
  padding:14px 16px;overflow-x:auto;font-family:'JetBrains Mono',monospace;
  font-size:12px;line-height:1.7;white-space:pre-wrap;word-break:break-all;
  color:#c9d1d9;max-height:380px;overflow-y:auto
}
.code-block::-webkit-scrollbar{width:6px;height:6px}
.code-block::-webkit-scrollbar-track{background:transparent}
.code-block::-webkit-scrollbar-thumb{background:rgba(82,93,115,.4);border-radius:3px}

/* ═════════ 报文明细标题区 ═════════ */
.detail-section{margin-bottom:28px;animation:fadeSlideUp .5s ease both}
.detail-section-title{
  font-size:17px;font-weight:700;margin:20px 0 14px;padding:12px 18px;
  display:flex;align-items:center;gap:10px;
  background:rgba(255,255,255,.02);border-radius:var(--radius-md);
  border-left:3px solid var(--accent);border:1px solid var(--border);border-left:3px solid var(--accent)
}
.detail-section-title .ds-icon{font-size:18px}

/* 错误摘要 */
.error-summary{
  margin:10px 0;padding:12px 16px;background:var(--danger-dim);border-radius:var(--radius-sm);
  border:1px solid rgba(255,77,106,.15);font-size:13px;color:#ff8a9e;
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
.empty-state .es-icon{font-size:40px;margin-bottom:12px;opacity:.4}

/* ═════════ 锚点偏移 ═════════ */
[id]{scroll-margin-top:76px}

/* 查看报文明细链接 */
.detail-link{
  display:inline-flex;align-items:center;gap:6px;margin-top:14px;padding:8px 16px;
  font-size:13px;color:var(--accent);text-decoration:none;font-weight:500;
  background:var(--accent-dim);border:1px solid var(--border-glow);border-radius:20px;
  transition:all .25s ease;cursor:pointer
}
.detail-link:hover{background:var(--accent);color:var(--bg-primary);box-shadow:0 0 20px var(--accent-glow)}

/* ═════════ 打印样式 ═════════ */
@media print{
  body{background:#fff!important;color:#222!important}
  body::before{display:none}
  .nav-bar{display:none!important}
  .container{padding:0!important;max-width:100%!important}
  .cover{background:linear-gradient(135deg,#1a365d 0%,#2c5282 100%)!important;color:#fff!important;border-radius:0}
  .scenario,.case,.msg-detail{border-color:#ccc!important;break-inside:avoid}
  .case-body,.msg-body{display:block!important}
  .stat-card{border-color:#ddd!important;page-break-inside:avoid}
  .code-block{background:#f5f5f5!important;color:#333!important;border-color:#ddd!important}
  .validation.pass{background:#e8fdf0!important;color:#22863a!important;border-color:#c3e9cb!important}
  .validation.fail{background:#ffe8e8!important;color:#cb2431!important;border-color: #ffcaca!important}
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
{{$totalScenarios := len .Reports}}
{{$totalCases := 0}}{{$totalPass := 0}}{{$totalFail := 0}}{{$totalMsgs := 0}}
{{range $_,$sr := .Reports}}
  {{$totalCases = add $totalCases $sr.Report.TotalCases}}
  {{$totalPass = add $totalPass $sr.Report.PassedCases}}
  {{$totalFail = add $totalFail $sr.Report.FailedCases}}
  {{$totalMsgs = add $totalMsgs $sr.Report.TotalMessages}}
{{end}}

<div class="stats-grid">
  <div class="stat-card">
    <div class="stat-label">场景数</div>
    <div class="stat-value">{{$totalScenarios}}</div>
  </div>
  <div class="stat-card">
    <div class="stat-label">总用例</div>
    <div class="stat-value">{{$totalCases}}</div>
    <div class="stat-sub">{{if gt $totalCases 0}}{{printf "%.1f" (div (mul $totalPass 100) $totalCases)}}% 通过率{{else}}--{{end}}</div>
  </div>
  <div class="stat-card">
    <div class="stat-label">通过</div>
    <div class="stat-value val-pass">{{$totalPass}}</div>
  </div>
  <div class="stat-card {{if gt $totalFail 0}}danger{{end}}">
    <div class="stat-label">失败</div>
    <div class="stat-value {{if gt $totalFail 0}}val-fail{{else}}val-warn{{end}}">{{$totalFail}}</div>
  </div>
  <div class="stat-card">
    <div class="stat-label">报文总量</div>
    <div class="stat-value">{{$totalMsgs}}</div>
  </div>
</div>

<!-- ══ 锚点导航 ══ -->
{{if gt $totalScenarios 1}}
<div class="nav-bar">
  <div class="nav-inner">
  {{$navIdx := 0}}{{range $_,$sr := .Reports}}
    <a href="#sc-{{$sr.Report.ScenarioID}}">#{{$navIdx}} {{$sr.Report.ScenarioName}}</a>
    {{$navIdx = add $navIdx 1}}
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

{{$sci := 0}}{{range $_,$sr := .Reports}}
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
      <div class="stat-card">
        <div class="stat-label">用例总数</div>
        <div class="stat-val ue">{{$sr.Report.TotalCases}}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">通过</div>
        <div class="stat-value val-pass">{{$sr.Report.PassedCases}}</div>
      </div>
      <div class="stat-card {{if gt $sr.Report.FailedCases 0}}danger{{end}}">
        <div class="stat-label">失败</div>
        <div class="stat-value {{if gt $sr.Report.FailedCases 0}}val-fail{{else}}val-warn{{end}}">{{$sr.Report.FailedCases}}</div>
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
{{$di := 0}}{{range $_,$sr := .Reports}}
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
