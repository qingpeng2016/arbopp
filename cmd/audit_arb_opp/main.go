// 临时审计 arb_binary_opportunity 自洽性（profit_space、腿价、JSON 顶档）。
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type oppRow struct {
	ID            uint             `gorm:"column:id"`
	OppKind       string           `gorm:"column:opp_kind"`
	MapKey        *string          `gorm:"column:map_key"`
	TotalCost     decimal.Decimal  `gorm:"column:total_cost"`
	ProfitSpace   decimal.Decimal  `gorm:"column:profit_space"`
	ThresholdVal  decimal.Decimal  `gorm:"column:threshold_value"`
	ArbQty        decimal.Decimal  `gorm:"column:arb_qty"`
	ArbProfit     decimal.Decimal  `gorm:"column:arb_profit"`
	Leg1Ask       *decimal.Decimal `gorm:"column:leg_1_best_ask"`
	Leg2Ask       *decimal.Decimal `gorm:"column:leg_2_best_ask"`
	Leg3Ask       *decimal.Decimal `gorm:"column:leg_3_best_ask"`
	Leg1Books     string           `gorm:"column:leg_1_order_books"`
	Leg2Books     string           `gorm:"column:leg_2_order_books"`
	Leg3Books     string           `gorm:"column:leg_3_order_books"`
	Leg1Info      string           `gorm:"column:leg_1_info"`
	Leg2Info      string           `gorm:"column:leg_2_info"`
	SnapshotAtMs  int64            `gorm:"column:snapshot_at_ms"`
}

type yesLegInfo struct {
	YesMarketID  string `json:"yes_market_id"`
	YesOutcomeID uint   `json:"yes_outcome_id"`
	YesTokenID   string `json:"yes_token_id"`
}

type noLegInfo struct {
	NoMarketID  string `json:"no_market_id"`
	NoOutcomeID uint   `json:"no_outcome_id"`
	NoTokenID   string `json:"no_token_id"`
}

type booksPayload struct {
	Platform                string `json:"platform"`
	PlatformMarketOutcomeID string `json:"platform_market_outcome_id"`
	Books                   []struct {
		Asks []book.Level `json:"asks"`
	} `json:"books"`
}

