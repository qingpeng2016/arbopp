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
	writeReportTo(&b, mapKey, rep, outcomeCSV, wsCSV, true)
	if err := os.MkdirAll(dataExportDir, 0755); err != nil {
		return "", err
	}
	return reportPath, os.WriteFile(reportPath, []byte(b.String()), 0644)
}

// writeReportTo：结论置顶；includeAppendix 为 false 时不打历史明细（终端用）。
func writeReportTo(b *strings.Builder, mapKey string, rep fixtureReport, outcomeCSV, wsCSV string, includeAppendix bool) {
	writeConclusionCN(b, mapKey, rep)
	fmt.Fprintf(b, "\n======== 数据统计 ========\n")
	writeDataStatsCN(b, mapKey, rep, outcomeCSV, wsCSV)
	if includeAppendix {
		fmt.Fprintf(b, "\n======== 历史明细（核对用，不是默认结论）========\n")
		writeHitsCN(b, mapKey, "历史回放", rep.YesNoReplay)
		writeHitsCN(b, mapKey, "同毫秒对齐", rep.YesNoSnapshot)
		writeHitsCN(b, mapKey, "独立最低价参考", rep.YesNoHistMin)
		writeTripleHitsCN(b, "三路买yes", rep.TripleYes)
		writeTripleHitsCN(b, "三路买no", rep.TripleNo)
	}
}

func writeConclusionCN(b *strings.Builder, mapKey string, rep fixtureReport) {
	fmt.Fprintf(b, "======== 套利结论 ========\n")
	fmt.Fprintf(b, "场次: %s\n", mapKey)

	if len(rep.YesNoLatest) == 0 && len(rep.TripleYes) == 0 && len(rep.TripleNo) == 0 {
		fmt.Fprintf(b, "二元跨所套利: 无\n")
		if rep.BestYesNoPer > 0 && rep.BestYesNoPer < 0.95 {
			fmt.Fprintf(b, "（最优成本 %.4f 未过深度校验，不算可下单）\n", rep.BestYesNoPer)
		} else if rep.BestYesNoPer >= 0.95 {
			fmt.Fprintf(b, "（最新跨所最优成本 %.4f，高于门槛 0.95）\n", rep.BestYesNoPer)
		} else {
			fmt.Fprintf(b, "（最新盘口未凑齐跨所 YES+NO 深度）\n")
		}
		if len(rep.YesNoReplay) > 0 {
			fmt.Fprintf(b, "（历史回放附录有试算点：对侧盘口非同一毫秒，不算正式结论）\n")
		}
		fmt.Fprintf(b, "三路独赢套利: 无\n")
		return
	}

	if len(rep.YesNoLatest) > 0 {
		h := rep.YesNoLatest[0]
		fmt.Fprintf(b, "二元跨所套利: 有\n")
		if orderBookFromWS() {
			fmt.Fprintf(b, "依据: 各平台最新一条 WS 拼出的盘口（与线上试算一致）\n")
		} else {
			fmt.Fprintf(b, "依据: 各平台最新一条 REST 快照拼出的盘口（orderbook_sync）\n")
		}
		fmt.Fprintf(b, "赛果: %s\n", groupKeyOutcomeCN(h.GroupKey))
		fmt.Fprintf(b, "下单: %s 买 YES + %s 买 NO\n", platformLabel(h.YesPlatform), platformLabel(h.NoPlatform))
		fmt.Fprintf(b, "单份成本含费: %.4f（规则: 须 < 0.95）\n", h.Per)
		fmt.Fprintf(b, "可成交份额约: %.2f\n", h.Qty)
		if h.Per < 0.50 {
			fmt.Fprintf(b, "警示: 成本异常偏低，务必用csv核价\n")
		}
		fmt.Fprintf(b, "核对: %s\n", sqlVerifyHit(mapKey, h))
	} else {
		fmt.Fprintf(b, "二元跨所套利: 无（最新快照未命中）\n")
	}

	if len(rep.TripleYes) > 0 || len(rep.TripleNo) > 0 {
		fmt.Fprintf(b, "三路独赢套利: 有（见报告明细）\n")
	} else {
		fmt.Fprintf(b, "三路独赢套利: 无\n")
	}

	if len(rep.YesNoReplay) > 0 || len(rep.YesNoSnapshot) > 0 {
		fmt.Fprintf(b, "补充: 历史回放命中 %d 条，同毫秒命中 %d 条（仅作核对，以上「最新」为准）\n",
			len(rep.YesNoReplay), len(rep.YesNoSnapshot))
	}
}

func groupKeyOutcomeCN(gk string) string {
	s := lastSegmentGroupKey(gk)
	s = strings.ReplaceAll(s, "_", " ")
	return s
}

func writeDataStatsCN(b *strings.Builder, mapKey string, rep fixtureReport, outcomeCSV, wsCSV string) {
	fmt.Fprintf(b, "盘口outcome总数: %d\n", rep.OutcomeCount)
	fmt.Fprintf(b, "各平台 outcome: %s\n", formatPlatformCountsInline(rep.OutcomesByPlatform))
	fmt.Fprintf(b, "订单簿%s总行数: %d（表 %s）\n", bookSourceLabelCN(), rep.WSSnapshotRows, orderBookTable)
	fmt.Fprintf(b, "各平台行数: %s\n", formatPlatformCountsInline(rep.WSByPlatform))
	fmt.Fprintf(b, "跨平台深度齐组数: %d\n", rep.CrossPlatGroups)
	fmt.Fprintf(b, "盘口csv: %s\n", outcomeCSV)
	fmt.Fprintf(b, "订单簿csv: %s\n", wsCSV)
}

func writeHitsCN(b *strings.Builder, mapKey string, title string, hits []yesnoHit) {
	if len(hits) == 0 {
		return
	}
	fmt.Fprintf(b, "[%s]\n", title)
	if title == "历史回放" {
		if orderBookFromWS() {
			fmt.Fprintf(b, "  说明: 按 WS 行 id 回放；「时刻」= 触发那条 WS 的毫秒，对侧为内存里更早的簿。勿用「同毫秒 IN 两平台」SQL 核对。\n")
		} else {
			fmt.Fprintf(b, "  说明: 按 REST 落库 id 回放（orderbook_sync 插入节奏）；对侧可能非同一毫秒。勿用「同毫秒 IN 两平台」SQL 核对。\n")
		}
	}
	for i, h := range hits {
		fmt.Fprintf(b, "  %d. %s\n", i+1, hitOneLiner(h))
		fmt.Fprintf(b, "     %s\n", sqlVerifyHit(mapKey, h))
	}
}

func writeTripleHitsCN(b *strings.Builder, title string, hits []tripleHit) {
	if len(hits) == 0 {
		return
	}
	fmt.Fprintf(b, "[%s] %d 条\n", title, len(hits))
	for i, h := range hits {
		fmt.Fprintf(b, "  #%d 成本%.4f 平台 %s/%s/%s\n", i+1, h.Per,
			platformLabel(h.DrawPlat), platformLabel(h.TeamAPlat), platformLabel(h.TeamBPlat))
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
	writeReportTo(&b, mapKey, rep, paths[0], paths[1], false)
	fmt.Print(b.String())
	if len(paths) > 2 && paths[2] != "" {
		fmt.Println("完整明细已写入:", paths[2])
	}
}

func summarizeCNReportOnly(reports []fixtureReport) string {
	if len(reports) == 0 {
		return "场次数: 0"
	}
	var b strings.Builder
	for i, r := range reports {
		if i > 0 {
			b.WriteString("\n")
		}
		writeConclusionCN(&b, r.MapKey, r)
	}
	return b.String()
}