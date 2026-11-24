// 审计 ONE_X2_YES_TRIPLE / ONE_X2_NO_TRIPLE：行内算术、腿 JSON、market_order_book_ws 与 market_order_book 同窗顶价。
//
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/audit_onextwo_opp -n=50
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/audit_onextwo_opp -id=2066 -v
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/audit_onextwo_opp -order=snap -n=30
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
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
)

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

type legBooks struct {
	NetCost float64 `json:"net_cost"`
	Books   []struct {
		Asks []book.Level `json:"asks"`
	} `json:"books"`
}

type drawLegInfo struct {
	DrawMarketID string `json:"draw_market_id"`
	DrawTokenID  string `json:"draw_token_id"`
	DrawOutcomeID uint  `json:"draw_outcome_id"`
}
type yesLegInfo struct {
	YesMarketID  string `json:"yes_market_id"`
	YesTokenID   string `json:"yes_token_id"`
	YesOutcomeID uint   `json:"yes_outcome_id"`
}
type noLegInfo struct {
	NoMarketID  string `json:"no_market_id"`
	NoTokenID   string `json:"no_token_id"`
	NoOutcomeID uint   `json:"no_outcome_id"`
}

func main() {
	limit := flag.Int("n", 50, "max opportunities (newest first)")
	oppID := flag.Uint("id", 0, "audit single id (overrides -n)")
	winMs := flag.Int64("window", 8000, "±ms around snapshot_at_ms for WS/REST")
	verbose := flag.Bool("v", false, "print every row with issues")
	sample := flag.Int("sample", 8, "detail lines for newest problematic rows")
	orderBy := flag.String("order", "id", "id=按主键倒序; snap=仅 snap>0 且三腿 ask 非空，按 snapshot_at_ms 倒序")
	flag.Parse()

	conf.NewCfg()
	db := boot.NewDBClient()

	var yesCnt, noCnt int64
	db.Table("arb_binary_opportunity").Where("opp_kind = ?", yesTriple).Count(&yesCnt)
	db.Table("arb_binary_opportunity").Where("opp_kind = ?", noTriple).Count(&noCnt)
	fmt.Printf("ONE_X2 rows: %s=%d %s=%d\n\n", yesTriple, yesCnt, noTriple, noCnt)

	q := db.Table("arb_binary_opportunity").Where("opp_kind IN ?", []string{yesTriple, noTriple}).
		Select(`id, opp_kind, map_key, threshold_key, total_cost, profit_space, arb_qty, arb_profit,
			threshold_value, snapshot_at_ms,
			leg_1_platform, leg_2_platform, leg_3_platform,
			leg_1_best_ask, leg_2_best_ask, leg_3_best_ask,
			leg_1_order_books, leg_2_order_books, leg_3_order_books,
			leg_1_info, leg_2_info, leg_3_info`)

	if *oppID > 0 {
		q = q.Where("id = ?", *oppID)
	} else if strings.EqualFold(*orderBy, "snap") {
		q = q.Where("snapshot_at_ms > 0 AND leg_1_best_ask IS NOT NULL AND leg_2_best_ask IS NOT NULL AND leg_3_best_ask IS NOT NULL").
			Order("snapshot_at_ms DESC").Limit(*limit)
	} else {
		q = q.Order("id DESC").Limit(*limit)
	}

	var rows []oppRow
	if err := q.Scan(&rows).Error; err != nil {
		fmt.Println("query:", err)
		os.Exit(1)
	}
	if len(rows) == 0 {
		fmt.Println("no ONE_X2 rows matched")
		os.Exit(0)
	}

	type tallies struct {
		internalOK, internalBad int
		thPass, thFail          int
		wsLegs, wsMatch         int
		restLegs, restMatch     int
		restMapHit              int
	}
	var t tallies
	var samples []string

	for i := len(rows) - 1; i >= 0; i-- {
		o := rows[i]
		flags := internalCheck(o)
		if len(flags) == 0 {
			t.internalOK++
		} else {
			t.internalBad++
		}
		tc, _ := o.TotalCost.Float64()
		th, _ := o.Threshold.Float64()
		if th > 0 && tc < th {
			t.thPass++
		} else if th > 0 {
			t.thFail++
		}

		wsM, wsH, restM, restH := corroborateLegs(db, o, *winMs)
		t.wsLegs += wsH
		t.wsMatch += wsM
		t.restLegs += restH
		t.restMatch += restM
		if hasRESTMap(db, o, *winMs) {
			t.restMapHit++
		}

		if len(flags) > 0 {
			line := fmt.Sprintf("#%d %s snap=%d flags: %s", o.ID, o.OppKind, o.SnapMs, strings.Join(flags, "; "))
			if len(samples) < *sample {
				samples = append(samples, line)
			}
			if *verbose || (*oppID > 0 && o.ID == uint(*oppID)) {
				fmt.Println(line)
				printLegDetail(db, o, *winMs)
			}
		}
	}

	fmt.Printf("=== SUMMARY (checked %d ONE_X2 opps) ===\n", len(rows))
	fmt.Printf("内部自洽: OK=%d BAD=%d\n", t.internalOK, t.internalBad)
	fmt.Printf("total_cost < threshold: pass=%d fail=%d (TEST 允许 profit_space<=0)\n", t.thPass, t.thFail)
	fmt.Printf("WS 腿: 有窗前数据 %d; 顶价±1%%对齐 %d\n", t.wsLegs, t.wsMatch)
	fmt.Printf("REST(token) 腿: 有窗前数据 %d; 顶价±1%%对齐 %d\n", t.restLegs, t.restMatch)
	fmt.Printf("REST(map_key) 同窗有行: %d/%d\n", t.restMapHit, len(rows))
	fmt.Println("\n策略对齐说明:")
	fmt.Println("  YES_TRIPLE: profit_space = 1 - total_cost; 阈值键", onextwo.ThresholdKeyYesTriple)
	fmt.Println("  NO_TRIPLE:  profit_space = 2 - total_cost; 阈值键", onextwo.ThresholdKeyNoTriple)
	fmt.Println("  total_cost = 三腿 net_cost 之和; leg_*_best_ask = 卖一裸价(不含费)")
	if len(samples) > 0 {
		fmt.Println("\n样例问题行:")
		for _, s := range samples {
			fmt.Println(" ", s)
		}
	}
	if *oppID > 0 && len(rows) == 1 {
		fmt.Println()
		printLegDetail(db, rows[0], *winMs)
	}
}

