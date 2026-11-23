// 单次佐证：arb 行 vs market_order_book_ws 同窗顶价。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"github.com/shopspring/decimal"
)

func main() {
	oppID := flag.Uint("id", 1367, "arb_binary_opportunity id")
	winMs := flag.Int64("window", 5000, "±ms around snapshot_at_ms")
	flag.Parse()

	_ = conf.NewCfg()
	db := boot.NewDBClient()

	type opp struct {
		ID          uint            `gorm:"column:id"`
		MapKey      *string         `gorm:"column:map_key"`
		TotalCost   decimal.Decimal `gorm:"column:total_cost"`
		ProfitSpace decimal.Decimal `gorm:"column:profit_space"`
		Threshold   decimal.Decimal `gorm:"column:threshold_value"`
		L1Ask       *decimal.Decimal `gorm:"column:leg_1_best_ask"`
		L2Ask       *decimal.Decimal `gorm:"column:leg_2_best_ask"`
		SnapMs      int64           `gorm:"column:snapshot_at_ms"`
		L1Info      string          `gorm:"column:leg_1_info"`
		L2Info      string          `gorm:"column:leg_2_info"`
	}
	var o opp
	if err := db.Table("arb_binary_opportunity").Where("id = ?", *oppID).First(&o).Error; err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	type yesL struct {
		YesTokenID string `json:"yes_token_id"`
	}
	type noL struct {
		NoTokenID string `json:"no_token_id"`
	}
	var y yesL
	var n noL
	_ = json.Unmarshal([]byte(o.L1Info), &y)
	_ = json.Unmarshal([]byte(o.L2Info), &n)

	var l1, l2 float64
	if o.L1Ask != nil {
		l1, _ = o.L1Ask.Float64()
	}
	if o.L2Ask != nil {
		l2, _ = o.L2Ask.Float64()
	}
	tc, _ := o.TotalCost.Float64()
	ps, _ := o.ProfitSpace.Float64()
	th, _ := o.Threshold.Float64()

	fmt.Printf("#%d map_key=%v snap_ms=%d\n", o.ID, o.MapKey, o.SnapMs)
	fmt.Printf("  leg1_best_ask=%.8f leg2_best_ask=%.8f sum=%.8f\n", l1, l2, l1+l2)
	fmt.Printf("  total_cost=%.8f profit_space=%.8f threshold=%.8f pass=%v\n", tc, ps, th, tc < th)

	ms := o.SnapMs
	lo, hi := ms-*winMs, ms+*winMs
	legs := []struct {
		label string
		tok   string
		want  float64
	}{
		{"leg1 POLY", y.YesTokenID, l1},
		{"leg2 PREDICT", n.NoTokenID, l2},
	}
	for _, leg := range legs {
		if leg.tok == "" {
			continue
		}
		type R struct {
			ID           int64  `gorm:"column:id"`
			Platform     string `gorm:"column:platform"`
			SnapshotAtMs int64  `gorm:"column:snapshot_at_ms"`
			Asks         string `gorm:"column:order_book_asks"`
		}
		var rows []R
		_ = db.Table("market_order_book_ws").
			Select("id, platform, snapshot_at_ms, order_book_asks").
			Where("token_id = ? AND snapshot_at_ms BETWEEN ? AND ?", leg.tok, lo, hi).
			Order(fmt.Sprintf("ABS(snapshot_at_ms - %d)", ms)).
			Limit(5).
			Find(&rows)
		fmt.Printf("\n%s want=%.4f token=%s\n", leg.label, leg.want, leg.tok)
		if len(rows) == 0 {
			fmt.Println("  (no market_order_book_ws in window)")
			continue
		}
		for _, r := range rows {
			top := wsTopAsk(r.Asks)
			d := r.SnapshotAtMs - ms
			match := ""
			if top > 0 && abs(top-leg.want) <= 0.01 {
				match = " MATCH"
			}
			fmt.Printf("  ws id=%d %s snap_ms=%d Δms=%d top_ask=%.4f%s\n", r.ID, r.Platform, r.SnapshotAtMs, d, top, match)
		}
	}
}

func wsTopAsk(raw string) float64 {
	var rows [][]string
	if json.Unmarshal([]byte(raw), &rows) != nil {
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

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}