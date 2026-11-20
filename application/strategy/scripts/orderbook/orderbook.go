package orderbook

import (
	"context"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/fetchopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/running"
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
	"gorm.io/gorm"
)

// OrderBook 拉取在跑场次订单簿，INSERT 历史快照到 market_order_book（批量 API + BatchCreate）。
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

// Run orderbook_sync：与 poll 相同，仅 stat_kv market_running_count_strategy.groups 非空时拉盘口。
func (j *OrderBook) Run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			notification.SendErrorLog(ctx, "orderbook_sync-Panic",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())))
		}
	}()
	j.syncRunningStatOnce(ctx, "orderbook_sync")
}

func (j *OrderBook) syncRunningStatOnce(ctx context.Context, logPrefix string) {
	if j.outcomeRepo == nil || j.marketRepo == nil || j.statRepo == nil || j.kalshiSyncer == nil || j.kalshiSyncer.Books == nil {
		notification.SendErrorLog(ctx, logPrefix+"-skipped", zap.String("reason", "repo nil"))
		return
	}
	snap, err := running.LoadFromStatKV(ctx, j.statRepo, j.marketRepo, j.outcomeRepo)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.InfoZ(ctx, logPrefix+"-no-running-stat")
			return
		}
		notification.SendErrorLog(ctx, logPrefix+"-stat-load-failed", zap.Error(err))
		return
	}
	if len(snap.Groups) == 0 || len(snap.Outcomes) == 0 {
		logger.InfoZ(ctx, logPrefix+"-no-running-groups")
		return
	}
	rows := snap.Outcomes
	kalshiMarkets, polyMarkets, predictMarkets := splitOutcomesByPlatform(rows)
	fetchCfg := j.resolveFetchConfig(ctx)
	stats := j.syncPlatformsConcurrent(ctx, snap.Fixtures, kalshiMarkets, polyMarkets, predictMarkets, fetchCfg)

	for _, st := range []platformSyncStat{stats.kalshi, stats.poly, stats.predict} {
		if st.platform == "" || st.err == nil {
			continue
		}
		notification.SendErrorLog(ctx, logPrefix+"-platform-failed",
			zap.String("platform", st.platform),
			zap.Error(st.err))
	}

	logger.InfoZ(ctx, logPrefix+"-completed",
		zap.Int("groups", len(snap.Groups)),
		zap.Int("outcomes", len(rows)),
		zap.Int("kalshi_targets", len(kalshiMarkets)),
		zap.Int("kalshi_upserted", stats.kalshi.upserted),
		zap.Int("poly_targets", len(polyMarkets)),
		zap.Int("poly_upserted", stats.poly.upserted),
		zap.Int("predict_targets", len(predictMarkets)),
		zap.Int("predict_upserted", stats.predict.upserted))
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
