package orderbook

import (
	"context"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/fetchopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/running"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RunPoll 常驻：每秒读 stat_kv 在跑场次，并发拉盘口写 market_order_book。
func (j *OrderBook) RunPoll(ctx context.Context) {
	if j == nil {
		return
	}
	j.pollMu.Lock()
	if j.pollRun {
		j.pollMu.Unlock()
		return
	}
	loopCtx, stop := context.WithCancel(context.Background())
	j.pollRun = true
	j.pollStop = stop
	j.pollDone = make(chan struct{})
	j.pollMu.Unlock()
	go j.pollLoop(loopCtx)
}

// StopPoll 停止 RunPoll 循环。
func (j *OrderBook) StopPoll() {
	if j == nil {
		return
	}
	j.pollMu.Lock()
	stop := j.pollStop
	done := j.pollDone
	j.pollRun = false
	j.pollStop = nil
	j.pollMu.Unlock()
	if stop != nil {
		stop()
	}
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		notification.SendErrorLog(context.Background(), "orderbook-poll-stop-timeout")
	}
}

func (j *OrderBook) pollLoop(ctx context.Context) {
	defer close(j.pollDone)
	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	var cached running.Snapshot
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			j.pollTick(ctx, &cached)
		}
	}
}

func (j *OrderBook) pollTick(ctx context.Context, cached *running.Snapshot) {
	defer func() {
		if r := recover(); r != nil {
			notification.SendErrorLog(ctx, "orderbook-poll-Panic", zap.Any("panic", r))
		}
	}()
	if j.statRepo == nil || j.marketRepo == nil || j.outcomeRepo == nil {
		return
	}
	snap, err := running.LoadFromStatKV(ctx, j.statRepo, j.marketRepo, j.outcomeRepo)
	if err != nil && err != gorm.ErrRecordNotFound {
		notification.SendErrorLog(ctx, "orderbook-poll-stat-load-failed", zap.Error(err))
		return
	}
	if snap.Fingerprint != cached.Fingerprint {
		*cached = snap
		if len(snap.Groups) > 0 {
			logger.InfoZ(ctx, "orderbook-poll-targets-reloaded",
				zap.Int("groups", len(snap.Groups)),
				zap.Int("outcomes", len(snap.Outcomes)))
		}
	}
	if len(cached.Outcomes) == 0 {
		return
	}

	kalshiMarkets, polyMarkets, predictMarkets := splitOutcomesByPlatform(cached.Outcomes)
	fetchCfg := aggressiveFetchConfig(len(cached.Outcomes))
	stats := j.syncPlatformsConcurrent(ctx, cached.Fixtures, kalshiMarkets, polyMarkets, predictMarkets, fetchCfg)

	totalUpserted := stats.kalshi.upserted + stats.poly.upserted + stats.predict.upserted
	if totalUpserted > 0 {
		logger.DebugZ(ctx, "orderbook-poll-tick",
			zap.Int("upserted", totalUpserted),
			zap.Int("outcomes", len(cached.Outcomes)))
	}
	for _, st := range []platformSyncStat{stats.kalshi, stats.poly, stats.predict} {
		if st.platform == "" || st.err == nil {
			continue
		}
		notification.SendErrorLog(ctx, "orderbook-poll-platform-failed",
			zap.String("platform", st.platform),
			zap.Error(st.err))
	}
}

// aggressiveFetchConfig 每盘口一批、高并发，便于 1s 内尽量写全。
func aggressiveFetchConfig(outcomeCount int) fetchopt.FetchConfig {
	c := outcomeCount
	if c < 1 {
		c = 1
	}
	if c > 32 {
		c = 32
	}
	bf := func() fetchopt.PlatformFetch {
		return fetchopt.PlatformFetch{BatchSize: 1, BatchConcurrency: c}
	}
	return fetchopt.FetchConfig{
		Polymarket: bf(),
		Kalshi:     bf(),
		Predict:    bf(),
	}
}