func profitBase(kind string) float64 {
	if kind == noTriple {
		return 2
	}
	return 1
}

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
	var f []string
	tc := decF(o.TotalCost)
	ps := decF(o.ProfitSpace)
	aq := decF(o.ArbQty)
	ap := decF(o.ArbProfit)
	th := decF(o.Threshold)

	if tc > 0 && askF(o.L1Ask) <= 0 && askF(o.L2Ask) <= 0 && askF(o.L3Ask) <= 0 {
		return []string{"legacy_or_incomplete_row (无 leg_* 列数据，跳过细项)"}
	}

	base := profitBase(o.OppKind)
	if math.Abs(ps-(base-tc)) > 0.0002 {
		f = append(f, fmt.Sprintf("profit_space!=%.0f-total_cost (%.8f+%.8f)", base, ps, tc))
	}
	if aq > 0 && math.Abs(ap-aq*ps) > 0.0002 {
		f = append(f, "arb_profit!=arb_qty*profit_space")
	}
	if th > 0 && tc >= th {
		f = append(f, fmt.Sprintf("total_cost>=threshold (%.8f>=%.8f)", tc, th))
	}
	wantTh := onextwo.ThresholdKeyYesTriple
	if o.OppKind == noTriple {
		wantTh = onextwo.ThresholdKeyNoTriple
	}
	if strings.TrimSpace(o.ThresholdKey) != "" && o.ThresholdKey != wantTh {
		f = append(f, "threshold_key="+o.ThresholdKey+" want "+wantTh)
	}

	asks := []*decimal.Decimal{o.L1Ask, o.L2Ask, o.L3Ask}
	books := []string{o.L1Books, o.L2Books, o.L3Books}
	infos := []string{o.L1Info, o.L2Info, o.L3Info}
	plats := []*string{o.L1Plat, o.L2Plat, o.L3Plat}

	var netSum, nakedSum float64
	legNets := 0
	for i := 0; i < 3; i++ {
		av := askF(asks[i])
		if av <= 0 {
			f = append(f, fmt.Sprintf("leg%d_best_ask missing", i+1))
			continue
		}
		top := legTopAsk(books[i])
		if top > 0 && math.Abs(av-top) > 0.0002 {
			f = append(f, fmt.Sprintf("leg%d_best_ask!=json top (%.8f vs %.8f)", i+1, av, top))
		}
		nc := legNetCost(books[i])
		if nc > 0 {
			netSum += nc
			legNets++
		}
		nakedSum += av
		tok := tokenForLeg(i, infos[i])
		if tok == "" {
			f = append(f, fmt.Sprintf("leg%d_info token missing", i+1))
		}
	}
	if legNets == 3 && math.Abs(netSum-tc) > 0.0003 {
		f = append(f, fmt.Sprintf("total_cost vs sum(net_cost) Δ=%.6f", tc-netSum))
	}
	if nakedSum > 0 && nakedSum > tc+0.0001 {
		f = append(f, fmt.Sprintf("sum裸卖一 %.6f > total_cost (fee ok if small)", nakedSum))
	}

	pSet := map[string]struct{}{}
	for i, p := range plats {
		if p == nil || strings.TrimSpace(*p) == "" {
			f = append(f, fmt.Sprintf("leg%d_platform empty", i+1))
			continue
		}
		up := strings.ToUpper(strings.TrimSpace(*p))
		if _, dup := pSet[up]; dup {
			f = append(f, "duplicate platform "+up)
		}
		pSet[up] = struct{}{}
	}

	// outcome_id 不应重复（平局/主/客三路）
	oids := outcomeIDs(infos)
	if len(oids) == 3 && len(uniqueUint(oids)) < 3 {
		f = append(f, "duplicate outcome_id across legs")
	}

	minSz := minTopSizeFrom(books)
	if minSz > 0 && aq > minSz+1e-6 {
		f = append(f, fmt.Sprintf("arb_qty %.4f > min top ask size %.4f", aq, minSz))
	}
	return f
}

