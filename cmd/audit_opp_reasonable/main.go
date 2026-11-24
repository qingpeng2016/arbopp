// 判断 arb_binary_opportunity 是否合理：行内自洽为硬标准，WS 按腿 JSON 的 quote_at_ms 核对顶价（不依赖机会 snapshot_at_ms）。
//
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/audit_opp_reasonable -n=40
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/audit_opp_reasonable -id=301 -v
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

func main() {
	limit := flag.Int("n", 30, "条数")
	oppID := flag.Uint("id", 0, "单条")
	kinds := flag.String("kinds", "ONE_X2", "opp_kind 前缀，ALL=全部")
	winMs := flag.Int64("window", 15000, "相对 quote_at_ms 查 WS 的 ±ms")
	askTol := flag.Float64("ask_tol", 0.015, "顶卖一允许相对偏差")
	verbose := flag.Int("v", 5, "打印明细条数")
	flag.Parse()

	conf.NewCfg()
	db := boot.NewDBClient()

	rows := loadRows(db, *oppID, *limit, *kinds)
	if len(rows) == 0 {
		fmt.Println("无样本")
		os.Exit(0)
	}

	type counts struct {
		internalOK, internalBad int
		wsLegs, wsMatch         int
		strong, weak, suspect    int
	}
	var c counts

	for i, o := range rows {
		internal, legs := checkInternal(o)
		if len(internal) == 0 {
			c.internalOK++
		} else {
			c.internalBad++
		}

		wsHit, wsOK := checkWS(db, legs, *winMs, *askTol)
		c.wsLegs += wsHit
		c.wsMatch += wsOK

		verdict, why := verdict(internal, wsHit, wsOK, len(legs))
		switch verdict {
		case "strong":
			c.strong++
		case "weak":
			c.weak++
		case "suspect":
			c.suspect++
		}

		if i < *verbose || *oppID > 0 {
			mk := ""
			if o.MapKey != nil {
				mk = *o.MapKey
			}
			fmt.Printf("#%d %s map=%s → %s\n", o.ID, o.OppKind, mk, why)
			if len(internal) > 0 {
				fmt.Printf("  行内: %s\n", strings.Join(internal, "; "))
			}
			for _, L := range legs {
				if L.quoteMs <= 0 {
					fmt.Printf("  leg%d tok=%s 无 quote_at_ms\n", L.idx, shortTok(L.token))
					continue
				}
				fmt.Printf("  leg%d quote_ms=%d ask=%.6f json_top=%.6f net=%.6f",
					L.idx, L.quoteMs, L.storedAsk, L.jsonTop, L.netCost)
				if L.wsTop > 0 {
					fmt.Printf(" ws_top=%.6f Δms=%d", L.wsTop, L.wsDeltaMs)
				}
				fmt.Println()
			}
		}
	}

	fmt.Println("\n=== 机会合理性（推荐判据）===")
	fmt.Printf("样本 %d  行内自洽 OK=%d BAD=%d\n", len(rows), c.internalOK, c.internalBad)
	fmt.Printf("WS 按 quote_at_ms(±%dms): 有行 %d 腿，顶价对齐 %d 腿\n", *winMs, c.wsLegs, c.wsMatch)
	fmt.Printf("结论分布: 强合理=%d  合理(无/部分WS)=%d  可疑=%d\n", c.strong, c.weak, c.suspect)
	fmt.Println("\n判据说明:")
	fmt.Println("  · 强合理 = 行内自洽 + 每腿 WS 顶价与存盘 ask 对齐（与策略同源时刻 quote_at_ms）")
	fmt.Println("  · 合理     = 仅行内自洽（JSON 与列字段同源，已足够说明落库不是乱写）")
	fmt.Println("  · 可疑     = 行内不自洽，或 WS 与存盘价明显矛盾")
	fmt.Println("  · 不看 threshold_value；REST/orderbook_sync 不参与本脚本（时间轴不同，易误判）")
}

func shortTok(t string) string {
	t = strings.TrimSpace(t)
	if len(t) <= 12 {
		return t
	}
	return t[:6] + "…" + t[len(t)-4:]
}

