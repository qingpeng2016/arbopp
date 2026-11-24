// 审计 arb_binary_opportunity 行内自洽，并用之后紧挨的 market_order_book（REST）快照
// 回放同一腿组合是否仍达阈值，估算机会持续多久。
//
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/audit_opp_duration -kinds=ONE_X2 -n=30
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/audit_opp_duration -id=35579 -follow=80
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/onextwo"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	yesTriple = onextwo.OppKindYesTriple
	noTriple  = onextwo.OppKindNoTriple
	binaryYN  = "BINARY_YES_NO"
)

func main() {
	limit := flag.Int("n", 40, "机会条数（有新 schema leg 数据优先）")
	oppID := flag.Uint("id", 0, "单条 id")
	kinds := flag.String("kinds", "ONE_X2", "逗号分隔 opp_kind 前缀；ALL=全部")
	follow := flag.Int("follow", 60, "每条机会向后最多跟踪多少个 REST 快照时刻")
	horizon := flag.Int64("horizon_ms", 0, "向后最长时间；0=不限制（仍受 follow 限制）")
	table := flag.String("book_table", "market_order_book", "订单簿表名")
	verbose := flag.Int("verbose", 3, "打印前 N 条机会的持续明细；0=只汇总")
	tol := flag.Float64("tol", 0.02, "行内 total_cost 与 REST 首快照重算 per 允许偏差")
	slack := flag.Int64("snap_slack_ms", 120000, "机会时刻前后找 REST 批次的毫秒半窗")
	flag.Parse()

	conf.NewCfg()
	db := boot.NewDBClient()

	kindFilter := parseKinds(*kinds)
	rows, err := loadOpps(db, kindFilter, *oppID, *limit)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if len(rows) == 0 {
		fmt.Println("无匹配机会行")
		os.Exit(0)
	}

	type stat struct {
		n, legacy, internalOK, internalBad int
		reproOK, reproMiss, reproNoBook     int
		stillAtLast                         int
		durations                           []float64 // ms
	}
	var s stat
	s.n = len(rows)

	for i, o := range rows {
		flags := internalCheck(o)
		if len(flags) == 1 && strings.Contains(flags[0], "legacy") {
			s.legacy++
			continue
		}
		if len(flags) == 0 {
			s.internalOK++
		} else {
			s.internalBad++
		}

		legs, kind, err := legsFromRow(o)
		if err != nil || len(legs) == 0 {
			continue
		}
		mapKey := ""
		if o.MapKey != nil {
			mapKey = strings.TrimSpace(*o.MapKey)
		}
		if mapKey == "" || o.SnapMs <= 0 {
			continue
		}
		th := decF(o.Threshold)
		snaps, _ := restSnapTimes(db, *table, mapKey, o.SnapMs-*slack, *follow, *horizon)
		after := filterSnapsGE(snaps, o.SnapMs)
		if len(after) == 0 && len(snaps) > 0 {
			// 命中时刻可能晚于最后一拍 REST：从最近一批起向后看
			after = filterSnapsGE(snaps, nearestSnap(snaps, o.SnapMs))
		}
		if len(snaps) == 0 {
			s.reproNoBook++
			continue
		}
		cache := map[int64]map[string]string{}
		reproMs := nearestSnap(snaps, o.SnapMs)
		firstPer, ok0 := replayPerCached(db, *table, mapKey, legs, reproMs, cache)
		if ok0 {
			if math.Abs(firstPer-decF(o.TotalCost)) <= *tol {
				s.reproOK++
			} else {
				s.reproMiss++
			}
		} else {
			s.reproNoBook++
		}

		lastStill := int64(0)
		stillOpen := false
		started := false
		for _, ms := range after {
			per, ok := replayPerCached(db, *table, mapKey, legs, ms, cache)
			if !ok {
				if started {
					break
				}
				continue
			}
			if per >= th {
				if started {
					break
				}
				continue
			}
			started = true
			lastStill = ms
			stillOpen = true
		}
		dur := float64(0)
		if started {
			dur = float64(lastStill - o.SnapMs)
			if dur < 0 {
				dur = 0
			}
			s.durations = append(s.durations, dur)
			if stillOpen && len(after) > 0 && lastStill == after[len(after)-1] {
				s.stillAtLast++
			}
		} else if ok0 && firstPer < th {
			// 命中后无更晚 REST 行，但最近一批仍满足
			s.durations = append(s.durations, 0)
		}

		if i < *verbose {
			printDurationLine(o, kind, th, reproMs, after, lastStill, dur, started && stillOpen, flags, firstPer, ok0)
		}
	}

	fmt.Println("\n=== 机会持续时长（market_order_book / REST 同腿组合）===")
	fmt.Printf("样本: %d（legacy跳过=%d）\n", s.n, s.legacy)
	fmt.Printf("行内自洽: OK=%d BAD=%d\n", s.internalOK, s.internalBad)
	fmt.Printf("首快照 REST 重算: 贴近行内 total_cost(±%.3f)=%d 偏差大=%d 缺簿=%d\n",
		*tol, s.reproOK, s.reproMiss, s.reproNoBook)
	fmt.Printf("仍在机会内到最后一帧 REST: %d\n", s.stillAtLast)
	if len(s.durations) > 0 {
		sort.Float64s(s.durations)
		fmt.Printf("持续 ms: min=%.0f p50=%.0f p90=%.0f max=%.0f\n",
			s.durations[0], pct(s.durations, 0.5), pct(s.durations, 0.9), s.durations[len(s.durations)-1])
	}
	fmt.Println("\n说明:")
	fmt.Println("  · 「仍有机会」= 三/两腿 token 在该 map_key 的 REST snapshot_at_ms 上顶价+费重算 per < 该行 threshold_value")
	fmt.Println("  · REST 约 60s 一拍，持续时长 = 相邻快照间隔的累加，不是毫秒级实盘窗口")
	fmt.Println("  · 行内合理性与 audit_onextwo_opp 同源；阈值仅用于与落库行一致地判断「算不算还有机会」")
}