func main() {
	_ = conf.NewCfg()
	db := boot.NewDBClient()

	var total int64
	if err := db.Table("arb_binary_opportunity").Count(&total).Error; err != nil {
		fmt.Println("DB error:", err)
		os.Exit(1)
	}
	fmt.Printf("arb_binary_opportunity rows: %d\n\n", total)

	type kindCnt struct {
		OppKind string
		C       int64
	}
	var kinds []kindCnt
	_ = db.Table("arb_binary_opportunity").Select("opp_kind, count(*) as c").Group("opp_kind").Scan(&kinds)
	fmt.Println("by opp_kind:")
	for _, k := range kinds {
		fmt.Printf("  %s: %d\n", k.OppKind, k.C)
	}

	var neg, overTh int64
	_ = db.Table("arb_binary_opportunity").Where("profit_space <= 0").Count(&neg)
	_ = db.Table("arb_binary_opportunity").Where("total_cost >= threshold_value").Count(&overTh)
	fmt.Printf("\nprofit_space<=0: %d\n", neg)
	fmt.Printf("total_cost >= threshold_value: %d\n\n", overTh)

	var rows []oppRow
	if err := db.Table("arb_binary_opportunity").Order("id desc").Limit(80).Find(&rows).Error; err != nil {
		fmt.Println("query:", err)
		os.Exit(1)
	}

	var (
		psMismatch      int
		arbMismatch     int
		jsonTopMismatch int
		sumLegNoFee     int
		issues          []string
	)

	for _, r := range rows {
		tc, _ := r.TotalCost.Float64()
		ps, _ := r.ProfitSpace.Float64()
		th, _ := r.ThresholdVal.Float64()

		base := 1.0
		if strings.Contains(r.OppKind, "NO_TRIPLE") {
			base = 2.0
		}
		if math.Abs(ps-(base-tc)) > 1e-6 {
			psMismatch++
			if len(issues) < 15 {
				issues = append(issues, fmt.Sprintf("#%d profit_space %.8f != %.8f - total_cost %.8f", r.ID, ps, base-tc, tc))
			}
		}
		if th > 0 && tc >= th {
			if len(issues) < 20 {
				issues = append(issues, fmt.Sprintf("#%d FAIL threshold total_cost=%.8f >= %.8f kind=%s", r.ID, tc, th, r.OppKind))
			}
		}

		aq, _ := r.ArbQty.Float64()
		ap, _ := r.ArbProfit.Float64()
		if aq > 0 && math.Abs(ap-aq*ps) > 1e-4 {
			arbMismatch++
		}

		asks := []*decimal.Decimal{r.Leg1Ask, r.Leg2Ask, r.Leg3Ask}
		books := []string{r.Leg1Books, r.Leg2Books, r.Leg3Books}
		sum := 0.0
		legs := 0
		for i, a := range asks {
			if a == nil {
				continue
			}
			v, _ := a.Float64()
			if v <= 0 {
				continue
			}
			sum += v
			legs++
			if i < len(books) {
				jp, ok := jsonFirstAsk(books[i])
				if ok && math.Abs(jp-v) > 0.02 {
					jsonTopMismatch++
				}
			}
		}
		if r.OppKind == "BINARY_YES_NO" && legs == 2 && sum > 0 {
			if math.Abs(sum-tc) > 0.05 {
				sumLegNoFee++
				if len(issues) < 25 {
					issues = append(issues, fmt.Sprintf("#%d leg1+leg2=%.6f total_cost=%.6f (fees?) kind=%s", r.ID, sum, tc, r.OppKind))
				}
			}
		}
	}

	fmt.Printf("last %d rows checks:\n", len(rows))
	fmt.Printf("  profit_space != base-total_cost: %d\n", psMismatch)
	fmt.Printf("  arb_profit != arb_qty*profit_space (qty>0): %d\n", arbMismatch)
	fmt.Printf("  leg_best_ask vs JSON first ask diff>0.02: %d leg checks\n", jsonTopMismatch)
	fmt.Printf("  BINARY leg1+leg2 vs total_cost diff>0.05: %d\n\n", sumLegNoFee)

	if len(issues) > 0 {
		fmt.Println("sample issues:")
		for _, s := range issues {
			fmt.Println(" ", s)
		}
	}

	auditAllRows(db)

	var wsCnt int64
	_ = db.Table("market_order_book_ws").Count(&wsCnt)
	fmt.Printf("\nmarket_order_book_ws total rows: %d\n", wsCnt)

	var bad oppRow
	_ = db.Table("arb_binary_opportunity").Where("profit_space <= 0 OR total_cost >= threshold_value").Order("id desc").First(&bad)
	if bad.ID > 0 {
		fmt.Printf("\n=== WS corroboration (threshold/profit anomaly) #%d ===\n", bad.ID)
		corroborateWS(db, bad)
	}
	if len(rows) > 0 {
		fmt.Printf("\n=== WS corroboration (newest) #%d ===\n", rows[0].ID)
		corroborateWS(db, rows[0])
	}
}

func jsonFirstAsk(raw string) (float64, bool) {
	var p booksPayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &p); err != nil {
		return 0, false
	}
	for _, bk := range p.Books {
		if len(bk.Asks) == 0 {
			continue
		}
		px, sz := bk.Asks[0].Price, bk.Asks[0].Size
		if book.ValidPrice(px) && sz > 0 {
			return px, true
		}
	}
	return 0, false
}

func corroborateWS(db *gorm.DB, r oppRow) {
	ms := r.SnapshotAtMs
	lo, hi := ms-120_000, ms+120_000
	legs := []struct {
		name string
		info string
		ask  *decimal.Decimal
	}{
		{"leg1", r.Leg1Info, r.Leg1Ask},
		{"leg2", r.Leg2Info, r.Leg2Ask},
	}
	tc, _ := r.TotalCost.Float64()
	ps, _ := r.ProfitSpace.Float64()
	fmt.Printf("  kind=%s map_key=%v snap_ms=%d total_cost=%.6f profit_space=%.6f\n", r.OppKind, r.MapKey, ms, tc, ps)
	for _, leg := range legs {
		if leg.ask == nil {
			continue
		}
		tokenID, marketID := tokenFromLegJSON(leg.name, leg.info)
		askStored, _ := leg.ask.Float64()
		jsonAsk, jsonOK := jsonFirstAsk(legBooksFor(leg.name, r))
		fmt.Printf("  %s market_id=%s token=%s stored_ask=%.6f json_top_ask=%.6f\n", leg.name, marketID, tokenID, askStored, jsonAsk)
		if jsonOK && math.Abs(jsonAsk-askStored) > 0.02 {
			fmt.Printf("    !! leg JSON top vs stored best_ask diff=%.4f\n", jsonAsk-askStored)
		}

		type wsRow struct {
			Platform      string `gorm:"column:platform"`
			TokenID       string `gorm:"column:token_id"`
			SnapshotAtMs  int64  `gorm:"column:snapshot_at_ms"`
			OrderBookAsks string `gorm:"column:order_book_asks"`
		}
		var hits []wsRow
		q := db.Table("market_order_book_ws").Where("snapshot_at_ms BETWEEN ? AND ?", lo, hi)
		if tokenID != "" {
			q = q.Where("token_id = ?", tokenID)
		} else if marketID != "" && r.MapKey != nil {
			q = q.Where("map_key = ? AND platform_market_id = ?", *r.MapKey, marketID)
		}
		_ = q.Order(fmt.Sprintf("ABS(snapshot_at_ms - %d)", ms)).Limit(5).Find(&hits)
		if len(hits) == 0 {
			fmt.Printf("    market_order_book_ws: no row in [%d,%d]\n", lo, hi)
			continue
		}
		for _, h := range hits {
			wsAsk := parseWSFirstAsk(h.OrderBookAsks)
			fmt.Printf("    ws %s snap_ms=%d token=%s ws_top_ask=%.6f delta_ms=%d\n", h.Platform, h.SnapshotAtMs, h.TokenID, wsAsk, h.SnapshotAtMs-ms)
		}
	}
}

