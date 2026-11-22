// 对比 market_order_book_ws 与 SX 当前 REST 快照（或仅罗列 Poly/Predict 顶价）。
//
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/verify_ws_book -samples=6
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	sxbetws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/sxbet"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/mysql"
)

func main() {
	samples := flag.Int("samples", 6, "per platform max rows to compare")
	hours := flag.Int("hours", 48, "lookback hours for latest rows; 0=all")
	flag.Parse()
	_ = conf.NewCfg()
	db := boot.NewDBClient()

	type row struct {
		ID       uint
		Platform string
		Hash     string
		URL      string
		Outcome  string
		Token    string
		Asks     string
		Bids     string
		SnapMs   int64
		Updated  string
	}

	var q string
	if *hours <= 0 {
		q = `SELECT id, platform, platform_market_outcome_id, market_url,
		COALESCE(outcome_name,''), COALESCE(token_name,''),
		order_book_asks, order_book_bids, snapshot_at_ms, updated_at
		FROM market_order_book_ws
		ORDER BY id DESC
		LIMIT 2000`
	} else {
		q = fmt.Sprintf(`SELECT id, platform, platform_market_outcome_id, market_url,
		COALESCE(outcome_name,''), COALESCE(token_name,''),
		order_book_asks, order_book_bids, snapshot_at_ms, updated_at
		FROM market_order_book_ws
		WHERE updated_at > NOW() - INTERVAL %d HOUR
		ORDER BY updated_at DESC
		LIMIT 2000`, *hours)
	}
	rws, err := db.Raw(q).Rows()
	if err != nil {
		fmt.Fprintf(os.Stderr, "query: %v\n", err)
		os.Exit(1)
	}
	defer rws.Close()

	var total int64
	_ = db.Raw(`SELECT COUNT(*) FROM market_order_book_ws`).Scan(&total)
	fmt.Printf("market_order_book_ws total rows: %d\n", total)

	by := map[string][]row{}
	seen := map[string]map[string]bool{}
	for rws.Next() {
		var x row
		var tok, out sql.NullString
		var snap sql.NullInt64
		var upd sql.NullTime
		if err := rws.Scan(&x.ID, &x.Platform, &x.Hash, &x.URL, &out, &tok, &x.Asks, &x.Bids, &snap, &upd); err != nil {
			continue
		}
		if tok.Valid {
			x.Token = tok.String
		}
		if out.Valid {
			x.Outcome = out.String
		}
		if snap.Valid {
			x.SnapMs = snap.Int64
		}
		if upd.Valid {
			x.Updated = upd.Time.UTC().Format(time.RFC3339)
		}
		p := strings.ToUpper(strings.TrimSpace(x.Platform))
		if x.Hash == "" {
			continue
		}
		if seen[p] == nil {
			seen[p] = map[string]bool{}
		}
		if seen[p][x.Hash] {
			continue
		}
		if len(by[p]) >= *samples {
			continue
		}
		seen[p][x.Hash] = true
		by[p] = append(by[p], x)
	}

	catalog, err := boot.NewPlatformCatalog(mysql.NewPlatformImpl(db))
	if err != nil {
		fmt.Fprintf(os.Stderr, "catalog: %v\n", err)
		os.Exit(1)
	}
	sxbetAPI := boot.NewSxbetRepoFromPlatform(catalog, mysql.NewAccountImpl(db))
	hc := &http.Client{Timeout: 20 * time.Second}
	var okN, warnN, badN int
	for _, plat := range []string{"SXBET", "POLYMARKET", "PREDICT", "KALSHI"} {
		rows := by[plat]
		if len(rows) == 0 {
			continue
		}
		fmt.Printf("\n======== %s (%d) ========\n", plat, len(rows))
		for _, x := range rows {
			dbAsks := parseLevels(x.Asks, true)
			dbBids := parseLevels(x.Bids, false)
			fmt.Printf("\nid=%d | %s\n", x.ID, trunc(x.Outcome, 72))
			fmt.Printf("url: %s\n", x.URL)
			fmt.Printf("updated=%s | snap_age=%s\n", x.Updated, snapAge(x.SnapMs))
			printTop("DB", dbAsks, dbBids)

			switch plat {
			case "SXBET":
				apiAsks, apiBids, err := sxbetSnapshotViaRepo(context.Background(), sxbetAPI, x.Hash)
				if err != nil {
					apiAsks, apiBids, err = sxbetSnapshot(hc, x.Hash)
				}
				if err != nil {
					fmt.Printf("REST: %v\n", err)
					badN++
					continue
				}
				printTop("API now", apiAsks, apiBids)
				a, w, b := judge(dbAsks, dbBids, apiAsks, apiBids, x.SnapMs)
				okN += a
				warnN += w
				badN += b
			default:
				fmt.Printf("(live REST not in this tool; DB top-of-book only)\n")
			}
		}
	}
	fmt.Printf("\n=== summary: ok=%d warn(stale/small diff)=%d bad=%d ===\n", okN, warnN, badN)
}

type lvl struct{ P, S float64 }

func parseLevels(js string, askSide bool) []lvl {
	var raw [][]string
	if json.Unmarshal([]byte(js), &raw) != nil {
		return nil
	}
	out := make([]lvl, 0, len(raw))
	for _, r := range raw {
		if len(r) < 2 {
			continue
		}
		p, _ := strconv.ParseFloat(r[0], 64)
		s, _ := strconv.ParseFloat(r[1], 64)
		out = append(out, lvl{p, s})
	}
	if askSide {
		sort.Slice(out, func(i, j int) bool { return out[i].P < out[j].P })
	} else {
		sort.Slice(out, func(i, j int) bool { return out[i].P > out[j].P })
	}
	return out
}

