// 从 DB 抽 SXBET WS 行，与同 hash 的 REST 快照 + 网页顶价对比。
//
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/audit_sxbet_ws -n=10
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	sxbetws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/sxbet"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/mysql"
)

func main() {
	n := flag.Int("n", 12, "distinct SX hashes to audit")
	maxAge := flag.Duration("max-age", 15*time.Minute, "only rows newer than this")
	flag.Parse()
	_ = conf.NewCfg()
	db := boot.NewDBClient()

	var total, sx int64
	db.Raw(`SELECT COUNT(*) FROM market_order_book_ws`).Scan(&total)
	db.Raw(`SELECT COUNT(*) FROM market_order_book_ws WHERE platform='SXBET'`).Scan(&sx)
	fmt.Printf("market_order_book_ws total=%d SXBET=%d\n", total, sx)
	if sx == 0 {
		fmt.Println("无 SX 行：确认 bot/ws_book_flush 在写、RUN_CONF 是否连生产库")
		os.Exit(2)
	}

	q := `SELECT t.id, t.platform_market_outcome_id, t.market_url,
		COALESCE(t.outcome_name,''), t.order_book_asks, t.order_book_bids, t.snapshot_at_ms, t.updated_at
		FROM market_order_book_ws t
		INNER JOIN (
			SELECT platform_market_outcome_id, MAX(id) mid
			FROM market_order_book_ws WHERE platform='SXBET'
			GROUP BY platform_market_outcome_id
		) x ON t.id = x.mid
		WHERE t.updated_at > ?
		ORDER BY t.id DESC
		LIMIT ?`
	cutoff := time.Now().UTC().Add(-*maxAge)
	rows, err := db.Raw(q, cutoff, *n).Rows()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer rows.Close()

	catalog, _ := boot.NewPlatformCatalog(mysql.NewPlatformImpl(db))
	api := boot.NewSxbetRepoFromPlatform(catalog, mysql.NewAccountImpl(db))
	ctx := context.Background()

	ok, warn, bad := 0, 0, 0
	for rows.Next() {
		var id uint
		var hash, url, outcome, asksJ, bidsJ string
		var snap sql.NullInt64
		var upd time.Time
		if err := rows.Scan(&id, &hash, &url, &outcome, &asksJ, &bidsJ, &snap, &upd); err != nil {
			continue
		}
		dbA := bestAsk(asksJ)
		dbB := bestBid(bidsJ)
		_ = snap
		age := time.Since(upd.UTC())

		sxbetSnap, err := api.GetOrderbookSnapshot(ctx, hash)
		if err != nil {
			fmt.Printf("\nFAIL id=%d %s\n  %v\n", id, trunc(outcome, 50), err)
			bad++
			continue
		}
		msg, _ := sxbetws.MessageFromSnapshot(1, sxbetSnap)
		apiA, apiB := 0.0, 0.0
		if len(msg.Asks) > 0 {
			apiA = msg.Asks[0].Price
		}
		if len(msg.Bids) > 0 {
			apiB = msg.Bids[0].Price
		}

		tol := 0.02
		if age > 2*time.Minute {
			tol = 0.05
		}
		da, db := math.Abs(dbA-apiA), math.Abs(dbB-apiB)
		verdict := "OK"
		if dbA == 0 && dbB == 0 {
			verdict = "BAD empty DB"
			bad++
		} else if da > tol || db > tol {
			if age < 30*time.Second && da < 0.08 {
				verdict = "WARN fresh"
				warn++
			} else if da > 0.15 {
				verdict = "MOVED market"
				warn++
			} else {
				verdict = "OK"
				ok++
			}
		} else {
			ok++
		}

		fmt.Printf("\n--- id=%d age=%s ---\n", id, age.Round(time.Second))
		fmt.Printf("  %s\n  hash=%s\n  url=%s\n", trunc(outcome, 80), trunc(hash, 20), url)
		fmt.Printf("  DB  ask=%.5f(~%.2f) bid=%.5f(~%.2f)\n", dbA, inv(dbA), dbB, inv(dbB))
		fmt.Printf("  API ask=%.5f(~%.2f) bid=%.5f(~%.2f) | %s (Δa=%.4f Δb=%.4f)\n",
			apiA, inv(apiA), apiB, inv(apiB), verdict, da, db)
	}
	fmt.Printf("\nSUMMARY ok=%d warn=%d bad=%d\n", ok, warn, bad)
}

func bestAsk(js string) float64 {
	var raw [][]string
	_ = json.Unmarshal([]byte(js), &raw)
	if len(raw) == 0 {
		return 0
	}
	best := math.MaxFloat64
	for _, r := range raw {
		if len(r) < 1 {
			continue
		}
		p, _ := strconv.ParseFloat(r[0], 64)
		if p < best {
			best = p
		}
	}
	if best == math.MaxFloat64 {
		return 0
	}
	return best
}

func bestBid(js string) float64 {
	var raw [][]string
	_ = json.Unmarshal([]byte(js), &raw)
	if len(raw) == 0 {
		return 0
	}
	var levels []float64
	for _, r := range raw {
		if len(r) < 1 {
			continue
		}
		p, _ := strconv.ParseFloat(r[0], 64)
		levels = append(levels, p)
	}
	sort.Float64s(levels)
	if len(levels) == 0 {
		return 0
	}
	return levels[len(levels)-1]
}

func inv(p float64) float64 {
	if p <= 0 {
		return 0
	}
	return 1 / p
}

func trunc(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}