func legNetCost(raw string) float64 {
	var p legBooks
	if json.Unmarshal([]byte(strings.TrimSpace(raw)), &p) != nil {
		return 0
	}
	return p.NetCost
}

func legTopAsk(raw string) float64 {
	var p legBooks
	if json.Unmarshal([]byte(strings.TrimSpace(raw)), &p) != nil {
		return 0
	}
	for _, b := range p.Books {
		for _, a := range b.Asks {
			if book.ValidPrice(a.Price) && a.Size > 0 {
				return a.Price
			}
		}
	}
	return 0
}

func minTopSizeFrom(rawBooks []string) float64 {
	min := 0.0
	for _, raw := range rawBooks {
		var p legBooks
		if json.Unmarshal([]byte(strings.TrimSpace(raw)), &p) != nil {
			continue
		}
		sz := 0.0
		for _, b := range p.Books {
			for _, a := range b.Asks {
				if book.ValidPrice(a.Price) && a.Size > 0 {
					sz = a.Size
					break
				}
			}
			if sz > 0 {
				break
			}
		}
		if sz <= 0 {
			continue
		}
		if min <= 0 || sz < min {
			min = sz
		}
	}
	return min
}

func tokenForLeg(idx int, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	switch idx {
	case 0:
		var d drawLegInfo
		if json.Unmarshal([]byte(raw), &d) == nil {
			return strings.TrimSpace(d.DrawTokenID)
		}
	case 1:
		var y yesLegInfo
		if json.Unmarshal([]byte(raw), &y) == nil {
			return strings.TrimSpace(y.YesTokenID)
		}
	case 2:
		var n noLegInfo
		if json.Unmarshal([]byte(raw), &n) == nil {
			return strings.TrimSpace(n.NoTokenID)
		}
	}
	return ""
}

func marketIDForLeg(idx int, raw string) string {
	switch idx {
	case 0:
		var d drawLegInfo
		if json.Unmarshal([]byte(raw), &d) == nil {
			return strings.TrimSpace(d.DrawMarketID)
		}
	case 1:
		var y yesLegInfo
		if json.Unmarshal([]byte(raw), &y) == nil {
			return strings.TrimSpace(y.YesMarketID)
		}
	case 2:
		var n noLegInfo
		if json.Unmarshal([]byte(raw), &n) == nil {
			return strings.TrimSpace(n.NoMarketID)
		}
	}
	return ""
}

func outcomeIDs(infos []string) []uint {
	var out []uint
	for i, raw := range infos {
		switch i {
		case 0:
			var d drawLegInfo
			if json.Unmarshal([]byte(raw), &d) == nil && d.DrawOutcomeID > 0 {
				out = append(out, d.DrawOutcomeID)
			}
		case 1:
			var y yesLegInfo
			if json.Unmarshal([]byte(raw), &y) == nil && y.YesOutcomeID > 0 {
				out = append(out, y.YesOutcomeID)
			}
		case 2:
			var n noLegInfo
			if json.Unmarshal([]byte(raw), &n) == nil && n.NoOutcomeID > 0 {
				out = append(out, n.NoOutcomeID)
			}
		}
	}
	return out
}

