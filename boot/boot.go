package boot

import (
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/backtest"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/historydata"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/historydata/synthetic"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderfillsync"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/statcollect"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/taskalert"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/wsbookflush"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/wsbookqueue"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/constants"
	log2 "github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	redisclient "github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/redis"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/mysql"
	"github.com/gph-tech/fgmm-strategy-bitfinex/interfaces/rest"

	"github.com/redis/go-redis/v9"
	"go.uber.org/dig"
	"go.uber.org/zap/zapcore"
)

func init() {
	time.Local = constants.UtcLocation
	log2.NewLogger("fgmm", "./log", zapcore.DebugLevel)
}

func BuildContainer() *dig.Container {
	c := dig.New()

	cfg := conf.NewCfg()
	_ = c.Provide(func() *conf.Config {
		return cfg
	})
	_ = c.Provide(rest.NewRouter)

	_ = c.Provide(taskalert.NewTaskAlert)
	_ = c.Provide(orderbook.NewOrderBook)
	_ = c.Provide(historydata.NewSync)
	_ = c.Provide(synthetic.NewSync)
	_ = c.Provide(backtest.NewReplay)
	_ = c.Provide(strategyentry.NewEntry)
	_ = c.Provide(marketsync.NewMarketSync)
	_ = c.Provide(marketmap.NewMarketMap)
	_ = c.Provide(statcollect.NewCollect)
	_ = c.Provide(wsbookflush.NewFlush)
	_ = c.Provide(orderfillsync.NewSync)
	_ = c.Provide(strategy.NewScheduler)
	_ = c.Provide(strategy.NewEntry)

	_ = c.Provide(func(cfg *conf.Config) redis.UniversalClient {
		return redisclient.NewRedisClient(cfg.RedisConf)
	})
	_ = c.Provide(wsbookqueue.New)
	_ = c.Provide(NewDBClient)
	_ = c.Provide(mysql.NewPlatformImpl)
	_ = c.Provide(mysql.NewAccountImpl)
	_ = c.Provide(mysql.NewConfigStrategyThresholdImpl)
	_ = c.Provide(mysql.NewConfigStrategyFeeImpl)
	_ = c.Provide(mysql.NewConfigStrategyImpl)
	_ = c.Provide(mysql.NewStatKVImpl)
	_ = c.Provide(mysql.NewConfigMarketCategoryImpl)
	_ = c.Provide(mysql.NewBotScheduleConfigImpl)
	_ = c.Provide(mysql.NewMarketImpl)
	_ = c.Provide(mysql.NewMarketOutcomeImpl)
	_ = c.Provide(mysql.NewOrderBookImpl)
	_ = c.Provide(mysql.NewOrderBookWSImpl)
	_ = c.Provide(mysql.NewDataPlatformImpl)
	_ = c.Provide(mysql.NewHistoryDataOrderBookImpl)
	_ = c.Provide(mysql.NewHistoryMarketEventImpl)
	_ = c.Provide(mysql.NewDataTaskImpl)
	_ = c.Provide(mysql.NewBacktestTaskImpl)
	_ = c.Provide(mysql.NewBacktestArbBinaryOpportunityImpl)
	_ = c.Provide(mysql.NewArbBinaryOpportunityImpl)
	_ = c.Provide(mysql.NewArbBinaryOrderImpl)
	_ = c.Provide(mysql.NewBacktestOrderImpl)

	_ = c.Provide(http.NewHttpClient)
	_ = c.Provide(NewPlatformCatalog)
	_ = c.Provide(NewKalshiRepoFromPlatform)
	_ = c.Provide(NewPolymarketRepoFromPlatform)
	_ = c.Provide(NewPredictRepoFromPlatform)
	_ = c.Provide(NewSxbetRepoFromPlatform)

	_ = c.Provide(notification.NewLarkClient)
	_ = c.Provide(notification.NewNotificationManager)

	return c
}
