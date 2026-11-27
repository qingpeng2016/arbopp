// 分析实盘双边成交：arb_binary_opportunity + orders + market_order_book_ws。
// 重点：哪边 FILLED / NO_FILL、成交相对限价滑点、发单前后 WS 卖一是否还在。
//
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/analyze_fill_race
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/analyze_fill_race -hours=12 -n=30
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/analyze_fill_race -map_key='v1|NFL|nfl-kc-buf-2026-11-27'
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
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"gorm.io/gorm"
)

type orderRow struct {
	ID             uint      `gorm:"column:id"`
	ArbOppID       uint      `gorm:"column:arb_opp_id"`
	MapKey         string    `gorm:"column:map_key"`
	SideNo         uint8     `gorm:"column:side_no"`
	Platform       string    `gorm:"column:platform"`
	TokenID        string    `gorm:"column:token_id"`
	OutcomeID      string    `gorm:"column:platform_market_outcome_id"`
	Side           string    `gorm:"column:side"`
	Price          float64   `gorm:"column:price"`
	Qty            float64   `gorm:"column:qty"`
	TradePrice     float64   `gorm:"column:trade_price"`
	TradeQty       float64   `gorm:"column:trade_qty"`
	TradeAmount    float64   `gorm:"column:trade_amount"`
	Status         string    `gorm:"column:status"`
	ErrMsg         string    `gorm:"column:err_msg"`
	SentAt         time.Time `gorm:"column:sent_at"`
	PlatformOID    string    `gorm:"column:platform_order_id"`
	OppKind        string    `gorm:"column:opp_kind"`
	OppSnapMs      int64     `gorm:"column:snapshot_at_ms"`
	OppExt         *string   `gorm:"column:ext_data"`
	L1Plat         *string   `gorm:"column:leg_1_platform"`
	L2Plat         *string   `gorm:"column:leg_2_platform"`
	L1Ask          *float64  `gorm:"column:leg_1_best_ask"`
	L2Ask          *float64  `gorm:"column:leg_2_best_ask"`
	ArbProfitAmt   *float64  `gorm:"column:arb_profit_amount"`
}

type oppVerdict struct {
	OppID     uint
	MapKey    string
	Kind      string
	SentAt    time.Time
	SnapMs    int64
	ExtSkip   string
	ExtOrdered bool
	Legs      []orderRow
	Label     string // BOTH_FILLED | ONE_SIDED | NO_FILL | MIXED
	Filled    []string
	Empty     []string
}

