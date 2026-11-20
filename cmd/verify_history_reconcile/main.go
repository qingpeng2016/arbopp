// 全面审计：DB ↔ Parquet 对账、utc_date、跨平台 BBO（只读）。
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/historydata/pmdata"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

const mapKey = "v1|英超|epl-eve-che-2026-10-17"

type snapRow struct {
	ID           uint64 `gorm:"column:id"`
	Platform     string `gorm:"column:venue_platform"`
	OutcomeID    uint   `gorm:"column:market_outcome_id"`
	UTCDate      string `gorm:"column:day_utc"`
	Ts           int64  `gorm:"column:snapshot_at_ms"`
	Bids         string `gorm:"column:order_book_bids"`
	Asks         string `gorm:"column:order_book_asks"`
	Path         string `gorm:"column:local_file_path"`
	VendorSymbol string `gorm:"column:vendor_symbol"`
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	_ = conf.NewCfg()
	db := boot.NewDBClient()

	var rows []snapRow
	err := db.Raw(`
SELECT id, venue_platform, market_outcome_id, DATE_FORMAT(`+"`utc_date`"+`,'%Y-%m-%d') AS day_utc,
  snapshot_at_ms, order_book_bids, order_book_asks, local_file_path, vendor_symbol
FROM data_history_order_book
WHERE map_key=? AND record_kind='snapshot'
ORDER BY venue_platform, snapshot_at_ms`, mapKey).Scan(&rows).Error
	if err != nil {
		fmt.Println("DB error:", err)
		os.Exit(1)
	}
	fmt.Printf("=== 总量 snapshot=%d ===\n", len(rows))

	// utc_date vs path
	type mis struct{ n int }
	pathMismatch := 0
	utcDates := map[string]int{}
	for _, r := range rows {
		utcDates[r.UTCDate]++
		pathDay := pathDate(r.Path)
		if pathDay != "" && pathDay != r.UTCDate {
			pathMismatch++
		}
	}
	fmt.Println("\n=== utc_date 分布 ===")
	for d, n := range utcDates {
		fmt.Printf("  %s: %d\n", d, n)
	}
	fmt.Printf("utc_date 与 local_file_path 中日历不一致行数: %d / %d\n", pathMismatch, len(rows))

	// 深度 / 单档
	oneLv, multi := 0, 0
	for _, r := range rows {
		if strings.Contains(r.Bids, "],[") || strings.Contains(r.Asks, "],[") {
			multi++
		} else {
			oneLv++
		}
	}
	fmt.Printf("\n=== 深度（全表）单档~%d 多档~%d ===\n", oneLv, multi)
	var multiCnt int64
	_ = db.Raw(`SELECT COUNT(*) FROM data_history_order_book WHERE map_key=? AND record_kind='snapshot'
      AND (order_book_bids LIKE '%,%' OR order_book_asks LIKE '%,%')`, mapKey).Scan(&multiCnt)
	fmt.Printf("SQL 多档行数（含逗号档位）: %d / %d\n", multiCnt, len(rows))
	var samp struct {
		Path string `gorm:"column:local_file_path"`
		Bids string `gorm:"column:order_book_bids"`
		Plat string `gorm:"column:venue_platform"`
		Day  string `gorm:"column:day_utc"`
	}
	_ = db.Raw(`SELECT venue_platform, DATE_FORMAT(`+"`utc_date`"+`,'%Y-%m-%d') AS day_utc,
      local_file_path, LEFT(order_book_bids,220) AS order_book_bids
    FROM data_history_order_book WHERE map_key=? AND record_kind='snapshot' AND venue_platform='POLYMARKET'
    ORDER BY LENGTH(order_book_bids) DESC LIMIT 1`, mapKey).Scan(&samp)
	fmt.Printf("Poly 最长 bids path=%s day=%s lv~%d\n  %s\n", samp.Path, samp.Day, countLevelMarkers(samp.Bids), truncate(samp.Bids, 200))
	var topPaths []struct {
		Path string `gorm:"column:path"`
		N    int64  `gorm:"column:n"`
	}
	_ = db.Raw(`SELECT local_file_path AS path, COUNT(*) AS n FROM data_history_order_book
    WHERE map_key=? AND record_kind='snapshot' GROUP BY local_file_path ORDER BY n DESC LIMIT 5`, mapKey).Scan(&topPaths)
	fmt.Println("=== local_file_path Top5 ===")
	for _, p := range topPaths {
		fmt.Printf("  n=%d %s\n", p.N, p.Path)
	}

	// Parquet 对账（随机 + 首尾）
	fmt.Println("\n=== DB ↔ Parquet 全文对账（按文件缓存）===")
	fileCache := map[string][]pmdata.PredictSnapshot{}
	verify := verifyStats{}
	for i, r := range rows {
		if i%500 == 0 || i < 5 || i == len(rows)-1 {
			doVerify(r, root, fileCache, &verify)
		}
	}
	// 全量对账太慢；再抽 200 条均匀抽样
	step := len(rows) / 200
	if step < 1 {
		step = 1
	}
	for i := 0; i < len(rows); i += step {
		doVerify(rows[i], root, fileCache, &verify)
	}
	fmt.Printf("抽样对账 %d 行: exact_ts_match=%d missing_ts=%d json_mismatch=%d no_file=%d\n",
		verify.total, verify.exact, verify.missTS, verify.badJSON, verify.noFile)

	// 配对 Everton 主胜：poly *-eve vs predict 2905316
	fmt.Println("\n=== 跨平台 BBO（分钟桶，主胜 Everton）===")
	byMinPoly := map[int64]bbo{}
	byMinPred := map[int64]bbo{}
	for _, r := range rows {
		if r.Platform == "POLYMARKET" && !strings.Contains(r.VendorSymbol, "epl-eve-che-2026-10-17-eve") {
			continue
		}
		if r.Platform == "PREDICT" && strings.TrimSpace(r.VendorSymbol) != "2905316" {
			continue
		}
		b, a := topBidAsk(r.Bids, r.Asks)
		if b == 0 && a == 0 {
			continue
		}
		m := r.Ts / 60000
		pt := bbo{b, a, r.Ts}
		if r.Platform == "POLYMARKET" {
			byMinPoly[m] = pt
		} else {
			byMinPred[m] = pt
		}
	}
	compareBBO(byMinPoly, byMinPred)
}

