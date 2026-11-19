package main

import (
	"context"
	"fmt"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/statcollect"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/mysql"
)

func main() {
	_ = conf.NewCfg()
	db := boot.NewDBClient()
	statRepo := mysql.NewStatKVImpl(db)
	c := statcollect.NewCollect(
		db,
		statRepo,
		mysql.NewMarketImpl(db),
		mysql.NewMarketOutcomeImpl(db),
		mysql.NewConfigStrategyImpl(db),
		mysql.NewPlatformImpl(db),
	)
	c.Run(context.Background())
	fmt.Println("stat_kv refreshed (map_key_audit + market_count + stat_meta)")
}