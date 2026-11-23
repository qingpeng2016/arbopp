// 批量核对 arb_binary_opportunity：行内自洽 + 同窗 WS/REST 顶价（不测盈亏语义）。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"gorm.io/gorm"
)

type oppRow struct {
	ID          uint
	OppKind     string
	MapKey      *string
	TotalCost   float64
	ProfitSpace float64
	ArbQty      float64
	ArbProfit   float64
	Threshold   float64
	ThKey       string
	SnapMs      int64
	L1Ask, L2Ask *float64
	L1Books, L2Books string
	L1Info, L2Info string
}

func main() {
	limit := flag.Int("n", 40, "max opportunities")
	winMs := flag.Int64("window", 5000, "WS/REST window ms before snap")
	flag.Parse()
	conf.NewCfg()
	db := boot.NewDBClient()

	var rows []oppRow
	db.Raw(`SELECT id, opp_kind, map_key, total_cost, profit_space, arb_qty, arb_profit,
		threshold_value, threshold_key, snapshot_at_ms,
		leg_1_best_ask, leg_2_best_ask, leg_1_order_books, leg_2_order_books,
		leg_1_platform, leg_2_platform,
		leg_1_info, leg_2_info
		FROM arb_binary_opportunity ORDER BY id DESC LIMIT ?`, *limit).Scan(&rows)
	if len(rows) == 0 {
		fmt.Println("no rows")
		os.Exit(0)
	}

	var (
		internalOK, internalBad int
		wsAny, wsTopMatch       int
		restAny                 int
		thOK                    int
	)

	for i := len(rows) - 1; i >= 0; i-- {
		o := rows[i]
		flags := internalCheck(o)
		if len(flags) == 0 {
			internalOK++
		} else {
			internalBad++
		}

		if o.Threshold > 0 && o.TotalCost < o.Threshold {
			thOK++
		}

		mk := ""
		if o.MapKey != nil {
			mk = *o.MapKey
		}
		t1 := legTopAsk(o.L1Books)
		t2 := legTopAsk(o.L2Books)
		tok1 := tokenFromLegInfo(o.L1Info, true)
		tok2 := tokenFromLegInfo(o.L2Info, false)
		if o.SnapMs > 0 && mk != "" {
			if matched, has := wsTopInWindow(db, tok1, o.SnapMs, *winMs, t1, o.L1Ask); has {
				wsAny++
				if matched {
					wsTopMatch++
				}
			}
			if matched, has := wsTopInWindow(db, tok2, o.SnapMs, *winMs, t2, o.L2Ask); has {
				wsAny++
				if matched {
					wsTopMatch++
				}
			}
			if hasREST(db, mk, o.SnapMs, *winMs) {
				restAny++
			}
		}

		if internalBad <= 5 || o.ID >= rows[0].ID-3 {
			if len(flags) > 0 {
				fmt.Printf("#%d flags: %s\n", o.ID, strings.Join(flags, "; "))
			}
		}
	}

	legs := wsAny
	fmt.Printf("\n=== SUMMARY (last %d opps, oldest..newest checked) ===\n", len(rows))
	fmt.Printf("内部自洽（算术/标量vs JSON net）: OK=%d BAD=%d\n", internalOK, internalBad)
	fmt.Printf("total_cost < threshold_value: %d/%d\n", thOK, len(rows))
	fmt.Printf("WS 腿级: 有行前窗数据 %d 腿; 顶价±1%%对齐 %d 腿\n", legs, wsTopMatch)
	fmt.Printf("REST 有 map 窗前窗行: %d/%d 场\n", restAny, len(rows))
	fmt.Printf("\n逻辑说明: 亏损行(profit<0)属 TEST 阈值; 本批不判漏洞。\n")
}

