// 从 leg_*_order_books JSON 回填 arb_qty / arb_profit（与落库 MinLegTopAskQty 同义）。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func main() {
	dry := flag.Bool("dry-run", false, "只计算不写库")
	limit := flag.Int("limit", 0, "最多处理行数，0=全部")
	onlyZero := flag.Bool("only-zero", true, "仅 arb_qty=0 的行")
	table := flag.String("table", "both", "arb_binary_opportunity | backtest_arb_binary_opportunity | both")
	flag.Parse()

	_ = conf.NewCfg()
	db := boot.NewDBClient()

	tables := tablesFromFlag(*table)
	var total int64
	for _, t := range tables {
		remain := int64(0)
		if *limit > 0 {
			remain = int64(*limit) - total
			if remain <= 0 {
				break
			}
		}
		n, err := backfillTable(db, t, *dry, remain, *onlyZero)
		if err != nil {
			fmt.Printf("FAIL %s: %v\n", t, err)
			os.Exit(1)
		}
		total += n
		if *limit > 0 && total >= int64(*limit) {
			break
		}
	}
	fmt.Printf("done rows=%d dry_run=%v\n", total, *dry)
}

func tablesFromFlag(s string) []string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "live", "arb", "arb_binary_opportunity":
		return []string{"arb_binary_opportunity"}
	case "backtest", "backtest_arb_binary_opportunity":
		return []string{"backtest_arb_binary_opportunity"}
	case "both", "all":
		return []string{"arb_binary_opportunity", "backtest_arb_binary_opportunity"}
	default:
		fmt.Fprintf(os.Stderr, "unknown -table %q\n", s)
		os.Exit(2)
		return nil
	}
}

type oppRow struct {
	ID          uint            `gorm:"column:id"`
	ProfitSpace decimal.Decimal `gorm:"column:profit_space"`
	Leg1Books   string          `gorm:"column:leg_1_order_books"`
	Leg2Books   string          `gorm:"column:leg_2_order_books"`
	Leg3Books   string          `gorm:"column:leg_3_order_books"`
}

func backfillTable(db *gorm.DB, table string, dry bool, limit int64, onlyZero bool) (int64, error) {
	q := db.Table(table).Select("id, profit_space, leg_1_order_books, leg_2_order_books, leg_3_order_books")
	if onlyZero {
		q = q.Where("arb_qty = 0")
	}
	q = q.Order("id ASC")
	if limit > 0 {
		q = q.Limit(int(limit))
	}
	var rows []oppRow
	if err := q.Find(&rows).Error; err != nil {
		return 0, err
	}
	var updated int64
	for _, row := range rows {
		qtyF := minLegTopAskQty(row.Leg1Books, row.Leg2Books, row.Leg3Books)
		qty := decimal.NewFromFloat(qtyF).Round(8)
		profit := qty.Mul(row.ProfitSpace).Round(8)
		if dry {
			fmt.Printf("[%s] id=%d arb_qty=%s arb_profit=%s profit_space=%s\n",
				table, row.ID, qty, profit, row.ProfitSpace)
			updated++
			continue
		}
		res := db.Exec(`UPDATE `+quoteIdent(table)+` SET arb_qty = ?, arb_profit = ? WHERE id = ?`,
			qty, profit, row.ID)
		if res.Error != nil {
			return updated, res.Error
		}
		updated += res.RowsAffected
	}
	fmt.Printf("OK %s updated=%d scanned=%d\n", table, updated, len(rows))
	return int64(len(rows)), nil
}

func quoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func minLegTopAskQty(legs ...string) float64 {
	var min float64
	var have bool
	for _, leg := range legs {
		sz, ok := topAskSzFromLegJSON(leg)
		if !ok {
			continue
		}
		if !have || sz < min {
			min, have = sz, true
		}
	}
	if !have {
		return 0
	}
	return min
}

func topAskSzFromLegJSON(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return 0, false
	}
	var body struct {
		Books []struct {
			Asks []book.Level `json:"asks"`
		} `json:"books"`
	}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		return 0, false
	}
	for _, bk := range body.Books {
		if len(bk.Asks) == 0 {
			continue
		}
		p, s := bk.Asks[0].Price, bk.Asks[0].Size
		if book.ValidPrice(p) && s > 0 {
			return s, true
		}
	}
	return 0, false
}