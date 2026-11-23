package main

import (
	"fmt"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

func main() {
	conf.NewCfg()
	db := boot.NewDBClient()

	type cnt struct {
		OppKind string
		N       int64
	}
	var kinds []cnt
	db.Raw(`SELECT opp_kind, COUNT(*) n FROM arb_binary_opportunity GROUP BY opp_kind ORDER BY n DESC`).Scan(&kinds)
	fmt.Println("=== arb_binary_opportunity by opp_kind ===")
	for _, c := range kinds {
		fmt.Printf("  %s: %d\n", c.OppKind, c.N)
	}
	var total int64
	db.Raw(`SELECT COUNT(*) FROM arb_binary_opportunity`).Scan(&total)
	fmt.Printf("  TOTAL: %d\n", total)

	var recent int64
	db.Raw(`SELECT COUNT(*) FROM arb_binary_opportunity WHERE created_at > ?`, time.Now().Add(-2*time.Hour)).Scan(&recent)
	fmt.Printf("  last 2h: %d\n", recent)

	db.Raw(`SELECT id, opp_kind, map_key, total_cost, profit_space, threshold_key, threshold_value, snapshot_at
		FROM arb_binary_opportunity ORDER BY id DESC LIMIT 8`).Scan(&[]struct{}{})

	type row struct {
		ID              uint
		OppKind         string
		MapKey          *string
		TotalCost       float64
		ProfitSpace     float64
		ThresholdKey    string
		ThresholdValue  float64
		SnapshotAt      time.Time
	}
	var rows []row
	db.Raw(`SELECT id, opp_kind, map_key, total_cost, profit_space, threshold_key, threshold_value, snapshot_at
		FROM arb_binary_opportunity ORDER BY id DESC LIMIT 8`).Scan(&rows)
	fmt.Println("\n=== latest opportunities ===")
	for _, r := range rows {
		mk := ""
		if r.MapKey != nil {
			mk = *r.MapKey
		}
		fmt.Printf("  id=%d kind=%s map=%s cost=%.4f profit=%.4f th=%s=%.2f at=%s\n",
			r.ID, r.OppKind, mk, r.TotalCost, r.ProfitSpace, r.ThresholdKey, r.ThresholdValue, r.SnapshotAt.Format(time.RFC3339))
	}

	fmt.Println("\n=== order books ===")
	var rest, ws int64
	var sxR, sxW int64
	db.Raw(`SELECT COUNT(*) FROM market_order_book`).Scan(&rest)
	db.Raw(`SELECT COUNT(*) FROM market_order_book WHERE platform='SXBET'`).Scan(&sxR)
	db.Raw(`SELECT COUNT(*) FROM market_order_book_ws`).Scan(&ws)
	db.Raw(`SELECT COUNT(*) FROM market_order_book_ws WHERE platform='SXBET'`).Scan(&sxW)
	fmt.Printf("  market_order_book=%d (SX=%d)\n  market_order_book_ws=%d (SX=%d)\n", rest, sxR, ws, sxW)

	fmt.Println("\n=== thresholds (status=1) ===")
	type th struct {
		ConfigKey   string
		ConfigValue float64
	}
	var ths []th
	db.Raw(`SELECT config_key, config_value FROM config_strategy_threshold WHERE status=1 ORDER BY id`).Scan(&ths)
	for _, t := range ths {
		fmt.Printf("  %s = %.4f\n", t.ConfigKey, t.ConfigValue)
	}

	fmt.Println("\n=== running groups (stat_kv) ===")
	var itemVal string
	var desc string
	db.Raw(`SELECT item_value, description FROM stat_kv WHERE item_key='market_running_count_strategy' LIMIT 1`).Row().Scan(&itemVal, &desc)
	fmt.Printf("  running groups count=%s\n", itemVal)

	fmt.Println("\n=== outcome types in running map_keys ===")
	db.Raw(`
		SELECT o.outcome_type, o.platform, COUNT(*) c
		FROM market_outcome o
		JOIN market m ON m.id=o.market_row_id
		WHERE m.map_key IN (
			'v1|International|crint-lkaw-pakw-2026-11-23',
			'v1|NCAA篮球|cbb-sjst-loych-2026-11-22'
		)
		GROUP BY o.outcome_type, o.platform
		ORDER BY o.outcome_type, o.platform`).Scan(&[]struct{}{})

	type oc struct {
		OutcomeType string
		Platform    string
		C           int64
	}
	var ocs []oc
	db.Raw(`
		SELECT o.outcome_type, o.platform, COUNT(*) c
		FROM market_outcome o
		JOIN market m ON m.id=o.market_row_id
		WHERE m.map_key IN (
			'v1|International|crint-lkaw-pakw-2026-11-23',
			'v1|NCAA篮球|cbb-sjst-loych-2026-11-22'
		)
		GROUP BY o.outcome_type, o.platform
		ORDER BY o.outcome_type, o.platform`).Scan(&ocs)
	for _, o := range ocs {
		fmt.Printf("  type=%s platform=%s count=%d\n", o.OutcomeType, o.Platform, o.C)
	}

	fmt.Println("\n=== 1X2 opps ever in DB ===")
	var x2 int64
	db.Raw(`SELECT COUNT(*) FROM arb_binary_opportunity WHERE opp_kind LIKE 'ONE_X2%'`).Scan(&x2)
	fmt.Printf("  ONE_X2_* total (all time): %d\n", x2)
}