func verdict(internal []string, wsHit, wsOK, nLeg int) (string, string) {
	if len(internal) > 0 {
		return "suspect", "可疑：行内不自洽"
	}
	if nLeg > 0 && wsHit == nLeg && wsOK == nLeg {
		return "strong", "强合理：行内OK + WS 全腿印证"
	}
	if wsHit > 0 && wsOK < wsHit {
		return "suspect", "可疑：部分 WS 与存盘顶价不一致"
	}
	return "weak", "合理：行内OK（WS 缺档或未全印证）"
}

type legEvidence struct {
	idx        int
	token      string
	quoteMs    int64
	storedAsk  float64
	jsonTop    float64
	netCost    float64
	wsTop      float64
	wsDeltaMs  int64
}

type oppRow struct {
	ID          uint            `gorm:"column:id"`
	OppKind     string          `gorm:"column:opp_kind"`
	MapKey      *string         `gorm:"column:map_key"`
	TotalCost   decimal.Decimal `gorm:"column:total_cost"`
	ProfitSpace decimal.Decimal `gorm:"column:profit_space"`
	ArbQty      decimal.Decimal `gorm:"column:arb_qty"`
	ArbProfit   decimal.Decimal `gorm:"column:arb_profit"`
	SnapMs      int64           `gorm:"column:snapshot_at_ms"`
	L1Ask       *decimal.Decimal `gorm:"column:leg_1_best_ask"`
	L2Ask       *decimal.Decimal `gorm:"column:leg_2_best_ask"`
	L3Ask       *decimal.Decimal `gorm:"column:leg_3_best_ask"`
	L1Books     string          `gorm:"column:leg_1_order_books"`
	L2Books     string          `gorm:"column:leg_2_order_books"`
	L3Books     string          `gorm:"column:leg_3_order_books"`
	L1Info      string          `gorm:"column:leg_1_info"`
	L2Info      string          `gorm:"column:leg_2_info"`
	L3Info      string          `gorm:"column:leg_3_info"`
}

func loadRows(db *gorm.DB, id uint, limit int, kinds string) []oppRow {
	q := db.Table("arb_binary_opportunity").
		Select(`id, opp_kind, map_key, total_cost, profit_space, arb_qty, arb_profit, snapshot_at_ms,
			leg_1_best_ask, leg_2_best_ask, leg_3_best_ask,
			leg_1_order_books, leg_2_order_books, leg_3_order_books,
			leg_1_info, leg_2_info, leg_3_info`)
	if id > 0 {
		q = q.Where("id=?", id)
	} else {
		q = q.Where(`snapshot_at_ms>0 AND leg_1_best_ask IS NOT NULL AND leg_1_info LIKE '%token%'`)
		if kinds != "" && !strings.EqualFold(kinds, "ALL") {
			q = q.Where("opp_kind LIKE ?", strings.TrimSpace(kinds)+"%")
		}
		q = q.Order("snapshot_at_ms DESC").Limit(limit)
	}
	var rows []oppRow
	_ = q.Scan(&rows).Error
	return rows
}

type legJSON struct {
	QuoteAtMs int64   `json:"quote_at_ms"`
	NetCost   float64 `json:"net_cost"`
	Books     []struct {
		Asks []book.Level `json:"asks"`
	} `json:"books"`
}

func parseLegBooks(raw string) legJSON {
	var p legJSON
	_ = json.Unmarshal([]byte(strings.TrimSpace(raw)), &p)
	return p
}

func jsonTopAsk(p legJSON) float64 {
	for _, b := range p.Books {
		for _, a := range b.Asks {
			if book.ValidPrice(a.Price) && a.Size > 0 {
				return a.Price
			}
		}
	}
	return 0
}

