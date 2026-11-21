package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	sxbetws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/sxbet"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/mysql"
)

func main() {
	_ = conf.NewCfg()
	db := boot.NewDBClient()
	var ws, hist int64
	_ = db.Raw("SELECT COUNT(*) FROM market_order_book_ws").Scan(&ws)
	_ = db.Raw("SELECT COUNT(*) FROM market_order_book_ws_history").Scan(&hist)
	cfg := conf.NewCfg()
	dn := ""
	if cfg.MysqlMasterConf != nil {
		dn = cfg.MysqlMasterConf.DbName
	}
	fmt.Printf("DB %s: market_order_book_ws=%d history=%d\n", dn, ws, hist)

	catalog, _ := boot.NewPlatformCatalog(mysql.NewPlatformImpl(db))
	api := boot.NewSxbetRepoFromPlatform(catalog, mysql.NewAccountImpl(db))
	ctx := context.Background()

	type stored struct {
		label     string
		hash      string
		askP, bidP float64
	}
	// 来自你贴的 15:01 WS 快照（Kilmarnock 场）
	samples := []stored{
		{"Kilmarnock vs Not Kilmarnock [1X2]", "0xe6d701bab95a272482565271817f6e02a05d25b7570f8f5aeceb049cce9ec28f", 0.33375, 0.43875},
		{"Aberdeen vs Not Aberdeen [1X2]", "0x36886fb6a5c9485f219a06f6408fb7982c0cdb4f2b74dcdb05970cbf4876fc1f", 0.29375, 0.395},
		{"Moneyline", "0x2eb23e35bd6c43ddd75d9d3e040691fe4326a795dbb183d3905ce8ea97d71924", 0.47, 0.6},
		{"Over 3.0 vs Under 3.0", "0x3f4b9043b373412ab272be107fb8646b7e0b94d472fee3ba0847e52584262d8d", 0.45375, 0.48875},
	}

	for _, s := range samples {
		snap, err := api.GetOrderbookSnapshot(ctx, s.hash)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s snapshot err: %v\n", s.label, err)
			continue
		}
		msg, ok := sxbetws.MessageFromSnapshot(1, snap)
		if !ok {
			fmt.Printf("%s: empty book\n", s.label)
			continue
		}
		apiAsk, apiBid := 0.0, 0.0
		if len(msg.Asks) > 0 {
			apiAsk = msg.Asks[0].Price
		}
		if len(msg.Bids) > 0 {
			apiBid = msg.Bids[0].Price
		}
		fmt.Printf("\n%s\n", s.label)
		fmt.Printf("  stored@15:01 ask=%.5f (dec %.2f) bid=%.5f (dec %.2f)\n",
			s.askP, 1/s.askP, s.bidP, 1/s.bidP)
		fmt.Printf("  API now    ask=%.5f (dec %.2f) bid=%.5f (dec %.2f)\n",
			apiAsk, inv(apiAsk), apiBid, inv(apiBid))
		fmt.Printf("  Δask=%.4f Δbid=%.4f → %s\n",
			apiAsk-s.askP, apiBid-s.bidP, verdict(apiAsk-s.askP, apiBid-s.bidP))
	}
}

func inv(p float64) float64 {
	if p <= 0 {
		return 0
	}
	return 1 / p
}

func verdict(da, db float64) string {
	if abs(da) <= 0.03 && abs(db) <= 0.03 {
		return "仍接近（几秒内快照可对上）"
	}
	if abs(da) <= 0.15 || abs(db) <= 0.15 {
		return "盘口已走动（比分/时间变化正常，不是存错）"
	}
	return "价差大，建议看当时 snapshot_at_ms 是否盘中剧变"
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}