type bbo struct{ bid, ask float64; ts int64 }

type verifyStats struct {
	total, exact, missTS, badJSON, noFile int
}

func doVerify(r snapRow, root string, cache map[string][]pmdata.PredictSnapshot, v *verifyStats) {
	v.total++
	path := r.Path
	if path == "" {
		v.noFile++
		return
	}
	if !filepath.IsAbs(path) {
		cand := filepath.Join(root, path)
		if _, err := os.Stat(cand); err == nil {
			path = cand
		} else if _, err2 := os.Stat(path); err2 == nil {
			// 相对路径可能相对 worker cwd
		} else {
			// 尝试从 path 中提取 var/pmdata 后缀
			if idx := strings.Index(r.Path, "var/pmdata/"); idx >= 0 {
				path = filepath.Join(root, r.Path[idx:])
			}
		}
	}
	snaps, ok := cache[r.Path]
	if !ok {
		data, err := os.ReadFile(path)
		if err != nil {
			v.noFile++
			return
		}
		var perr error
		if r.Platform == "POLYMARKET" {
			snaps, perr = pmdata.ParsePolymarketHonestBookOnly(data, 1_000_000)
		} else {
			snaps, perr = pmdata.ParsePredictL2Parquet(data, 1_000_000)
		}
		if perr != nil {
			v.noFile++
			return
		}
		cache[r.Path] = snaps
		snaps = cache[r.Path]
		ok = true
	}
	_ = ok
	var hit *pmdata.PredictSnapshot
	for i := range snaps {
		if snaps[i].TsMs == r.Ts {
			hit = &snaps[i]
			break
		}
	}
	if hit == nil {
		v.missTS++
		return
	}
	jb, _ := json.Marshal(hit.Bids)
	ja, _ := json.Marshal(hit.Asks)
	if string(jb) == r.Bids && string(ja) == r.Asks {
		v.exact++
	} else {
		v.badJSON++
	}
}

func pathDate(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	parts := strings.Split(p, "/")
	for _, s := range parts {
		if len(s) == 10 && s[4] == '-' && s[7] == '-' {
			if _, err := time.Parse("2006-01-02", s); err == nil {
				return s
			}
		}
	}
	return ""
}

func singleLevel(j string) bool {
	// [[p,s],[p,s]] 多档
	return !strings.Contains(j, "],[")
}

func countLevelMarkers(j string) int {
	if j == "" || j == "[]" {
		return 0
	}
	return strings.Count(j, "],[") + 1
}

func topBidAsk(bids, asks string) (float64, float64) {
	var b [][2]float64
	var a [][2]float64
	_ = json.Unmarshal([]byte(bids), &b)
	_ = json.Unmarshal([]byte(asks), &a)
	bid := 0.0
	for _, lv := range b {
		if lv[0] > bid {
			bid = lv[0]
		}
	}
	ask := 0.0
	for _, lv := range a {
		if ask == 0 || lv[0] < ask {
			ask = lv[0]
		}
	}
	return bid, ask
}

func compareBBO(poly, pred map[int64]bbo) {
	var mins []int64
	for m := range poly {
		if _, ok := pred[m]; ok {
			mins = append(mins, m)
		}
	}
	sort.Slice(mins, func(i, j int) bool { return mins[i] < mins[j] })
	if len(mins) == 0 {
		fmt.Println("无重叠分钟（Poly book 稀疏 + Predict 降采样）")
		return
	}
	var sb, sa float64
	maxBid, maxAsk := 0.0, 0.0
	over5c := 0
	for _, m := range mins {
		p := poly[m]
		q := pred[m]
		db := math.Abs(p.bid - q.bid)
		da := math.Abs(p.ask - q.ask)
		sb += db
		sa += da
		if db > maxBid {
			maxBid = db
		}
		if da > maxAsk {
			maxAsk = da
		}
		if db > 0.05 || da > 0.05 {
			over5c++
		}
	}
	n := float64(len(mins))
	fmt.Printf("重叠分钟: %d（DB 内 Poly eve book × Predict 2905316）\n", len(mins))
	fmt.Printf("平均 |Δbid|=%.4f |Δask|=%.4f  最大 |Δbid|=%.4f |Δask|=%.4f  >5¢点数: %d\n", sb/n, sa/n, maxBid, maxAsk, over5c)
	if len(mins) > 0 {
		m := mins[len(mins)/2]
		p := poly[m]
		q := pred[m]
		fmt.Printf("中位分钟样例 ts_poly=%d bid/ask=%.3f/%.3f vs pred %.3f/%.3f\n", p.ts, p.bid, p.ask, q.bid, q.ask)
	}
	fmt.Println("说明: Poly 仅为 book 事件分钟；Predict 为降采样后的 orderbook 行，重叠分钟偏少但仍应价差在几¢内")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}