func main() {
	hours := flag.Float64("hours", 24, "回看小时")
	limit := flag.Int("n", 40, "最多分析多少组有单的机会")
	mapKey := flag.String("map_key", "", "只看该 map_key")
	winMs := flag.Int64("window", 3000, "相对 sent_at / snapshot_at_ms 查 WS ±ms")
	verbose := flag.Int("v", 15, "明细打印条数")
	flag.Parse()

	_ = conf.NewCfg()
	db := boot.NewDBClient()
	if db == nil {
		fmt.Println("db nil")
		os.Exit(1)
	}

	since := time.Now().UTC().Add(-time.Duration(*hours * float64(time.Hour)))
	fmt.Printf("===== analyze_fill_race =====\nsince=%s hours=%.1f map_key=%q\n\n", since.Format(time.RFC3339), *hours, *mapKey)

	printOrderStats(db, since, *mapKey)
	opps := loadOrderedOpps(db, since, *mapKey, *limit)
	if len(opps) == 0 {
		fmt.Println("无带 orders 的机会")
		os.Exit(0)
	}

	var both, one, none, mixed int
	platFill := map[string]int{}
	platEmpty := map[string]int{}
	var oneSided []oppVerdict

	for i := range opps {
		v := classify(&opps[i])
		switch v.Label {
		case "BOTH_FILLED":
			both++
		case "ONE_SIDED":
			one++
			oneSided = append(oneSided, v)
		case "NO_FILL":
			none++
		default:
			mixed++
		}
		for _, p := range v.Filled {
			platFill[p]++
		}
		for _, p := range v.Empty {
			platEmpty[p]++
		}
	}

	fmt.Println("=== 有单机会判定（按 arb_opp_id）===")
	fmt.Printf("  groups=%d  BOTH_FILLED=%d  ONE_SIDED=%d  NO_FILL=%d  OTHER=%d\n", len(opps), both, one, none, mixed)
	fmt.Println("  腿成交平台计数（组内 FILLED/部分成交）:")
	for _, p := range sortedKeys(platFill) {
		fmt.Printf("    fill %-12s %d\n", p, platFill[p])
	}
	fmt.Println("  腿空成平台计数（NO_FILL / trade_qty=0）:")
	for _, p := range sortedKeys(platEmpty) {
		fmt.Printf("    empty %-12s %d\n", p, platEmpty[p])
	}

	fmt.Println("\n=== Predict 成交滑点（trade_price - limit）===")
	printSlipStats(flattenOrders(opps), "PREDICT")
	fmt.Println("=== Polymarket 成交滑点 ===")
	printSlipStats(flattenOrders(opps), "POLYMARKET")

	show := *verbose
	if show > len(opps) {
		show = len(opps)
	}
	fmt.Printf("\n=== 明细（最近 %d 组）===\n", show)
	for i := 0; i < show; i++ {
		v := classify(&opps[i])
		printOppDetail(db, v, *winMs)
	}

	if len(oneSided) > 0 {
		fmt.Printf("\n=== 单边摘要（%d）===\n", len(oneSided))
		predOnly, polyOnly := 0, 0
		for _, v := range oneSided {
			f := strings.Join(v.Filled, ",")
			e := strings.Join(v.Empty, ",")
			fmt.Printf("  opp=%d %s filled=[%s] empty=[%s] at=%s\n",
				v.OppID, v.MapKey, f, e, v.SentAt.UTC().Format("15:04:05"))
			if strings.Contains(f, "PREDICT") && strings.Contains(e, "POLYMARKET") {
				predOnly++
			}
			if strings.Contains(f, "POLYMARKET") && strings.Contains(e, "PREDICT") {
				polyOnly++
			}
		}
		fmt.Printf("  → 仅 Predict 成 / Poly 空: %d\n", predOnly)
		fmt.Printf("  → 仅 Poly 成 / Predict 空: %d\n", polyOnly)
	}
}

func printOrderStats(db *gorm.DB, since time.Time, mapKey string) {
	type row struct {
		Platform string
		Status   string
		N        int64
		FillN    int64
	}
	q := db.Table("orders").
		Select("platform, status, COUNT(*) n, SUM(CASE WHEN trade_qty>0 THEN 1 ELSE 0 END) fill_n").
		Where("backtest_task_id = 0 AND sent_at >= ?", since)
	if mapKey != "" {
		q = q.Where("map_key = ?", mapKey)
	}
	var rows []row
	_ = q.Group("platform, status").Order("platform, status").Find(&rows)
	fmt.Println("=== orders 按平台×状态 ===")
	if len(rows) == 0 {
		fmt.Println("  (无)")
		return
	}
	for _, r := range rows {
		fmt.Printf("  %-12s %-16s n=%d fill_qty>0=%d\n", r.Platform, r.Status, r.N, r.FillN)
	}
	fmt.Println()
}

