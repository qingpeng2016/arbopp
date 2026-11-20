package orderbook

import (
	"context"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/fetchopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/kalshi"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/polymarket"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/predict"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"go.uber.org/zap"
)

// OrderBook 拉取已核对市场的订单簿并写入 market_order_book（批量 API + BatchUpsert）。
type OrderBook struct {
	configStrategyRepo persistentrepo.ConfigStrategyRepo
	platformRepo       persistentrepo.PlatformRepo
	marketRepo         persistentrepo.MarketRepo
	outcomeRepo        persistentrepo.MarketOutcomeRepo
	statRepo           persistentrepo.StatKVRepo
	kalshiSyncer       *kalshi.Syncer
	polySyncer         *polymarket.Syncer
	predictSyncer      *predict.Syncer

	pollMu   sync.Mutex
	pollRun  bool
	pollStop context.CancelFunc
	pollDone chan struct{}
}

func NewOrderBook(
	kalshiRepo httprepo.KalshiRepo,
	polyRepo httprepo.PolymarketRepo,
	predictRepo httprepo.PredictRepo,
	marketRepo persistentrepo.MarketRepo,
	outcomeRepo persistentrepo.MarketOutcomeRepo,
	orderBookRepo persistentrepo.OrderBookRepo,
	configStrategyRepo persistentrepo.ConfigStrategyRepo,
	platformRepo persistentrepo.PlatformRepo,
	statRepo persistentrepo.StatKVRepo,
) *OrderBook {
	return &OrderBook{
		configStrategyRepo: configStrategyRepo,
		platformRepo:       platformRepo,
		marketRepo:         marketRepo,
		outcomeRepo:        outcomeRepo,
		statRepo:           statRepo,
		kalshiSyncer: &kalshi.Syncer{
			API:   kalshiRepo,
			Books: orderBookRepo,
		},
		polySyncer: &polymarket.Syncer{
			API:   polyRepo,
			Books: orderBookRepo,
		},
		predictSyncer: &predict.Syncer{
			API:   predictRepo,
			Books: orderBookRepo,
		},
	}
}

// Run orderbook_sync：config_strategy.watch_map_keys 指定场次下的 ACTIVE 盘口，按平台批量拉盘口并 upsert。
func (j *OrderBook) Run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			notification.SendErrorLog(ctx, "orderbook_sync-Panic",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())))
		}
	}()
	if j.outcomeRepo == nil || j.marketRepo == nil || j.kalshiSyncer == nil || j.kalshiSyncer.Books == nil {
		notification.SendErrorLog(ctx, "orderbook_sync-skipped", zap.String("reason", "repo nil"))
		return
	}

	fixtures, err := marketmap.LoadVerifiedFixturesNew(ctx, j.marketRepo, j.configStrategyRepo, j.platformRepo)
	if err != nil {
		notification.SendErrorLog(ctx, "orderbook_sync-ListFixturesFailed", zap.Error(err))
		return
	}
	_, raw, err := j.outcomeRepo.List(ctx, nil, nil, "platform asc, id asc", "")
	if err != nil {
		notification.SendErrorLog(ctx, "orderbook_sync-ListMarketsFailed", zap.Error(err))
		return
	}
	rows := marketmap.FilterOutcomesUnderVerifiedFixtures(raw, fixtures)
	kalshiMarkets, polyMarkets, predictMarkets := splitOutcomesByPlatform(rows)
	if len(kalshiMarkets) == 0 && len(polyMarkets) == 0 && len(predictMarkets) == 0 {
		logger.InfoZ(ctx, "orderbook_sync-no-markets",
			zap.Int("scanned", len(rows)))
		return
	}

	fetchCfg := j.resolveFetchConfig(ctx)
	stats := j.syncPlatformsConcurrent(ctx, fixtures, kalshiMarkets, polyMarkets, predictMarkets, fetchCfg)

	for _, st := range []platformSyncStat{stats.kalshi, stats.poly, stats.predict} {
		if st.platform == "" || st.err == nil {
			continue
		}
		notification.SendErrorLog(ctx, "orderbook_sync-PlatformFailed",
			zap.String("platform", st.platform),
			zap.Error(st.err))
	}

	logger.InfoZ(ctx, "orderbook_sync-Completed",
		zap.Int("markets_scanned", len(rows)),
		zap.Int("kalshi_targets", len(kalshiMarkets)),
		zap.Int("kalshi_upserted", stats.kalshi.upserted),
		zap.Int("kalshi_missed", stats.kalshi.missed),
		zap.Int("poly_targets", len(polyMarkets)),
		zap.Int("poly_upserted", stats.poly.upserted),
		zap.Int("poly_missed", stats.poly.missed),
		zap.Int("predict_targets", len(predictMarkets)),
		zap.Int("predict_upserted", stats.predict.upserted),
		zap.Int("predict_missed", stats.predict.missed))
}

