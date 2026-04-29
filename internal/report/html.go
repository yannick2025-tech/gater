// Package report provides HTML test report generation.
package report

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
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

// GenerateHTML 生成 V3 HTML 测试报告
func GenerateHTML(sessionID string) (string, error) {
	db := database.GetDB()
	if db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	// 1. 查询 session
	var sess model.Session
	if err := db.Where("id = ?", sessionID).First(&sess).Error; err != nil {
		return "", fmt.Errorf("session not found: %w", err)
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

		// 无用例时也保留场景信息
		if len(cases) == 0 {
			// 不属于特定用例的报文
			var msgs []model.MessageArchive
			db.Where("session_id = ? AND (case_id = '' OR case_id IS NULL)", sessionID).
				Order("timestamp ASC").Find(&msgs)
			if len(msgs) > 0 {
				caseReports = append(caseReports, CaseReport{
					TestCase: model.TestCase{
						CaseID:   "",
						CaseName: "其他报文",
						Status:   "completed",
						Result:   "pass",
					},
					Messages: msgs,
				})
			}
		}

		sr.TestCases = caseReports
		scenarioReports = append(scenarioReports, sr)
	}

	// 4. 渲染 HTML
	data := HTMLReportData{
		Session:     sess,
		Reports:     scenarioReports,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	outputDir := "reports"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create reports dir: %w", err)
	}

	filename := fmt.Sprintf("report_%s_%s.html", sessionID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(outputDir, filename)

	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create html file: %w", err)
	}
	defer f.Close()

	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"fmtDate":    func(t time.Time) string { return t.Format("2006-01-02 15:04:05") },
		"fmtDateMs":  func(t time.Time) string { return t.Format("2006-01-02 15:04:05.000") },
		"fmtRate":    func(r float64) string { return fmt.Sprintf("%.1f%%", r) },
		"fmtDur":     func(ms int64) string { return formatDuration(ms) },
		"statusIcon": func(passed bool) string { if passed { return "✅" }; return "❌" },
		"resultClass": func(r string) string {
			switch r {
			case "pass":
				return "result-pass"
			case "fail":
				return "result-fail"
			default:
				return "result-skip"
			}
		},
	}).Parse(reportHTMLTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	if err := tmpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return filePath, nil
}

// reportHTMLTemplate HTML报告模板
const reportHTMLTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>测试报告 - {{.Session.ID}}</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;color:#333;background:#f5f7fa;line-height:1.6}
.container{max-width:1200px;margin:0 auto;padding:20px}

