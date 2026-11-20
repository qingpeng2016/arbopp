// 对照本地 Parquet 与入库逻辑：时间戳、BBO、跨平台一致性（只读）。
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/historydata/pmdata"
)

const mapKey = "v1|英超|epl-eve-che-2026-10-17"

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	day := "2026-10-17"
	polyPath := filepath.Join(root, "var/pmdata/polymarket/l2", day, "epl-eve-che-2026-10-17-eve.parquet")
	predPath := filepath.Join(root, "var/pmdata/predict_fun/l2", day, "2905316.parquet")

	fmt.Println("=== 1. 文件是否存在 ===")
	fmt.Println("POLY:", polyPath, fileOK(polyPath))
	fmt.Println("PRED:", predPath, fileOK(predPath))

	polyRaw, _ := pmdata.ParsePolymarketBookAndPCRaw(read(polyPath))
	predSnaps := parsePredictSnaps(read(predPath))

	fmt.Println("\n=== 2. PMData 源形态（未降采样）===")
	fmt.Printf("Poly book 事件(有列表): %d  price_change: %d\n", polyRaw.BookRows, polyRaw.PcRows)
	fmt.Printf("Predict orderbook 行: %d\n", len(predSnaps))

	fmt.Println("\n=== 3. 解析器输出（当前代码，含 Poly carry）===")
	polyParsed, _ := pmdata.ParsePolymarketL2Parquet(read(polyPath), 500)
	predParsed, _ := pmdata.ParsePredictL2Parquet(read(predPath), 500)
	fmt.Printf("Poly 分钟快照: %d  Predict 分钟快照: %d\n", len(polyParsed), len(predParsed))
	printBBOStats("Poly parsed", polyParsed)
	printBBOStats("Predict parsed", predParsed)

	fmt.Println("\n=== 4. 同源 BBO 对比（按分钟桶，Poly 用 price_change 真 BBO；Predict 用行内最优）===")
	var polyPC []bboPoint
	for _, p := range polyRaw.PcSnaps {
		polyPC = append(polyPC, bboPoint{ts: p.Ts, bid: p.BestBid, ask: p.BestAsk})
	}
	polyBBO := bboByMinuteFromPolyPC(polyPC)
	predBBO := bboByMinuteFromSnaps(predSnaps)
	compareCrossPlatform(polyBBO, predBBO)

	fmt.Println("\n=== 5. Poly：book 事件 vs 入库 JSON（抽样对照 Parquet）===")
	honestPoly, _ := pmdata.ParsePolymarketHonestBookOnly(read(polyPath), 500)
	fmt.Printf("仅 book 原文（无 carry）分钟快照: %d\n", len(honestPoly))
	if len(honestPoly) > 0 {
		s := honestPoly[0]
		fmt.Printf("  首条 ts=%d bid_lv=%d top_bid=%.4f top_ask=%.4f\n", s.TsMs, len(s.Bids), topPx(s.Bids), topPx(s.Asks))
	}

	fmt.Println("\n=== 6. 远程 DB 抽样（若可连）===")
	_ = conf.NewCfg()
	db := boot.NewDBClient()
	type row struct {
		ID       uint64 `gorm:"column:id"`
		Platform string `gorm:"column:venue_platform"`
		Ts       int64  `gorm:"column:snapshot_at_ms"`
		Bids     string `gorm:"column:order_book_bids"`
		Path     string `gorm:"column:local_file_path"`
	}
	var rows []row
	err := db.Raw(`SELECT id, venue_platform, snapshot_at_ms, order_book_bids, local_file_path
		FROM data_history_order_book
		WHERE map_key=? AND record_kind='snapshot' AND `+"`utc_date`"+`=? AND venue_platform IN ('POLYMARKET','PREDICT')
		ORDER BY snapshot_at_ms DESC LIMIT 5`, mapKey, day).Scan(&rows).Error
	if err != nil {
		fmt.Println("DB skip:", err)
		return
	}
	for _, r := range rows {
		bidLv, topB := jsonLevelInfo(r.Bids)
		match := ""
		if r.Platform == "POLYMARKET" && r.Path != "" {
			match = verifyPolyRow(read(r.Path), r.Ts, r.Bids)
		}
		fmt.Printf("  id=%d %s ts=%d bid_lv=%d top=%.4f verify=%s\n", r.ID, r.Platform, r.Ts, bidLv, topB, match)
	}
}

func read(path string) []byte {
	b, _ := os.ReadFile(path)
	return b
}