func legBooksFor(legName string, r oppRow) string {
	switch legName {
	case "leg1":
		return r.Leg1Books
	case "leg2":
		return r.Leg2Books
	default:
		return r.Leg3Books
	}
}

func tokenFromLegJSON(_, raw string) (tokenID, marketID string) {
	var y yesLegInfo
	if json.Unmarshal([]byte(raw), &y) == nil && y.YesTokenID != "" {
		return strings.TrimSpace(y.YesTokenID), strings.TrimSpace(y.YesMarketID)
	}
	var n noLegInfo
	if json.Unmarshal([]byte(raw), &n) == nil {
		return strings.TrimSpace(n.NoTokenID), strings.TrimSpace(n.NoMarketID)
	}
	return "", ""
}

func auditAllRows(db *gorm.DB) {
	type slim struct {
		ID          uint
		TotalCost   float64 `gorm:"column:total_cost"`
		Threshold   float64 `gorm:"column:threshold_value"`
		ProfitSpace float64 `gorm:"column:profit_space"`
		L1, L2      *float64
		Snap        int64 `gorm:"column:snapshot_at_ms"`
	}
	var all []slim
	_ = db.Table("arb_binary_opportunity").
		Select("id, total_cost, threshold_value, profit_space, leg_1_best_ask as l1, leg_2_best_ask as l2, snapshot_at_ms").
		Order("id").Find(&all)
	var overTh, legDiff, negPS int
	for _, r := range all {
		if r.ProfitSpace <= 0 {
			negPS++
		}
		if r.Threshold > 0 && r.TotalCost >= r.Threshold-1e-9 {
			overTh++
		}
		s := 0.0
		if r.L1 != nil {
			s += *r.L1
		}
		if r.L2 != nil {
			s += *r.L2
		}
		if s > 0 && math.Abs(s-r.TotalCost) > 0.02 {
			legDiff++
		}
	}
	fmt.Printf("\n=== all %d rows ===\n", len(all))
	fmt.Printf("  profit_space<=0: %d\n", negPS)
	fmt.Printf("  total_cost >= threshold: %d\n", overTh)
	fmt.Printf("  |leg1+leg2 - total_cost| > 0.02: %d (fee in total_cost is expected)\n", legDiff)

	var r1346 oppRow
	if db.Table("arb_binary_opportunity").Where("id = ?", 1346).First(&r1346).Error == nil {
		tc, _ := r1346.TotalCost.Float64()
		ps, _ := r1346.ProfitSpace.Float64()
		fmt.Printf("\n#1346 total_cost=%.8f profit_space=%.8f threshold=%s\n", tc, ps, r1346.ThresholdVal.String())
	}
}

func parseWSFirstAsk(asksJSON string) float64 {
	var rows [][]string
	if err := json.Unmarshal([]byte(asksJSON), &rows); err != nil {
		return 0
	}
	for _, r := range rows {
		if len(r) < 2 {
			continue
		}
		px, _ := strconv.ParseFloat(r[0], 64)
		sz, _ := strconv.ParseFloat(r[1], 64)
		if book.ValidPrice(px) && sz > 0 {
			return px
		}
	}
	return 0
}