func parseKinds(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "ALL") {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func loadOpps(db *gorm.DB, kindPrefix []string, id uint, limit int) ([]oppRow, error) {
	q := db.Table("arb_binary_opportunity").
		Select(`id, opp_kind, map_key, threshold_key, total_cost, profit_space, arb_qty, arb_profit,
			threshold_value, snapshot_at_ms,
			leg_1_platform, leg_2_platform, leg_3_platform,
			leg_1_best_ask, leg_2_best_ask, leg_3_best_ask,
			leg_1_order_books, leg_2_order_books, leg_3_order_books,
			leg_1_info, leg_2_info, leg_3_info`)
	if id > 0 {
		q = q.Where("id = ?", id)
	} else {
		q = q.Where(`snapshot_at_ms > 0 AND leg_1_best_ask IS NOT NULL AND leg_1_info LIKE '%token_id%'
			AND map_key IS NOT NULL AND map_key != ''`)
		for _, p := range kindPrefix {
			q = q.Where("opp_kind LIKE ?", p+"%")
		}
		q = q.Order("snapshot_at_ms DESC").Limit(limit)
	}
	var rows []oppRow
	return rows, q.Scan(&rows).Error
}

type oppRow struct {
	ID           uint             `gorm:"column:id"`
	OppKind      string           `gorm:"column:opp_kind"`
	MapKey       *string          `gorm:"column:map_key"`
	ThresholdKey string           `gorm:"column:threshold_key"`
	TotalCost    decimal.Decimal  `gorm:"column:total_cost"`
	ProfitSpace  decimal.Decimal  `gorm:"column:profit_space"`
	ArbQty       decimal.Decimal  `gorm:"column:arb_qty"`
	ArbProfit    decimal.Decimal  `gorm:"column:arb_profit"`
	Threshold    decimal.Decimal  `gorm:"column:threshold_value"`
	SnapMs       int64            `gorm:"column:snapshot_at_ms"`
	L1Plat       *string          `gorm:"column:leg_1_platform"`
	L2Plat       *string          `gorm:"column:leg_2_platform"`
	L3Plat       *string          `gorm:"column:leg_3_platform"`
	L1Ask        *decimal.Decimal `gorm:"column:leg_1_best_ask"`
	L2Ask        *decimal.Decimal `gorm:"column:leg_2_best_ask"`
	L3Ask        *decimal.Decimal `gorm:"column:leg_3_best_ask"`
	L1Books      string           `gorm:"column:leg_1_order_books"`
	L2Books      string           `gorm:"column:leg_2_order_books"`
	L3Books      string           `gorm:"column:leg_3_order_books"`
	L1Info       string           `gorm:"column:leg_1_info"`
	L2Info       string           `gorm:"column:leg_2_info"`
	L3Info       string           `gorm:"column:leg_3_info"`
}

type legSpec struct {
	token    string
	platform string
	fee      float64
}

func legsFromRow(o oppRow) ([]legSpec, string, error) {
	fees := defaultFees()
	switch o.OppKind {
	case yesTriple, noTriple:
		infos := []string{o.L1Info, o.L2Info, o.L3Info}
		plats := []*string{o.L1Plat, o.L2Plat, o.L3Plat}
		var legs []legSpec
		for i := 0; i < 3; i++ {
			tok := tokenForLeg(i, infos[i])
			if tok == "" {
				return nil, o.OppKind, fmt.Errorf("leg%d token", i+1)
			}
			plat := ""
			if plats[i] != nil {
				plat = strings.ToUpper(strings.TrimSpace(*plats[i]))
			}
			legs = append(legs, legSpec{token: tok, platform: plat, fee: fees[plat]})
		}
		return legs, o.OppKind, nil
	case binaryYN:
		t1 := tokenFromBinary(o.L1Info, true)
		t2 := tokenFromBinary(o.L2Info, false)
		if t1 == "" || t2 == "" {
			return nil, o.OppKind, fmt.Errorf("binary tokens")
		}
		p1, p2 := "", ""
		if o.L1Plat != nil {
			p1 = strings.ToUpper(strings.TrimSpace(*o.L1Plat))
		}
		if o.L2Plat != nil {
			p2 = strings.ToUpper(strings.TrimSpace(*o.L2Plat))
		}
		return []legSpec{
			{token: t1, platform: p1, fee: fees[p1]},
			{token: t2, platform: p2, fee: fees[p2]},
		}, o.OppKind, nil
	default:
		return nil, o.OppKind, fmt.Errorf("unsupported kind %s", o.OppKind)
	}
}

func filterSnapsGE(snaps []int64, minMs int64) []int64 {
	var out []int64
	for _, ms := range snaps {
		if ms >= minMs {
			out = append(out, ms)
		}
	}
	return out
}

func nearestSnap(snaps []int64, target int64) int64 {
	if len(snaps) == 0 {
		return target
	}
	best := snaps[0]
	bd := abs64(best - target)
	for _, ms := range snaps[1:] {
		d := abs64(ms - target)
		if d < bd {
			best, bd = ms, d
		}
	}
	return best
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func restSnapTimes(db *gorm.DB, table, mapKey string, fromMs int64, maxN int, horizonMs int64) ([]int64, error) {
	var snaps []int64
	q := db.Table(table).Where("map_key=? AND snapshot_at_ms>=?", mapKey, fromMs)
	if horizonMs > 0 {
		q = q.Where("snapshot_at_ms <= ?", fromMs+horizonMs)
	}
	var raw []int64
	err := q.Select("snapshot_at_ms").Order("snapshot_at_ms ASC").Limit(maxN * 5).Scan(&raw).Error
	if err != nil {
		return nil, err
	}
	var last int64 = -1
	for _, ms := range raw {
		if ms == last {
			continue
		}
		snaps = append(snaps, ms)
		last = ms
		if len(snaps) >= maxN {
			break
		}
	}
	return snaps, nil
}

func replayPerCached(db *gorm.DB, table, mapKey string, legs []legSpec, snapMs int64, cache map[int64]map[string]string) (float64, bool) {
	asksByTok, err := loadAsksBatch(db, table, mapKey, legs, snapMs, cache)
	if err != nil || len(asksByTok) < len(legs) {
		return 0, false
	}
	var sum float64
	for _, leg := range legs {
		raw, ok := asksByTok[leg.token]
		if !ok {
			return 0, false
		}
		px, sz := parseBestAsk(raw)
		if !validPrice(px) || sz <= 0 {
			return 0, false
		}
		sum += px + tradeFee(px, leg.fee)
	}
	return sum, true
}

func loadAsksBatch(db *gorm.DB, table, mapKey string, legs []legSpec, snapMs int64, cache map[int64]map[string]string) (map[string]string, error) {
	if c, ok := cache[snapMs]; ok && len(c) >= len(legs) {
		return c, nil
	}
	tokens := make([]string, len(legs))
	for i, l := range legs {
		tokens[i] = l.token
	}
	type R struct {
		Token string `gorm:"column:token_id"`
		Asks  string `gorm:"column:order_book_asks"`
	}
	var rows []R
	q := db.Table(table).Select("token_id, order_book_asks").Where("snapshot_at_ms=? AND token_id IN ?", snapMs, tokens)
	if mapKey != "" {
		q = q.Where("map_key=?", mapKey)
	}
	err := q.Scan(&rows).Error
	out := map[string]string{}
	for _, r := range rows {
		if strings.TrimSpace(r.Asks) != "" {
			out[r.Token] = r.Asks
		}
	}
	if len(out) < len(legs) && mapKey != "" {
		_ = db.Table(table).Select("token_id, order_book_asks").
			Where("snapshot_at_ms=? AND token_id IN ?", snapMs, tokens).Scan(&rows)
		for _, r := range rows {
			if strings.TrimSpace(r.Asks) != "" {
				out[r.Token] = r.Asks
			}
		}
	}
	if len(out) < len(legs) {
		for _, leg := range legs {
			if out[leg.token] != "" {
				continue
			}
			var one R
			q1 := db.Table(table).Select("token_id, order_book_asks").
				Where("token_id=? AND snapshot_at_ms<=?", leg.token, snapMs)
			if mapKey != "" {
				q1 = q1.Where("map_key=?", mapKey)
			}
			e := q1.Order("snapshot_at_ms DESC").Limit(1).Scan(&one).Error
			if e == nil && strings.TrimSpace(one.Asks) != "" {
				out[leg.token] = one.Asks
			} else {
				e2 := db.Table(table).Select("token_id, order_book_asks").
					Where("token_id=? AND snapshot_at_ms<=?", leg.token, snapMs).
					Order("snapshot_at_ms DESC").Limit(1).Scan(&one).Error
				if e2 == nil && strings.TrimSpace(one.Asks) != "" {
					out[leg.token] = one.Asks
				}
			}
		}
	}
	if cache != nil {
		cache[snapMs] = out
	}
	return out, err
}

func printDurationLine(o oppRow, kind string, th float64, reproMs int64, after []int64, lastStill int64, dur float64, open bool, flags []string, firstPer float64, repro bool) {
	mk := ""
	if o.MapKey != nil {
		mk = *o.MapKey
	}
	fmt.Printf("\n#%d %s map=%s snap=%d 行内cost=%.6f th=%.4f\n",
		o.ID, kind, mk, o.SnapMs, decF(o.TotalCost), th)
	if len(flags) > 0 {
		fmt.Printf("  自洽: %s\n", strings.Join(flags, "; "))
	}
	if repro {
		fmt.Printf("  最近 REST 批 ms=%d per=%.6f Δ=%.4f\n", reproMs, firstPer, firstPer-decF(o.TotalCost))
	} else {
		fmt.Println("  最近 REST 批: 无完整三腿簿")
	}
	if lastStill <= 0 {
		fmt.Printf("  命中后 REST 帧 %d 个：未观察到仍<%v 的持续段\n", len(after), th)
		return
	}
	fmt.Printf("  命中后 REST 帧 %d 个 → 持续至 ms=%d 约 %.0fms%s\n",
		len(after), lastStill, dur, map[bool]string{true: "（末帧仍满足）", false: ""}[open])
}

func pct(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := int(math.Round(q * float64(len(sorted)-1)))
	if i < 0 {
		i = 0
	}
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return sorted[i]
}

// --- 与 audit_onextwo_opp 对齐的简版自洽 ---

func decF(d decimal.Decimal) float64 {
	v, _ := d.Float64()
	return v
}

func askF(p *decimal.Decimal) float64 {
	if p == nil {
		return 0
	}
	return decF(*p)
}

func internalCheck(o oppRow) []string {
	tc := decF(o.TotalCost)
	ps := decF(o.ProfitSpace)
	if askF(o.L1Ask) <= 0 {
		return []string{"legacy_or_incomplete_row (无 leg_* 列数据，跳过细项)"}
	}
	if o.OppKind == binaryYN && askF(o.L2Ask) <= 0 {
		return []string{"legacy_or_incomplete_row (无 leg_* 列数据，跳过细项)"}
	}
	if strings.Contains(o.OppKind, "ONE_X2") && (askF(o.L2Ask) <= 0 || askF(o.L3Ask) <= 0) {
		return []string{"legacy_or_incomplete_row (无 leg_* 列数据，跳过细项)"}
	}
	var f []string
	base := 1.0
	if strings.Contains(o.OppKind, "NO_TRIPLE") {
		base = 2
	} else if o.OppKind == binaryYN {
		base = 1
	}
	if math.Abs(ps-(base-tc)) > 0.0003 {
		f = append(f, fmt.Sprintf("profit_space vs base-cost Δ%.6f", ps-(base-tc)))
	}
	aq, ap := decF(o.ArbQty), decF(o.ArbProfit)
	if aq > 0 && math.Abs(ap-aq*ps) > 0.0003 {
		f = append(f, "arb_profit!=qty*profit_space")
	}
	return f
}

func tokenForLeg(idx int, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	switch idx {
	case 0:
		var d struct {
			DrawTokenID string `json:"draw_token_id"`
		}
		if json.Unmarshal([]byte(raw), &d) == nil {
			return strings.TrimSpace(d.DrawTokenID)
		}
	case 1:
		var y struct {
			YesTokenID string `json:"yes_token_id"`
		}
		if json.Unmarshal([]byte(raw), &y) == nil {
			return strings.TrimSpace(y.YesTokenID)
		}
	case 2:
		var n struct {
			NoTokenID string `json:"no_token_id"`
		}
		if json.Unmarshal([]byte(raw), &n) == nil {
			return strings.TrimSpace(n.NoTokenID)
		}
	}
	return ""
}

func tokenFromBinary(raw string, yes bool) string {
	raw = strings.TrimSpace(raw)
	if yes {
		var y struct {
			YesTokenID string `json:"yes_token_id"`
		}
		if json.Unmarshal([]byte(raw), &y) == nil {
			return strings.TrimSpace(y.YesTokenID)
		}
	}
	var n struct {
		NoTokenID string `json:"no_token_id"`
	}
	if json.Unmarshal([]byte(raw), &n) == nil {
		return strings.TrimSpace(n.NoTokenID)
	}
	return ""
}

func defaultFees() map[string]float64 {
	return map[string]float64{
		"POLYMARKET": 0.05,
		"KALSHI":     0.07,
		"PREDICT":    0.02,
		"SXBET":      0.02,
	}
}

func validPrice(p float64) bool {
	return p > 0 && p < 1 && !math.IsNaN(p) && !math.IsInf(p, 0)
}

func tradeFee(p, coef float64) float64 {
	if coef <= 0 || !validPrice(p) {
		return 0
	}
	return coef * p * (1 - p)
}

func parseBestAsk(raw string) (float64, float64) {
	var rows [][]string
	if json.Unmarshal([]byte(raw), &rows) == nil {
		bestP, bestS := 0.0, 0.0
		for _, r := range rows {
			if len(r) < 2 {
				continue
			}
			p, _ := strconv.ParseFloat(r[0], 64)
			s, _ := strconv.ParseFloat(r[1], 64)
			if book.ValidPrice(p) && s > 0 && (bestP <= 0 || p < bestP) {
				bestP, bestS = p, s
			}
		}
		return bestP, bestS
	}
	var ls []book.Level
	if json.Unmarshal([]byte(raw), &ls) == nil {
		for _, lv := range ls {
			if book.ValidPrice(lv.Price) && lv.Size > 0 {
				return lv.Price, lv.Size
			}
		}
	}
	return 0, 0
}