func fileOK(path string) string {
	if _, err := os.Stat(path); err != nil {
		return "MISSING"
	}
	return "OK"
}

func parsePredictSnaps(data []byte) []pmdata.PredictSnapshot {
	s, _ := pmdata.ParsePredictL2Parquet(data, 1_000_000)
	return s
}

type bboPoint struct {
	ts                            int64
	bid, ask                      float64
}

func bboByMinuteFromPolyPC(snaps []bboPoint) map[int64]bboPoint {
	out := make(map[int64]bboPoint)
	for _, s := range snaps {
		b := s.ts / 60000
		out[b] = s
	}
	return out
}

func bboByMinuteFromSnaps(snaps []pmdata.PredictSnapshot) map[int64]bboPoint {
	out := make(map[int64]bboPoint)
	for _, s := range snaps {
		if s.TsMs <= 0 {
			continue
		}
		b := s.TsMs / 60000
		out[b] = bboPoint{ts: s.TsMs, bid: topPx(s.Bids), ask: topAsk(s.Asks)}
	}
	return out
}

func compareCrossPlatform(poly, pred map[int64]bboPoint) {
	var mins []int64
	for b := range poly {
		if _, ok := pred[b]; ok {
			mins = append(mins, b)
		}
	}
	sort.Slice(mins, func(i, j int) bool { return mins[i] < mins[j] })
	if len(mins) == 0 {
		fmt.Println("无重叠分钟")
		return
	}
	var sumBidDiff, sumAskDiff float64
	big := 0
	for _, b := range mins {
		p := poly[b]
		q := pred[b]
		bd := math.Abs(p.bid - q.bid)
		ad := math.Abs(p.ask - q.ask)
		sumBidDiff += bd
		sumAskDiff += ad
		if bd > 0.05 || ad > 0.05 {
			big++
		}
	}
	n := float64(len(mins))
	fmt.Printf("重叠分钟: %d  平均 |Δbid|=%.4f |Δask|=%.4f  价差>5%%点数: %d\n",
		len(mins), sumBidDiff/n, sumAskDiff/n, big)
	fmt.Println("说明: 同一逻辑盘(主胜)跨所价差 1~3¢ 常见；>5¢ 需人工看是否不同 outcome/token")
}

func printBBOStats(name string, snaps []pmdata.PredictSnapshot) {
	if len(snaps) == 0 {
		return
	}
	one, multi := 0, 0
	for _, s := range snaps {
		if len(s.Bids) <= 1 && len(s.Asks) <= 1 {
			one++
		} else {
			multi++
		}
	}
	fmt.Printf("%s: 单档(<=1)=%d 多档=%d  mid样例 ts=%d bid=%.3f ask=%.3f lv=%d/%d\n",
		name, one, multi, snaps[len(snaps)/2].TsMs,
		topPx(snaps[len(snaps)/2].Bids), topAsk(snaps[len(snaps)/2].Asks),
		len(snaps[len(snaps)/2].Bids), len(snaps[len(snaps)/2].Asks))
}

func topPx(b []pmdata.BookLevel) float64 {
	if len(b) == 0 {
		return 0
	}
	best := b[0][0]
	for _, lv := range b {
		if lv[0] > best {
			best = lv[0]
		}
	}
	return best
}

func topAsk(a []pmdata.BookLevel) float64 {
	if len(a) == 0 {
		return 0
	}
	best := a[0][0]
	for _, lv := range a {
		if lv[0] < best || best == 0 {
			best = lv[0]
		}
	}
	return best
}

func jsonLevelInfo(raw string) (levels int, top float64) {
	var pairs [][2]float64
	_ = json.Unmarshal([]byte(raw), &pairs)
	if len(pairs) == 0 {
		return 0, 0
	}
	return len(pairs), pairs[0][0]
}

func verifyPolyRow(data []byte, ts int64, bidsJSON string) string {
	snaps, _ := pmdata.ParsePolymarketHonestBookOnly(data, 1_000_000)
	// 找最近 book 或该分钟 price_change
	parsed, _ := pmdata.ParsePolymarketL2Parquet(data, 1_000_000)
	for _, s := range parsed {
		if s.TsMs == ts {
			b, _ := json.Marshal(s.Bids)
			if string(b) == bidsJSON {
				return "exact_match_parsed"
			}
			return "parsed_diff"
		}
	}
	_ = snaps
	return "ts_not_in_parse"
}