/* 封面 */
.cover{background:linear-gradient(135deg,#667eea 0%,#764ba2 100%);color:#fff;padding:40px;border-radius:12px;margin-bottom:30px}
.cover h1{font-size:28px;margin-bottom:10px}
.cover .meta{opacity:0.9;font-size:14px}
.cover .meta span{margin-right:20px}

/* 统计卡片 */
.stats-row{display:flex;gap:16px;margin-bottom:30px;flex-wrap:wrap}
.stat-card{background:#fff;border-radius:8px;padding:20px;flex:1;min-width:150px;box-shadow:0 2px 8px rgba(0,0,0,0.06)}
.stat-card .label{font-size:12px;color:#999;margin-bottom:4px}
.stat-card .value{font-size:24px;font-weight:700}
.stat-card .value.pass{color:#67c23a}
.stat-card .value.fail{color:#f56c6c}
.stat-card .value.warn{color:#e6a23c}

/* 场景区 */
.scenario{background:#fff;border-radius:8px;margin-bottom:24px;box-shadow:0 2px 8px rgba(0,0,0,0.06);overflow:hidden}
.scenario-header{background:#f0f2f5;padding:16px 20px;border-bottom:1px solid #e4e7ed}
.scenario-header h2{font-size:18px;display:flex;align-items:center;gap:8px}
.scenario-header .scenario-id{font-size:12px;color:#999;font-weight:400}
.scenario-body{padding:20px}

/* 用例区 */
.case{border:1px solid #e4e7ed;border-radius:6px;margin-bottom:16px;overflow:hidden}
.case-header{padding:12px 16px;background:#fafafa;border-bottom:1px solid #e4e7ed;display:flex;justify-content:space-between;align-items:center;cursor:pointer}
.case-header h3{font-size:15px;display:flex;align-items:center;gap:6px}
.case-header .case-status{font-size:13px}
.case-body{padding:16px;display:none}
.case-body.expanded{display:block}

/* 统计表 */
.stats-table{width:100%;border-collapse:collapse;margin:12px 0;font-size:13px}
.stats-table th{background:#f0f2f5;padding:8px 12px;text-align:left;font-weight:600;border:1px solid #e4e7ed}
.stats-table td{padding:8px 12px;border:1px solid #e4e7ed}
.stats-table tr:hover td{background:#f5f7fa}

/* 结果标签 */
.result-pass{color:#67c23a;font-weight:600}
.result-fail{color:#f56c6c;font-weight:600}
.result-skip{color:#909399}

/* 校验结果 */
.validation{margin:8px 0;padding:8px 12px;border-radius:4px;font-size:13px}
.validation.pass{background:#f0f9eb;border:1px solid #e1f3d8;color:#67c23a}
.validation.fail{background:#fef0f0;border:1px solid #fde2e2;color:#f56c6c}

/* 报文明细 */
.msg-detail{margin:12px 0;border:1px solid #e4e7ed;border-radius:6px;overflow:hidden}
.msg-detail-header{padding:8px 12px;background:#f0f2f5;font-size:13px;font-weight:600;cursor:pointer;display:flex;justify-content:space-between}
.msg-detail-body{padding:12px;display:none;font-size:13px}
.msg-detail-body.expanded{display:block}
.msg-detail pre{background:#1e1e1e;color:#d4d4d4;padding:12px;border-radius:4px;overflow-x:auto;font-size:12px;line-height:1.5;white-space:pre-wrap;word-break:break-all}
.msg-detail .msg-meta{color:#999;font-size:12px;margin-bottom:8px}

/* 锚点导航 */
.nav-bar{position:sticky;top:0;background:#fff;z-index:100;padding:12px 0;border-bottom:1px solid #e4e7ed;margin-bottom:20px}
.nav-bar a{display:inline-block;padding:4px 12px;margin:2px;border-radius:4px;font-size:13px;color:#409eff;text-decoration:none;background:#ecf5ff}
.nav-bar a:hover{background:#409eff;color:#fff}

/* 锚点偏移 */
[id]{scroll-margin-top:80px}

/* 打印优化 */
@media print{
  .nav-bar{display:none}
  .container{padding:0}
  .case-body{display:block!important}
  .msg-detail-body{display:block!important}
  .scenario{break-inside:avoid}
}

/* 空状态 */
.empty{text-align:center;padding:40px;color:#999;font-size:14px}
</style>
</head>
<body>
<div class="container">

<!-- 封面 -->
<div class="cover">
  <h1>📋 充电桩协议测试报告</h1>
  <div class="meta">
    <span>桩号: {{.Session.PostNo}}</span>
    <span>Session: {{.Session.ID}}</span>
    <span>协议: {{.Session.ProtocolName}} {{.Session.ProtocolVersion}}</span>
    <span>连接时间: {{fmtDate .Session.CreatedAt}}</span>
    <span>断开时间: {{if .Session.ClosedAt}}{{fmtDate .Session.ClosedAt.Time}}{{else}}--{{end}}</span>
  </div>
</div>

<!-- 锚点导航 -->
<div class="nav-bar">
{{range $i, $sr := .Reports}}
  <a href="#sc-{{$sr.Report.ScenarioID}}">场景{{$i.Add 1}}: {{$sr.Report.ScenarioName}}</a>
{{end}}
</div>

<!-- 场景列表 -->
{{if not .Reports}}
<div class="empty">无测试场景执行</div>
{{end}}

{{range $i, $sr := .Reports}}
<div class="scenario" id="sc-{{$sr.Report.ScenarioID}}">
  <div class="scenario-header">
    <h2>
      {{if $sr.Report.IsPass}}✅{{else}}❌{{end}}
      场景{{$i.Add 1}}: {{$sr.Report.ScenarioName}}
      <span class="scenario-id">{{$sr.Report.ScenarioID}}</span>
    </h2>
  </div>
  <div class="scenario-body">
    <!-- 场景统计 -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="label">总用例数</div>
        <div class="value">{{$sr.Report.TotalCases}}</div>
      </div>
      <div class="stat-card">
        <div class="label">通过</div>
        <div class="value pass">{{$sr.Report.PassedCases}}</div>
      </div>
      <div class="stat-card">
        <div class="label">失败</div>
        <div class="value fail">{{$sr.Report.FailedCases}}</div>
      </div>
      <div class="stat-card">
        <div class="label">总消息数</div>
        <div class="value">{{$sr.Report.TotalMessages}}</div>
      </div>
      <div class="stat-card">
        <div class="label">成功率</div>
        <div class="value {{if ge $sr.Report.SuccessRate 100.0}}pass{{else}}fail{{end}}">{{fmtRate $sr.Report.SuccessRate}}</div>
      </div>
    </div>

    {{if not $sr.TestCases}}
    <div class="empty">无测试用例执行</div>
    {{end}}

    <!-- 用例列表 -->
    {{range $cr := $sr.TestCases}}
    <div class="case" id="tc-{{$cr.TestCase.CaseID}}">
      <div class="case-header" onclick="this.nextElementSibling.classList.toggle('expanded')">
        <h3>
          {{if eq $cr.TestCase.Result "pass"}}✅{{else if eq $cr.TestCase.Result "fail"}}❌{{else}}⏸️{{end}}
          {{$cr.TestCase.CaseName}}
          {{if $cr.TestCase.CaseID}}<span style="color:#999;font-size:12px;font-weight:400">[{{$cr.TestCase.CaseID}}]</span>{{end}}
        </h3>
        <span class="case-status {{resultClass $cr.TestCase.Result}}">
          {{$cr.TestCase.Result}} | 消息: {{$cr.TestCase.TotalMessages}} | 成功率: {{fmtRate $cr.TestCase.SuccessRate}}
        </span>
      </div>
      <div class="case-body">
        <!-- 报文统计表 -->
        {{if $cr.TestCase.TotalMessages}}
        <table class="stats-table">
          <tr><th>指标</th><th>成功</th><th>解码失败</th><th>字段非法</th><th>业务失败</th></tr>
          <tr>
            <td>数量</td>
            <td>{{$cr.TestCase.SuccessCount}}</td>
            <td>{{$cr.TestCase.DecodeFail}}</td>
            <td>{{$cr.TestCase.InvalidField}}</td>
            <td>{{$cr.TestCase.BusinessFail}}</td>
          </tr>
        </table>
        {{end}}

        <!-- 校验结果 -->
        {{range $v := $cr.Validations}}
        <div class="validation {{if $v.Passed}}pass{{else}}fail{{end}}">
          {{statusIcon $v.Passed}} [{{$v.FuncCode}}] {{$v.RuleName}}
          {{if $v.DetailMsg}}<br><small>{{$v.DetailMsg}}</small>{{end}}
        </div>
        {{end}}

        <!-- 错误摘要 -->
        {{if $cr.TestCase.ErrorSummary}}
        <div style="margin:8px 0;padding:8px 12px;background:#fef0f0;border-radius:4px;font-size:13px;color:#f56c6c">
          ⚠️ {{$cr.TestCase.ErrorSummary}}
        </div>
        {{end}}

        <!-- 报文明细 -->
        <div style="margin-top:12px">
          <a href="#detail-{{$cr.TestCase.CaseID}}" style="font-size:13px;color:#409eff;text-decoration:none">📎 查看报文明细 ↓</a>
        </div>
      </div>
    </div>
    {{end}}
  </div>
</div>
{{end}}

<!-- 报文明细区 -->
{{range $i, $sr := .Reports}}
{{range $cr := $sr.TestCases}}
{{if $cr.Messages}}
<div id="detail-{{$cr.TestCase.CaseID}}" style="margin-bottom:24px">
  <h3 style="font-size:16px;margin:16px 0 8px;padding-bottom:8px;border-bottom:2px solid #409eff">
    📨 报文明细: {{$cr.TestCase.CaseName}}
    {{if $cr.TestCase.CaseID}}<span style="color:#999;font-size:13px">[{{$cr.TestCase.CaseID}}]</span>{{end}}
  </h3>
  {{range $m := $cr.Messages}}
  <div class="msg-detail">
    <div class="msg-detail-header" onclick="this.nextElementSibling.classList.toggle('expanded')">
      <span>
        {{if eq $m.Status "success"}}✅{{else}}❌{{end}}
        [{{$m.FuncCode}}] {{$m.Direction}} - {{$m.Status}}
      </span>
      <span style="color:#999">{{fmtDate $m.Timestamp}}</span>
    </div>
    <div class="msg-detail-body">
      <div class="msg-meta">用例: {{$m.CaseID}} | 功能码: {{$m.FuncCode}} | 方向: {{$m.Direction}} | 状态: {{$m.Status}}</div>
      {{if $m.ErrorMsg}}<div style="color:#f56c6c;margin-bottom:8px">错误: {{$m.ErrorMsg}}</div>{{end}}
      {{if $m.HexData}}
      <div style="margin-bottom:8px"><strong>HEX:</strong></div>
      <pre>{{$m.HexData}}</pre>
      {{end}}
      {{if $m.JSONData}}
      <div style="margin-top:8px;margin-bottom:8px"><strong>JSON:</strong></div>
      <pre>{{$m.JSONData}}</pre>
      {{end}}
    </div>
  </div>
  {{end}}
</div>
{{end}}
{{end}}
{{end}}

<!-- 页脚 -->
<div style="text-align:center;padding:20px;color:#999;font-size:12px;border-top:1px solid #e4e7ed;margin-top:30px">
  报告生成时间: {{.GeneratedAt}} | NTS-Gater 充电桩协议测试平台
</div>

</div>

<script>
// 自动展开含失败用例的 case-body
document.querySelectorAll('.case-header').forEach(header => {
  const statusEl = header.querySelector('.case-status');
  if (statusEl && (statusEl.textContent.includes('fail') || statusEl.textContent.includes('error'))) {
    header.nextElementSibling.classList.add('expanded');
  }
});
</script>
</body>
</html>`