func checkInternal(o oppRow) ([]string, []legEvidence) {
	tc, _ := o.TotalCost.Float64()
	ps, _ := o.ProfitSpace.Float64()
	aq, _ := o.ArbQty.Float64()
	ap, _ := o.ArbProfit.Float64()

	var f []string
	base := 1.0
	if strings.Contains(o.OppKind, onextwo.OppKindNoTriple) {
		base = 2
	}
	if math.Abs(ps-(base-tc)) > 0.0003 {
		f = append(f, "profit_space")
	}
	if aq > 0 && math.Abs(ap-aq*ps) > 0.0003 {
		f = append(f, "arb_profit")
	}

	asks := []*decimal.Decimal{o.L1Ask, o.L2Ask, o.L3Ask}
	books := []string{o.L1Books, o.L2Books, o.L3Books}
	infos := []string{o.L1Info, o.L2Info, o.L3Info}
	nLeg := 2
	if strings.Contains(o.OppKind, "ONE_X2") {
		nLeg = 3
	}

	var legs []legEvidence
	var netSum float64
	netN := 0
	for i := 0; i < nLeg; i++ {
		p := parseLegBooks(books[i])
		st := 0.0
		if asks[i] != nil {
			st, _ = asks[i].Float64()
		}
		top := jsonTopAsk(p)
		ev := legEvidence{
			idx: i + 1, token: tokenForLeg(o.OppKind, i, infos[i]), quoteMs: p.QuoteAtMs,
			storedAsk: st, jsonTop: top, netCost: p.NetCost,
		}
		legs = append(legs, ev)
		if st > 0 && top > 0 && math.Abs(st-top) > 0.0003 {
			f = append(f, fmt.Sprintf("leg%d_ask", i+1))
		}
		if p.NetCost > 0 {
			netSum += p.NetCost
			netN++
		}
	}
	if netN == nLeg && math.Abs(netSum-tc) > 0.0005 {
		f = append(f, "net_cost_sum")
	}
	return f, legs
}

func checkWS(db *gorm.DB, legs []legEvidence, winMs int64, askTol float64) (hit, match int) {
	for i := range legs {
		L := &legs[i]
		if L.token == "" || L.quoteMs <= 0 || L.storedAsk <= 0 {
			continue
		}
		lo, hi := L.quoteMs-winMs, L.quoteMs+winMs
		type R struct {
			Asks string
			Ms   int64 `gorm:"column:snapshot_at_ms"`
		}
		var r R
		err := db.Table("market_order_book_ws").Select("order_book_asks, snapshot_at_ms").
			Where("token_id=? AND snapshot_at_ms BETWEEN ? AND ?", L.token, lo, hi).
			Order(fmt.Sprintf("ABS(snapshot_at_ms - %d)", L.quoteMs)).Limit(1).Scan(&r).Error
		if err != nil || strings.TrimSpace(r.Asks) == "" {
			continue
		}
		hit++
		top := topAskRaw(r.Asks)
		L.wsTop = top
		L.wsDeltaMs = r.Ms - L.quoteMs
		if top > 0 && closePrice(top, L.storedAsk, askTol) {
			match++
		}
	}
	return hit, match
}

func closePrice(a, b, relTol float64) bool {
	if a <= 0 || b <= 0 {
		return false
	}
	if math.Abs(a-b) <= 0.01 {
		return true
	}
	return math.Abs(a-b)/b <= relTol
}

func topAskRaw(raw string) float64 {
	var rows [][]string
	if json.Unmarshal([]byte(raw), &rows) == nil {
		for _, r := range rows {
			if len(r) < 1 {
				continue
			}
			var p float64
			fmt.Sscanf(r[0], "%f", &p)
			if book.ValidPrice(p) {
				return p
			}
		}
	}
	var ls []book.Level
	if json.Unmarshal([]byte(raw), &ls) == nil {
		for _, lv := range ls {
			if book.ValidPrice(lv.Price) {
				return lv.Price
			}
		}
	}
	return 0
}

func tokenForLeg(kind string, idx int, raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(kind, "BINARY") {
		var y struct{ YesTokenID string `json:"yes_token_id"` }
		var n struct{ NoTokenID string `json:"no_token_id"` }
		if idx == 0 {
			_ = json.Unmarshal([]byte(raw), &y)
			return strings.TrimSpace(y.YesTokenID)
		}
		_ = json.Unmarshal([]byte(raw), &n)
		return strings.TrimSpace(n.NoTokenID)
	}
	switch idx {
	case 0:
		var d struct{ DrawTokenID string `json:"draw_token_id"` }
		_ = json.Unmarshal([]byte(raw), &d)
		return strings.TrimSpace(d.DrawTokenID)
	case 1:
		var y struct{ YesTokenID string `json:"yes_token_id"` }
		_ = json.Unmarshal([]byte(raw), &y)
		return strings.TrimSpace(y.YesTokenID)
	case 2:
		var n struct{ NoTokenID string `json:"no_token_id"` }
		_ = json.Unmarshal([]byte(raw), &n)
		return strings.TrimSpace(n.NoTokenID)
	}
	return ""
}