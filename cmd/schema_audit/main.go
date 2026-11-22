package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

func main() {
	_ = conf.NewCfg()
	db := boot.NewDBClient()
	if db == nil {
		fmt.Println("db nil")
		os.Exit(1)
	}
	ctx := context.Background()
	var tables []string
	if err := db.WithContext(ctx).Raw(`
SELECT table_name FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'
ORDER BY table_name`).Scan(&tables).Error; err != nil {
		fmt.Println("err:", err)
		os.Exit(1)
	}
	fmt.Println("=== TABLES ===")
	for _, t := range tables {
		fmt.Println(t)
	}

	checks := []struct{ table, column string }{
		{"orders", "strategy_type"},
		{"orders", "backtest_task_id"},
		{"arb_binary_opportunity", "draw_info"},
		{"arb_binary_opportunity", "opp_kind"},
		{"backtest_orders", "strategy_type"},
		{"backtest_arb_binary_opportunity", "draw_info"},
		{"stat_kv", "group_name"},
		{"market_category", "config_json"},
		{"config_strategy_fee", "taker_fee_coef"},
	}
	fmt.Println("\n=== COLUMN CHECK ===")
	for _, c := range checks {
		var cnt int64
		q := `SELECT COUNT(*) FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`
		_ = db.Raw(q, c.table, c.column).Scan(&cnt)
		ok := "MISSING"
		if cnt > 0 {
			ok = "ok"
		}
		// table exists?
		var tc int64
		_ = db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, c.table).Scan(&tc)
		if tc == 0 {
			ok = "TABLE_MISSING"
		}
		fmt.Printf("%s.%s: %s\n", c.table, c.column, ok)
	}

	legacy := []string{"arb_binary_order", "backtest_arb_binary_order"}
	fmt.Println("\n=== LEGACY TABLES ===")
	for _, t := range legacy {
		var tc int64
		_ = db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, t).Scan(&tc)
		if tc > 0 {
			var n int64
			_ = db.Raw("SELECT COUNT(*) FROM `" + t + "`").Scan(&n)
			fmt.Printf("%s: EXISTS rows=%d\n", t, n)
		}
	}
	if has(tables, "arb_binary_order") && !has(tables, "orders") {
		fmt.Println("NOTE: need RENAME arb_binary_order -> orders")
	}

	// thresholds
	var keys []string
	_ = db.Raw(`SELECT config_key FROM config_strategy_threshold WHERE status = 1 ORDER BY config_key`).Scan(&keys)
	fmt.Println("\n=== THRESHOLDS (active) ===")
	for _, k := range keys {
		fmt.Println(k)
	}
	var feeN int64
	_ = db.Raw(`SELECT COUNT(*) FROM config_strategy_fee WHERE status = 1`).Scan(&feeN)
	fmt.Printf("\nconfig_strategy_fee active rows: %d\n", feeN)

	wantTables := []string{
		"backtest_orders", "backtest_task", "backtest_arb_binary_opportunity",
	}
	fmt.Println("\n=== MISSING EXPECTED TABLES ===")
	for _, t := range wantTables {
		if !has(tables, t) {
			fmt.Println(t)
		}
	}

	var bots []struct {
		Task      string
		IsEnabled int
	}
	_ = db.Raw(`SELECT task_name, is_enabled FROM bot_schedule_config WHERE module='less_one' ORDER BY exe_sort`).Scan(&bots)
	fmt.Println("\n=== BOT SCHEDULE ===")
	for _, b := range bots {
		fmt.Printf("%s enabled=%d\n", b.Task, b.IsEnabled)
	}
}

func has(ss []string, x string) bool {
	i := sort.SearchStrings(ss, x)
	return i < len(ss) && ss[i] == x
}