func loadOrderedOpps(db *gorm.DB, since time.Time, mapKey string, limit int) []oppVerdict {
	var raw []orderRow
	q := db.Table("orders AS o").
		Select(`o.id, o.arb_opp_id, o.map_key, o.side_no, o.platform, o.token_id, o.platform_market_outcome_id,
			o.side, o.price, o.qty, o.trade_price, o.trade_qty, o.trade_amount, o.status, o.err_msg, o.sent_at, o.platform_order_id,
			a.opp_kind, a.snapshot_at_ms, a.ext_data, a.leg_1_platform, a.leg_2_platform,
			a.leg_1_best_ask, a.leg_2_best_ask, a.arb_profit_amount`).
		Joins("JOIN arb_binary_opportunity a ON a.id = o.arb_opp_id").
		Where("o.backtest_task_id = 0 AND o.sent_at >= ?", since)
	if mapKey != "" {
		q = q.Where("o.map_key = ?", mapKey)
	}
	_ = q.Order("o.sent_at DESC, o.id DESC").Limit(limit * 6).Find(&raw)

	byOpp := map[uint]*oppVerdict{}
	order := make([]uint, 0)
	for _, r := range raw {
		v, ok := byOpp[r.ArbOppID]
		if !ok {
			v = &oppVerdict{
				OppID:  r.ArbOppID,
				MapKey: r.MapKey,
				Kind:   r.OppKind,
				SentAt: r.SentAt,
				SnapMs: r.OppSnapMs,
			}
			if r.OppExt != nil {
				v.ExtSkip, v.ExtOrdered = parseExt(*r.OppExt)
			}
			byOpp[r.ArbOppID] = v
			order = append(order, r.ArbOppID)
		}
		v.Legs = append(v.Legs, r)
		if r.SentAt.Before(v.SentAt) {
			v.SentAt = r.SentAt
		}
	}
	out := make([]oppVerdict, 0, len(order))
	for _, id := range order {
		out = append(out, *byOpp[id])
		if len(out) >= limit {
			break
		}
	}
	return out
}

func parseExt(raw string) (skip string, ordered bool) {
	var m map[string]any
	if json.Unmarshal([]byte(raw), &m) != nil {
		return "", false
	}
	if s, ok := m["skip_order_reason"].(string); ok {
		skip = s
	}
	if b, ok := m["ordered"].(bool); ok {
		ordered = b
	}
	return skip, ordered
}

func classify(v *oppVerdict) oppVerdict {
	filled, empty := []string{}, []string{}
	fillN, emptyN := 0, 0
	for _, leg := range v.Legs {
		p := strings.ToUpper(leg.Platform)
		if legFilled(leg) {
			fillN++
			filled = append(filled, p)
		} else {
			emptyN++
			empty = append(empty, p)
		}
	}
	switch {
	case fillN >= 2 && emptyN == 0:
		v.Label = "BOTH_FILLED"
	case fillN >= 1 && emptyN >= 1:
		v.Label = "ONE_SIDED"
	case fillN == 0:
		v.Label = "NO_FILL"
	default:
		v.Label = "MIXED"
	}
	v.Filled, v.Empty = unique(filled), unique(empty)
	return *v
}

func legFilled(o orderRow) bool {
	if o.TradeQty > 1e-9 {
		return true
	}
	st := strings.ToUpper(o.Status)
	return st == "FILLED" || st == "PARTIAL" || st == "MATCHED"
}

func printOppDetail(db *gorm.DB, v oppVerdict, winMs int64) {
	fmt.Printf("\n#opp=%d %s kind=%s label=%s sent=%s snap_ms=%d ordered=%v skip=%q\n",
		v.OppID, v.MapKey, v.Kind, v.Label, v.SentAt.UTC().Format(time.RFC3339), v.SnapMs, v.ExtOrdered, v.ExtSkip)
	for _, leg := range v.Legs {
		slip := 0.0
		if leg.TradeQty > 0 && leg.Price > 0 {
			slip = leg.TradePrice - leg.Price
		}
		fmt.Printf("  leg%d %-11s %s limit=%.4f qty=%.4f → trade=%.4f/%.4f amt=%.4f status=%s oid=%s\n",
			leg.SideNo, leg.Platform, leg.Side, leg.Price, leg.Qty, leg.TradePrice, leg.TradeQty, leg.TradeAmount, leg.Status, shortOID(leg.PlatformOID))
		if leg.TradeQty > 0 {
			fmt.Printf("       slip=%+.4f (trade-limit)\n", slip)
		}
		printWSAround(db, leg, v.SnapMs, winMs)
	}
}