type platformSyncStat struct {
	platform         string
	upserted, missed int
	err              error
}

type orderbookPlatformTotals struct {
	kalshi, poly, predict platformSyncStat
}

func (j *OrderBook) syncPlatformsConcurrent(ctx context.Context, fixtures map[uint]persistententity.Market, kalshiMarkets, polyMarkets, predictMarkets []persistententity.MarketOutcome, fetchCfg fetchopt.FetchConfig) orderbookPlatformTotals {
	type job struct {
		platform string
		run      func(context.Context) (int, int, error)
	}
	var jobs []job
	if len(kalshiMarkets) > 0 {
		mk := kalshiMarkets
		syncer := j.kalshiSyncer
		jobs = append(jobs, job{
			platform: string(cryptoPredictionEnum.PlatformKalshi),
			run:      func(c context.Context) (int, int, error) { return syncer.Sync(c, mk, fixtures, fetchCfg.Kalshi) },
		})
	}
	if len(polyMarkets) > 0 {
		mp := polyMarkets
		syncer := j.polySyncer
		jobs = append(jobs, job{
			platform: string(cryptoPredictionEnum.PlatformPolymarket),
			run:      func(c context.Context) (int, int, error) { return syncer.Sync(c, mp, fixtures, fetchCfg.Polymarket) },
		})
	}
	if len(predictMarkets) > 0 {
		md := predictMarkets
		syncer := j.predictSyncer
		jobs = append(jobs, job{
			platform: string(cryptoPredictionEnum.PlatformPredict),
			run:      func(c context.Context) (int, int, error) { return syncer.Sync(c, md, fixtures, fetchCfg.Predict) },
		})
	}

	out := make([]platformSyncStat, len(jobs))
	var wg sync.WaitGroup
	wg.Add(len(jobs))
	for i := range jobs {
		i := i
		go func() {
			defer wg.Done()
			u, m, err := jobs[i].run(ctx)
			out[i] = platformSyncStat{
				platform: jobs[i].platform,
				upserted: u,
				missed:   m,
				err:      err,
			}
		}()
	}
	wg.Wait()

	totals := orderbookPlatformTotals{}
	for _, st := range out {
		switch strings.ToUpper(st.platform) {
		case string(cryptoPredictionEnum.PlatformKalshi):
			totals.kalshi = st
		case string(cryptoPredictionEnum.PlatformPolymarket):
			totals.poly = st
		case string(cryptoPredictionEnum.PlatformPredict):
			totals.predict = st
		}
	}
	return totals
}

// RunProbe 单市场探针（bot task: orderbook），仅打日志不写库。
func (j *OrderBook) RunProbe(ctx context.Context) {
	if j.kalshiSyncer != nil {
		kalshi.RunProbe(ctx, j.kalshiSyncer.API)
	}
	if j.polySyncer != nil {
		polymarket.RunProbe(ctx, j.polySyncer.API)
	}
}
