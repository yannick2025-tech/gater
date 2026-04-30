// Package report provides PDF test report generation.
package report

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jung-kurt/gofpdf"

	"github.com/yannick2025-tech/nts-gater/internal/model"
	"github.com/yannick2025-tech/nts-gater/internal/recorder"
)

// GeneratePDF 生成测试报告PDF
func GeneratePDF(reportData *model.TestReport, stats []model.FuncCodeStat, archives []model.MessageArchive) (string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// 标题
	pdf.SetFont("Arial", "B", 20)
	pdf.CellFormat(190, 15, "Test Report", "", 0, "C", false, 0, "")
	pdf.Ln(20)

	// 基本信息
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(190, 8, "Basic Information", "", 0, "L", false, 0, "")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 10)
	basicInfo := [][]string{
		{"Session ID", reportData.SessionID},
		{"Post No", fmt.Sprintf("%d", reportData.PostNo)},
		{"Protocol", fmt.Sprintf("%s %s", reportData.ProtocolName, reportData.ProtocolVer)},
		{"Start Time", reportData.StartTime.Format(time.DateTime)},
		{"End Time", func() string { if reportData.EndTime != nil { return reportData.EndTime.Format(time.DateTime) }; return "--" }()},
		{"Duration", formatDuration(reportData.DurationMs)},
		{"Total Messages", fmt.Sprintf("%d", reportData.TotalMessages)},
		{"Success", fmt.Sprintf("%d", reportData.SuccessTotal)},
		{"Failed", fmt.Sprintf("%d", reportData.FailTotal)},
		{"Success Rate", fmt.Sprintf("%.1f%%", reportData.SuccessRate)},
		{"Result", func() string {
			if reportData.IsPass {
				return "PASS"
			}
			return "FAIL"
		}()},
	}

	for _, row := range basicInfo {
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(50, 7, row[0]+":", "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(140, 7, row[1], "", 0, "L", false, 0, "")
		pdf.Ln(7)
	}
	pdf.Ln(5)

	// 功能码统计
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(190, 8, "Func Code Statistics", "", 0, "L", false, 0, "")
	pdf.Ln(10)

	// 表头
	colWidths := []float64{25, 40, 25, 25, 25, 25, 25}
	headers := []string{"Code", "Direction", "Total", "Success", "DecodeFail", "Invalid", "Rate"}
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(200, 200, 200)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(7)

	// 数据行
	pdf.SetFont("Arial", "", 8)
	for _, stat := range stats {
		row := []string{
			stat.FuncCode,
			stat.Direction,
			fmt.Sprintf("%d", stat.TotalMessages),
			fmt.Sprintf("%d", stat.SuccessCount),
			fmt.Sprintf("%d", stat.DecodeFail),
			fmt.Sprintf("%d", stat.InvalidField),
			fmt.Sprintf("%.1f%%", stat.SuccessRate),
		}
		for i, cell := range row {
			pdf.CellFormat(colWidths[i], 6, cell, "1", 0, "C", false, 0, "")
		}
		pdf.Ln(6)
	}
	pdf.Ln(5)

	// 消息存档（最近100条）
	if len(archives) > 0 {
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(190, 8, fmt.Sprintf("Message Archive (total %d, showing recent)", len(archives)), "", 0, "L", false, 0, "")
		pdf.Ln(10)

		// 只显示前100条
		showArchives := archives
		if len(showArchives) > 100 {
			showArchives = showArchives[:100]
		}

		for _, arc := range showArchives {
			if pdf.GetY() > 270 {
				pdf.AddPage()
			}

			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(190, 5, fmt.Sprintf("[%s] %s %s - %s",
				arc.Timestamp.Format("15:04:05"), arc.FuncCode, arc.Direction, arc.Status), "", 0, "L", false, 0, "")
			pdf.Ln(5)

			if arc.HexData != "" {
				pdf.SetFont("Arial", "", 7)
				// 截断过长的hex数据
				hex := arc.HexData
				if len(hex) > 120 {
					hex = hex[:120] + "..."
				}
				pdf.CellFormat(190, 4, "HEX: "+hex, "", 0, "L", false, 0, "")
				pdf.Ln(4)
			}

			if arc.ErrorMsg != "" {
				pdf.SetFont("Arial", "", 7)
				pdf.CellFormat(190, 4, "ERR: "+arc.ErrorMsg, "", 0, "L", false, 0, "")
				pdf.Ln(4)
			}
		}
	}

	// 保存PDF
	outputDir := "reports"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create reports dir: %w", err)
	}

	filename := fmt.Sprintf("report_%s_%s.pdf", reportData.SessionID, time.Now().Format("20060102_150405"))
	filepath := filepath.Join(outputDir, filename)

	if err := pdf.OutputFileAndClose(filepath); err != nil {
		return "", fmt.Errorf("write pdf: %w", err)
	}

	return filepath, nil
}

// formatDuration 格式化持续时间
func formatDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm%.0fs", d.Minutes(), d.Seconds()-float64(int(d.Minutes())*60))
	}
	return fmt.Sprintf("%.0fh%.0fm", d.Hours(), d.Minutes()-float64(int(d.Hours())*60))
}

// Ensure recorder is imported
var _ = recorder.FormatFuncCode