func sxbetSnapshot(hc *http.Client, hash string) (asks, bids []lvl, err error) {
	url := "https://api.sx.bet/orderbook-v3/snapshot/market?marketHash=" + hash
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var ob struct {
		Data struct {
			OutcomeOne []map[string]string `json:"outcomeOne"`
			OutcomeTwo []map[string]string `json:"outcomeTwo"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &ob); err != nil {
		return nil, nil, err
	}
	const denom = 1e20
	for _, lv := range ob.Data.OutcomeOne {
		m := parseSXOdds(lv["percentageOdds"], denom)
		p := 1 - m
		s := parseSXSize(lv["size"])
		if validPrice(p) && s > 0 {
			asks = append(asks, lvl{p, s})
		}
	}
	for _, lv := range ob.Data.OutcomeTwo {
		p := parseSXOdds(lv["percentageOdds"], denom)
		s := parseSXSize(lv["size"])
		if validPrice(p) && s > 0 {
			bids = append(bids, lvl{p, s})
		}
	}
	sort.Slice(asks, func(i, j int) bool { return asks[i].P < asks[j].P })
	sort.Slice(bids, func(i, j int) bool { return bids[i].P > bids[j].P })
	return asks, bids, nil
}

func sxbetSnapshotViaRepo(ctx context.Context, api httprepo.SxbetRepo, hash string) (asks, bids []lvl, err error) {
	if api == nil {
		return nil, nil, fmt.Errorf("nil api")
	}
	snap, err := api.GetOrderbookSnapshot(ctx, hash)
	if err != nil {
		return nil, nil, err
	}
	msg, ok := sxbetws.MessageFromSnapshot(1, snap)
	if !ok {
		return nil, nil, fmt.Errorf("empty book")
	}
	for _, lv := range msg.Asks {
		asks = append(asks, lvl{lv.Price, lv.Size})
	}
	for _, lv := range msg.Bids {
		bids = append(bids, lvl{lv.Price, lv.Size})
	}
	sort.Slice(asks, func(i, j int) bool { return asks[i].P < asks[j].P })
	sort.Slice(bids, func(i, j int) bool { return bids[i].P > bids[j].P })
	return asks, bids, nil
}

func parseSXOdds(s string, denom float64) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return math.NaN()
	}
	return f / denom
}

func parseSXSize(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f / 1e6
}

func validPrice(p float64) bool { return p > 0 && p < 1 && !math.IsNaN(p) }

func printTop(label string, asks, bids []lvl) {
	if len(asks) > 0 {
		fmt.Printf("  %s best ask: p=%.5f decimal~%.3f size=%.2f\n", label, asks[0].P, 1/asks[0].P, asks[0].S)
	} else {
		fmt.Printf("  %s asks: empty\n", label)
	}
	if len(bids) > 0 {
		fmt.Printf("  %s best bid: p=%.5f decimal~%.3f size=%.2f\n", label, bids[0].P, 1/bids[0].P, bids[0].S)
	} else {
		fmt.Printf("  %s bids: empty\n", label)
	}
}

func judge(dbA, dbB, apiA, apiB []lvl, snapMs int64) (ok, warn, bad int) {
	age := time.Duration(0)
	if snapMs > 0 {
		age = time.Since(time.UnixMilli(snapMs))
	}
	if len(dbA) == 0 && len(dbB) == 0 {
		fmt.Printf("  verdict: EMPTY book in DB\n")
		return 0, 0, 1
	}
	da, aa := math.NaN(), math.NaN()
	if len(dbA) > 0 {
		da = dbA[0].P
	}
	if len(apiA) > 0 {
		aa = apiA[0].P
	}
	db, ab := math.NaN(), math.NaN()
	if len(dbB) > 0 {
		db = dbB[0].P
	}
	if len(apiB) > 0 {
		ab = apiB[0].P
	}
	tol := 0.025
	if age > 2*time.Minute {
		tol = 0.05
	}
	if age > 10*time.Minute {
		tol = 0.08
	}
	askOK := len(dbA) == 0 && len(apiA) == 0 || (!math.IsNaN(da) && !math.IsNaN(aa) && math.Abs(da-aa) <= tol)
	bidOK := len(dbB) == 0 && len(apiB) == 0 || (!math.IsNaN(db) && !math.IsNaN(ab) && math.Abs(db-ab) <= tol)
	switch {
	case askOK && bidOK:
		fmt.Printf("  verdict: OK (tol=%.2f age=%s)\n", tol, age.Round(time.Second))
		return 1, 0, 0
	case (askOK || bidOK) && age > time.Minute:
		fmt.Printf("  verdict: WARN partial/stale askΔ=%.4f bidΔ=%.4f\n", da-aa, db-ab)
		return 0, 1, 0
	default:
		fmt.Printf("  verdict: MISMATCH ask db=%.4f api=%.4f | bid db=%.4f api=%.4f\n", da, aa, db, ab)
		return 0, 0, 1
	}
}

func snapAge(ms int64) string {
	if ms <= 0 {
		return "n/a"
	}
	return time.Since(time.UnixMilli(ms)).Round(time.Second).String() + " ago"
}

func trunc(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}