func internalCheck(o oppRow) []string {
	var f []string
	const eps = 1e-4
	if math.Abs((o.TotalCost+o.ProfitSpace)-1.0) > 0.0001 && strings.HasPrefix(o.OppKind, "BINARY") {
		f = append(f, fmt.Sprintf("profit_space!=1-cost (%.6f+%.6f)", o.ProfitSpace, o.TotalCost))
	}
	if math.Abs(o.ArbProfit-o.ArbQty*o.ProfitSpace) > 0.0002 {
		f = append(f, "arb_profit!=qty*profit_space")
	}
	n1 := legNetCost(o.L1Books)
	n2 := legNetCost(o.L2Books)
	if n1 > 0 && n2 > 0 {
		sum := n1 + n2
		if math.Abs(sum-o.TotalCost) > 0.0002 {
			f = append(f, fmt.Sprintf("total_cost vs JSON net sum Δ=%.6f", o.TotalCost-sum))
		}
	}
	top1 := legTopAsk(o.L1Books)
	if o.L1Ask != nil && top1 > 0 && math.Abs(*o.L1Ask-top1) > 0.0001 {
		f = append(f, "leg1_best_ask!=json top ask")
	}
	top2 := legTopAsk(o.L2Books)
	if o.L2Ask != nil && top2 > 0 && math.Abs(*o.L2Ask-top2) > 0.0001 {
		f = append(f, "leg2_best_ask!=json top ask")
	}
	return f
}

type legBooks struct {
	NetCost float64 `json:"net_cost"`
	Books   []struct {
		Asks []struct {
			Price float64 `json:"price"`
			Size  float64 `json:"size"`
		} `json:"asks"`
	} `json:"books"`
}

func legNetCost(raw string) float64 {
	var p legBooks
	if json.Unmarshal([]byte(raw), &p) != nil {
		return 0
	}
	return p.NetCost
}

func legTopAsk(raw string) float64 {
	var p legBooks
	if json.Unmarshal([]byte(raw), &p) != nil {
		return 0
	}
	for _, b := range p.Books {
		if len(b.Asks) > 0 {
			return b.Asks[0].Price
		}
	}
	return 0
}

func tokenFromLegInfo(raw string, preferYes bool) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var m map[string]interface{}
	if json.Unmarshal([]byte(raw), &m) != nil {
		return ""
	}
	if preferYes {
		if v, ok := m["yes_token_id"].(string); ok {
			return strings.TrimSpace(v)
		}
	}
	if v, ok := m["no_token_id"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := m["yes_token_id"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func wsTopInWindow(db *gorm.DB, token string, snapMs, winMs int64, jsonTop float64, colAsk *float64) (matched, hasRow bool) {
	if token == "" {
		return false, false
	}
	lo, hi := snapMs-winMs, snapMs
	type R struct {
		Asks string
	}
	var r R
	err := db.Raw(`SELECT order_book_asks AS asks FROM market_order_book_ws
		WHERE token_id=? AND snapshot_at_ms BETWEEN ? AND ?
		ORDER BY ABS(snapshot_at_ms-?) LIMIT 1`, token, lo, hi, snapMs).Scan(&r).Error
	if err != nil || r.Asks == "" {
		return false, false
	}
	hasRow = true
	wsTop := topAskJSON(r.Asks)
	want := jsonTop
	if colAsk != nil && *colAsk > 0 {
		want = *colAsk
	}
	if wsTop > 0 && want > 0 && math.Abs(wsTop-want) <= 0.01 {
		matched = true
	}
	return matched, hasRow
}

func hasREST(db *gorm.DB, mapKey string, snapMs, winMs int64) bool {
	var n int64
	db.Raw(`SELECT COUNT(*) FROM market_order_book WHERE map_key=? AND snapshot_at_ms BETWEEN ? AND ?`,
		mapKey, snapMs-winMs, snapMs).Scan(&n)
	return n > 0
}

func topAskJSON(raw string) float64 {
	var rows [][]string
	if json.Unmarshal([]byte(raw), &rows) != nil {
		var ls []book.Level
		if json.Unmarshal([]byte(raw), &ls) == nil && len(ls) > 0 {
			return ls[0].Price
		}
		return 0
	}
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
	return 0
}
