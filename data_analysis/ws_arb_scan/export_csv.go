package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 查询结果 CSV 落盘到 data_analysis/ws_arb_scan/data/（见 .gitignore）。
func exportFixtureCSV(mapKey string, outcomes []outcomeRec, wsRows []wsRow) (outcomePath, wsPath string, err error) {
	base := sanitizeMapKeyForFile(mapKey)
	if err := os.MkdirAll(dataExportDir, 0755); err != nil {
		return "", "", err
	}
	outcomePath = filepath.Join(dataExportDir, base+"_盘口outcome.csv")
	wsPath = filepath.Join(dataExportDir, base+"_订单簿WS.csv")

	if err := writeOutcomeCSV(outcomePath, mapKey, outcomes); err != nil {
		return "", "", err
	}
	if err := writeWSCSV(wsPath, mapKey, wsRows); err != nil {
		return "", "", err
	}
	return outcomePath, wsPath, nil
}

func sanitizeMapKeyForFile(mapKey string) string {
	s := strings.TrimSpace(mapKey)
	s = strings.ReplaceAll(s, "|", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, " ", "_")
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}

func writeOutcomeCSV(path, mapKey string, rows []outcomeRec) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{
		"场次对齐键", "平台", "数据库id", "平台盘口id", "赛果名称", "赛果类型",
		"yes_token", "no_token", "slug", "联赛",
	})
	for _, o := range rows {
		_ = w.Write([]string{
			mapKey, o.Platform, fmt.Sprint(o.ID), o.PlatformMarketOutcomeID,
			o.OutcomeName, o.OutcomeType, o.YesTokenID, o.NoTokenID, o.Slug, o.Competition,
		})
	}
	return w.Error()
}

func writeWSCSV(path, mapKey string, rows []wsRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{
		"场次对齐键", "平台", "平台盘口id", "token_id", "快照时间毫秒", "行id", "卖盘json", "买盘json",
	})
	for _, r := range rows {
		_ = w.Write([]string{
			mapKey, r.Platform, r.PlatformMarketOutcomeID, r.TokenID,
			fmt.Sprint(r.SnapshotAtMs), fmt.Sprint(r.ID), r.Asks, r.Bids,
		})
	}
	return w.Error()
}

func countByPlatformPlatforms(rows []wsRow) map[string]int {
	c := map[string]int{}
	for _, r := range rows {
		p := platformLabel(r.Platform)
		c[p]++
	}
	return c
}

func countOutcomesByPlatform(outcomes []outcomeRec) map[string]int {
	c := map[string]int{}
	for _, o := range outcomes {
		p := platformLabel(o.Platform)
		c[p]++
	}
	return c
}

func platformLabel(platform string) string {
	p := strings.ToUpper(strings.TrimSpace(platform))
	switch p {
	case "POLYMARKET":
		return "Polymarket"
	case "PREDICT":
		return "Predict"
	case "SXBET":
		return "SXBet"
	case "KALSHI":
		return "Kalshi"
	default:
		if platform == "" {
			return "未知平台"
		}
		return platform
	}
}

func formatPlatformCounts(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var lines []string
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("  %s: %d", k, m[k]))
	}
	return lines
}