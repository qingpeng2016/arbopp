package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/backtest"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/mysql"
)

func main() {
	mapKey := "v1|俱乐部友谊赛|clf-tur-jab-2026-11-17"
	if len(os.Args) > 1 {
		mapKey = os.Args[1]
	}
	_ = conf.NewCfg()
	db := boot.NewDBClient()
	ctx := context.Background()

	res, err := backtest.ScanMapKeyOpportunities(
		ctx, db,
		mysql.NewHistoryDataOrderBookImpl(db),
		mysql.NewMarketOutcomeImpl(db),
		mysql.NewMarketImpl(db),
		mysql.NewPlatformImpl(db),
		mysql.NewConfigStrategyThresholdImpl(db),
		mysql.NewConfigStrategyFeeImpl(db),
		mapKey,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("map_key=%s seconds=%d hit_seconds=%d total_hits=%d first_hit_sec=%d sample_per=%.4f\n",
		mapKey, res.Seconds, res.HitSeconds, res.TotalHits, res.FirstHitSec, res.SamplePer)
	if res.TotalHits == 0 {
		os.Exit(2)
	}
}