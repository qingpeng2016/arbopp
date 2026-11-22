package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writeChineseReport(mapKey string, rep fixtureReport, outcomeCSV, wsCSV string) (reportPath string, err error) {
	base := sanitizeMapKeyForFile(mapKey)
	reportPath = filepath.Join(dataExportDir, base+"_套利分析报告.txt")
	var b strings.Builder
	writeReportLines(&b, mapKey, rep, outcomeCSV, wsCSV)
	if err := os.MkdirAll(dataExportDir, 0755); err != nil {
		return "", err
	}
	return reportPath, os.WriteFile(reportPath, []byte(b.String()), 0644)
}

func writeReportLines(b *strings.Builder, mapKey string, rep fixtureReport, outcomeCSV, wsCSV string) {
	fmt.Fprintf(b, "场次对齐键: %s\n", mapKey)
	fmt.Fprintf(b, "盘口outcome总数: %d\n", rep.OutcomeCount)
	fmt.Fprintf(b, "各平台盘口outcome条数:\n")
	for _, line := range formatPlatformCounts(rep.OutcomesByPlatform) {
		fmt.Fprintf(b, "%s\n", line)
	}
	fmt.Fprintf(b, "订单簿WS总行数: %d\n", rep.WSSnapshotRows)
	fmt.Fprintf(b, "各平台订单簿WS条数:\n")
	for _, line := range formatPlatformCounts(rep.WSByPlatform) {
		fmt.Fprintf(b, "%s\n", line)
	}
	fmt.Fprintf(b, "有盘口的平台: %s\n", strings.Join(rep.PlatformsCN(), "、"))
	fmt.Fprintf(b, "赛果分组数: %d\n", rep.OutcomeGroups)
	fmt.Fprintf(b, "跨平台且深度齐的组数: %d\n", rep.CrossPlatGroups)
	fmt.Fprintf(b, "最新顶价下最优成本含费: %.4f\n", rep.BestYesNoPer)
	if rep.BestYesNoPer <= 0 {
		fmt.Fprintf(b, "说明: 无可算跨所组合，或缺 bid 导致 no 边未齐\n")
	} else if rep.BestYesNoPer >= 0.95 {
		fmt.Fprintf(b, "说明: 有跨所报价但单份成本未低于 0.95 门槛\n")
	}

	fmt.Fprintf(b, "二元套利命中数-最新快照: %d\n", len(rep.YesNoLatest))
	fmt.Fprintf(b, "二元套利命中数-历史回放: %d\n", len(rep.YesNoReplay))
	fmt.Fprintf(b, "二元套利命中数-同毫秒对齐: %d\n", len(rep.YesNoSnapshot))
	fmt.Fprintf(b, "二元套利命中数-独立最低价参考: %d\n", len(rep.YesNoHistMin))
	fmt.Fprintf(b, "三路独赢套利命中数-买yes: %d\n", len(rep.TripleYes))
	fmt.Fprintf(b, "三路独赢套利命中数-买no: %d\n", len(rep.TripleNo))
	if rep.Note != "" {
		fmt.Fprintf(b, "备注: %s\n", rep.Note)
	}
	fmt.Fprintf(b, "导出盘口csv: %s\n", outcomeCSV)
	fmt.Fprintf(b, "导出订单簿csv: %s\n", wsCSV)

	writeHitsCN(b, "最新快照二元套利", rep.YesNoLatest)
	writeHitsCN(b, "历史回放二元套利", rep.YesNoReplay)
	writeHitsCN(b, "同毫秒二元套利", rep.YesNoSnapshot)
	writeTripleHitsCN(b, "三路买yes", rep.TripleYes)
	writeTripleHitsCN(b, "三路买no", rep.TripleNo)
}

func writeHitsCN(b *strings.Builder, title string, hits []yesnoHit) {
	if len(hits) == 0 {
		return
	}
	fmt.Fprintf(b, "\n--- %s ---\n", title)
	for i, h := range hits {
		fmt.Fprintf(b, "第%d条\n", i+1)
		fmt.Fprintf(b, "  模式: %s\n", hitModeCN(h.Mode))
		fmt.Fprintf(b, "  赛果组: %s\n", h.GroupKey)
		fmt.Fprintf(b, "  买yes平台: %s\n", platformLabel(h.YesPlatform))
		fmt.Fprintf(b, "  买no平台: %s\n", platformLabel(h.NoPlatform))
		fmt.Fprintf(b, "  单份成本含费: %.4f\n", h.Per)
		fmt.Fprintf(b, "  可成交份额: %.4f\n", h.Qty)
		if h.SnapshotMs > 0 {
			fmt.Fprintf(b, "  快照时间毫秒: %d\n", h.SnapshotMs)
		}
	}
}

func writeTripleHitsCN(b *strings.Builder, title string, hits []tripleHit) {
	if len(hits) == 0 {
		return
	}
	fmt.Fprintf(b, "\n--- %s ---\n", title)
	for i, h := range hits {
		fmt.Fprintf(b, "第%d条\n", i+1)
		fmt.Fprintf(b, "  类型: %s\n", h.Kind)
		fmt.Fprintf(b, "  平局平台: %s\n", platformLabel(h.DrawPlat))
		fmt.Fprintf(b, "  主队平台: %s\n", platformLabel(h.TeamAPlat))
		fmt.Fprintf(b, "  客队平台: %s\n", platformLabel(h.TeamBPlat))
		fmt.Fprintf(b, "  总成本: %.4f\n", h.Per)
		fmt.Fprintf(b, "  份额: %.4f\n", h.Qty)
	}
}

func hitModeCN(mode string) string {
	switch mode {
	case "latest":
		return "最新快照"
	case "replay":
		return "历史回放"
	case "snapshot":
		return "同毫秒对齐"
	case "hist_min":
		return "独立最低价参考"
	default:
		return mode
	}
}

func (r fixtureReport) PlatformsCN() []string {
	out := make([]string, len(r.Platforms))
	for i, p := range r.Platforms {
		out[i] = platformLabel(p)
	}
	return out
}

func printChineseSummary(mapKey string, rep fixtureReport, paths ...string) {
	var b strings.Builder
	writeReportLines(&b, mapKey, rep, paths[0], paths[1])
	fmt.Print(b.String())
	if len(paths) > 2 && paths[2] != "" {
		fmt.Println("分析报告已写入:", paths[2])
	}
}

func summarizeCNReportOnly(reports []fixtureReport) string {
	if len(reports) == 0 {
		return "场次数: 0"
	}
	var b strings.Builder
	for i, r := range reports {
		if i > 0 {
			b.WriteString("\n========\n")
		}
		writeReportLines(&b, r.MapKey, r, "(见各场 csv)", "(见各场 csv)")
	}
	return b.String()
}