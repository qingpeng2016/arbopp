package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

func main() {
	conf.NewCfg()
	db := boot.NewDBClient()

	type opp struct {
		ID           uint
		MapKey       *string
		TotalCost    float64
		ProfitSpace  float64
		SnapshotMs   int64
		Leg1Platform *string
		Leg2Platform *string
		L1Books      string
		L2Books      string
		L1Ask        *float64
		L2Ask        *float64
	}
	var o opp
	db.Raw(`SELECT id, map_key, total_cost, profit_space, snapshot_at_ms, leg_1_platform, leg_2_platform,
		leg_1_order_books, leg_2_order_books, leg_1_best_ask, leg_2_best_ask
		FROM arb_binary_opportunity WHERE id=1`).Scan(&o)
	fmt.Printf("opp #1 cost=%.6f profit_space=%.6f snap_ms=%d\n", o.TotalCost, o.ProfitSpace, o.SnapshotMs)

	var byKind []struct {
		Kind string
		N    int64
		Neg  int64
	}
	db.Raw(`SELECT opp_kind kind, COUNT(*) n,
		SUM(CASE WHEN profit_space < 0 THEN 1 ELSE 0 END) neg
		FROM arb_binary_opportunity GROUP BY opp_kind`).Scan(&byKind)
	fmt.Println("\nprofit_space<0 (TEST 无硬顶):")
	for _, r := range byKind {
		fmt.Printf("  %s total=%d negative=%d\n", r.Kind, r.N, r.Neg)
	}

	// draw groups in running map keys
	fmt.Println("\noutcome names (moneyline) for 1X2 check:")
	var lines []struct {
		MapKey      string
		Platform    string
		OutcomeName *string
		OutcomeType string
	}
	db.Raw(`
		SELECT m.map_key, o.platform, o.outcome_name, o.outcome_type
		FROM market_outcome o JOIN market m ON m.id=o.market_row_id
		WHERE m.map_key IN (
			'v1|International|crint-lkaw-pakw-2026-11-23',
			'v1|NCAA篮球|cbb-sjst-loych-2026-11-22'
		) AND o.outcome_type='moneyline'
		ORDER BY m.map_key, o.platform`).Scan(&lines)
	for _, l := range lines {
		name := ""
		if l.OutcomeName != nil {
			name = *l.OutcomeName
		}
		fmt.Printf("  %s | %s | %s\n", l.MapKey, l.Platform, name)
	}

	// internal consistency id=1
	var net1, net2 float64
	parseNet := func(j string) float64 {
		j = strings.TrimSpace(j)
		var p struct {
			NetCost float64 `json:"net_cost"`
		}
		_ = json.Unmarshal([]byte(j), &p)
		return p.NetCost
	}
	net1 = parseNet(o.L1Books)
	net2 = parseNet(o.L2Books)
	fmt.Printf("\n#1 leg net_cost sum=%.6f vs total_cost=%.6f (delta=%.6f)\n", net1+net2, o.TotalCost, o.TotalCost-net1-net2)

	mk := ""
	if o.MapKey != nil {
		mk = *o.MapKey
	}
	if mk != "" && o.SnapshotMs > 0 {
		lo, hi := o.SnapshotMs-5000, o.SnapshotMs
		var wsCnt int64
		db.Raw(`SELECT COUNT(*) FROM market_order_book_ws
			WHERE map_key=? AND snapshot_at_ms BETWEEN ? AND ?`, mk, lo, hi).Scan(&wsCnt)
		var restCnt int64
		db.Raw(`SELECT COUNT(*) FROM market_order_book
			WHERE map_key=? AND snapshot_at_ms BETWEEN ? AND ?`, mk, lo, hi).Scan(&restCnt)
		fmt.Printf("WS rows in [-5s,0] window for map: %d | REST: %d\n", wsCnt, restCnt)
	}
}