func uniqueUint(in []uint) []uint {
	seen := map[uint]struct{}{}
	var out []uint
	for _, x := range in {
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func corroborateLegs(db *gorm.DB, o oppRow, winMs int64) (wsMatch, wsHas, restMatch, restHas int) {
	if o.SnapMs <= 0 {
		return
	}
	lo, hi := o.SnapMs-winMs, o.SnapMs+winMs
	infos := []string{o.L1Info, o.L2Info, o.L3Info}
	asks := []*decimal.Decimal{o.L1Ask, o.L2Ask, o.L3Ask}
	books := []string{o.L1Books, o.L2Books, o.L3Books}
	for i := 0; i < 3; i++ {
		want := askF(asks[i])
		if want <= 0 {
			want = legTopAsk(books[i])
		}
		if want <= 0 {
			continue
		}
		tok := tokenForLeg(i, infos[i])
		if tok == "" {
			continue
		}
		if m, h := bookTopInWindow(db, "market_order_book_ws", tok, lo, hi, o.SnapMs, want); h {
			wsHas++
			if m {
				wsMatch++
			}
		}
		if m, h := bookTopInWindow(db, "market_order_book", tok, lo, hi, o.SnapMs, want); h {
			restHas++
			if m {
				restMatch++
			}
		}
		_ = marketIDForLeg(i, infos[i])
	}
	return
}

func bookTopInWindow(db *gorm.DB, table, token string, lo, hi, snapMs int64, want float64) (matched, hasRow bool) {
	type R struct {
		Asks string
	}
	var r R
	err := db.Table(table).Select("order_book_asks AS asks").
		Where("token_id=? AND snapshot_at_ms BETWEEN ? AND ?", token, lo, hi).
		Order(fmt.Sprintf("ABS(snapshot_at_ms - %d)", snapMs)).
		Limit(1).Scan(&r).Error
	if err != nil || strings.TrimSpace(r.Asks) == "" {
		return false, false
	}
	hasRow = true
	top := topAskFromRaw(r.Asks)
	if top > 0 && want > 0 && priceClose(top, want) {
		matched = true
	}
	return matched, hasRow
}

func hasRESTMap(db *gorm.DB, o oppRow, winMs int64) bool {
	if o.MapKey == nil || strings.TrimSpace(*o.MapKey) == "" || o.SnapMs <= 0 {
		return false
	}
	var n int64
	db.Table("market_order_book").
		Where("map_key=? AND snapshot_at_ms BETWEEN ? AND ?", strings.TrimSpace(*o.MapKey), o.SnapMs-winMs, o.SnapMs+winMs).
		Count(&n)
	return n > 0
}

func priceClose(a, b float64) bool {
	if a <= 0 || b <= 0 {
		return false
	}
	if math.Abs(a-b) <= 0.01 {
		return true
	}
	rel := math.Abs(a-b) / b
	return rel <= 0.01
}

func topAskFromRaw(raw string) float64 {
	var rows [][]string
	if json.Unmarshal([]byte(raw), &rows) == nil {
		for _, r := range rows {
			if len(r) < 1 {
				continue
			}
			var p float64
			fmt.Sscanf(r[0], "%f", &p)
			if p > 0 {
				return p
			}
		}
	}
	var ls []book.Level
	if json.Unmarshal([]byte(raw), &ls) == nil {
		for _, lv := range ls {
			if book.ValidPrice(lv.Price) && lv.Size > 0 {
				return lv.Price
			}
		}
	}
	return 0
}

func printLegDetail(db *gorm.DB, o oppRow, winMs int64) {
	fmt.Printf("  map_key=%v total_cost=%.8f profit_space=%.8f arb_qty=%.4f\n",
		o.MapKey, decF(o.TotalCost), decF(o.ProfitSpace), decF(o.ArbQty))
	lo, hi := o.SnapMs-winMs, o.SnapMs+winMs
	infos := []string{o.L1Info, o.L2Info, o.L3Info}
	books := []string{o.L1Books, o.L2Books, o.L3Books}
	asks := []*decimal.Decimal{o.L1Ask, o.L2Ask, o.L3Ask}
	for i := 0; i < 3; i++ {
		ask := askF(asks[i])
		tok := tokenForLeg(i, infos[i])
		fmt.Printf("  leg%d token=%s stored_ask=%.6f json_top=%.6f\n", i+1, tok, ask, legTopAsk(books[i]))
		for _, tbl := range []string{"market_order_book_ws", "market_order_book"} {
			type R struct {
				Platform string
				SnapMs   int64
				Asks     string
			}
			var hit R
			if tok == "" {
				continue
			}
			err := db.Table(tbl).Select("platform, snapshot_at_ms, order_book_asks").
				Where("token_id=? AND snapshot_at_ms BETWEEN ? AND ?", tok, lo, hi).
				Order(fmt.Sprintf("ABS(snapshot_at_ms - %d)", o.SnapMs)).
				Limit(1).Scan(&hit).Error
			if err != nil || hit.Asks == "" {
				fmt.Printf("    %s: (no row [%d,%d])\n", tbl, lo, hi)
				continue
			}
			fmt.Printf("    %s %s snap=%d Δms=%d top=%.6f\n", tbl, hit.Platform, hit.SnapMs, hit.SnapMs-o.SnapMs, topAskFromRaw(hit.Asks))
		}
	}
}