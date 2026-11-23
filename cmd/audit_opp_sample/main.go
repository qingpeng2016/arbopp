package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

func main() {
	conf.NewCfg()
	db := boot.NewDBClient()
	rows, err := db.Raw(`SELECT id, snapshot_at_ms,
		CAST(total_cost AS CHAR), CAST(threshold_value AS CHAR), CAST(profit_space AS CHAR)
		FROM arb_binary_opportunity ORDER BY id DESC LIMIT 8`).Rows()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rows.Close()
	fmt.Println("=== latest opps (raw) ===")
	for rows.Next() {
		var id, snap int64
		var tc, th, ps string
		_ = rows.Scan(&id, &snap, &tc, &th, &ps)
		cost, _ := strconv.ParseFloat(strings.TrimSpace(tc), 64)
		thv, _ := strconv.ParseFloat(strings.TrimSpace(th), 64)
		fmt.Printf("id=%d snap_ms=%d cost=%s th=%s ps=%s pass_th=%v\n", id, snap, tc, th, ps, cost < thv)
	}
	var total, snapZero, snapPos int64
	db.Raw(`SELECT COUNT(*) FROM arb_binary_opportunity`).Scan(&total)
	db.Raw(`SELECT COUNT(*) FROM arb_binary_opportunity WHERE snapshot_at_ms=0`).Scan(&snapZero)
	db.Raw(`SELECT COUNT(*) FROM arb_binary_opportunity WHERE snapshot_at_ms>0`).Scan(&snapPos)
	fmt.Printf("\nsnapshot_at_ms=0: %d / %d (positive ms: %d)\n", snapZero, total, snapPos)
}