func printWSAround(db *gorm.DB, leg orderRow, snapMs, winMs int64) {
	anchor := snapMs
	if anchor <= 0 {
		anchor = leg.SentAt.UTC().UnixMilli()
	}
	lo, hi := anchor-winMs, anchor+winMs

	type wsRow struct {
		ID       int64  `gorm:"column:id"`
		Platform string `gorm:"column:platform"`
		TokenID  string `gorm:"column:token_id"`
		SnapMs   int64  `gorm:"column:snapshot_at_ms"`
		Asks     string `gorm:"column:order_book_asks"`
		Outcome  string `gorm:"column:platform_market_outcome_id"`
	}
	var rows []wsRow
	q := db.Table("market_order_book_ws").
		Select("id, platform, token_id, snapshot_at_ms, order_book_asks, platform_market_outcome_id").
		Where("platform = ? AND snapshot_at_ms BETWEEN ? AND ?", leg.Platform, lo, hi)
	tok := strings.TrimSpace(leg.TokenID)
	outID := strings.TrimSpace(leg.OutcomeID)
	switch {
	case tok != "":
		q = q.Where("token_id = ?", tok)
	case outID != "":
		q = q.Where("platform_market_outcome_id = ?", outID)
	default:
		fmt.Printf("       ws: (no token/outcome id)\n")
		return
	}
	_ = q.Order(fmt.Sprintf("ABS(snapshot_at_ms - %d)", anchor)).Limit(4).Find(&rows)
	if len(rows) == 0 {
		fmt.Printf("       ws: none in ±%dms around %d\n", winMs, anchor)
		return
	}
	for _, r := range rows {
		top, topSz := wsTopAskSize(r.Asks)
		d := r.SnapMs - anchor
		vsLimit := ""
		if top > 0 && leg.Price > 0 {
			vsLimit = fmt.Sprintf(" Δlimit=%+.4f", top-leg.Price)
			if top > leg.Price+1e-9 {
				vsLimit += " ABOVE_LIMIT"
			} else if topSz+1e-9 < leg.Qty {
				vsLimit += " THIN"
			} else {
				vsLimit += " ok"
			}
		}
		fmt.Printf("       ws id=%d Δms=%+d top=%.4f sz=%.2f%s\n", r.ID, d, top, topSz, vsLimit)
	}
}

func wsTopAskSize(raw string) (px, sz float64) {
	var rows [][]string
	if json.Unmarshal([]byte(raw), &rows) != nil {
		return 0, 0
	}
	for _, r := range rows {
		if len(r) < 2 {
			continue
		}
		p, _ := strconv.ParseFloat(r[0], 64)
		s, _ := strconv.ParseFloat(r[1], 64)
		if book.ValidPrice(p) && s > 0 {
			return p, s
		}
	}
	return 0, 0
}

func printSlipStats(orders []orderRow, platform string) {
	var slips []float64
	var nFill, nEmpty int
	for _, o := range orders {
		if !strings.EqualFold(o.Platform, platform) {
			continue
		}
		if !legFilled(o) {
			nEmpty++
			continue
		}
		nFill++
		if o.Price > 0 {
			slips = append(slips, o.TradePrice-o.Price)
		}
	}
	fmt.Printf("  %s fill=%d empty=%d", platform, nFill, nEmpty)
	if len(slips) == 0 {
		fmt.Println()
		return
	}
	sort.Float64s(slips)
	sum := 0.0
	for _, s := range slips {
		sum += s
	}
	fmt.Printf("  slip min=%+.4f p50=%+.4f max=%+.4f avg=%+.4f\n",
		slips[0], slips[len(slips)/2], slips[len(slips)-1], sum/float64(len(slips)))
}

func flattenOrders(opps []oppVerdict) []orderRow {
	var out []orderRow
	for _, o := range opps {
		out = append(out, o.Legs...)
	}
	return out
}

func unique(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func shortOID(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 14 {
		return s
	}
	return s[:10] + "…"
}

func abs(x float64) float64 { return math.Abs(x) }
