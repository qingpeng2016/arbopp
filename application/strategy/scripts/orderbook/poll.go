package orderbook

import (
	"context"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/running"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// pollSleepAfterTick 每轮拉盘后休眠，避免打满 API/DB。
const pollSleepAfterTick = 700 * time.Millisecond

// RunPoll 常驻：读 stat_kv 在跑场次，并发拉盘口写 market_order_book。
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
	var cached running.Snapshot
	for {
		if ctx.Err() != nil {
			return
		}
		j.pollTick(ctx, &cached)
		select {
		case <-ctx.Done():
			return
		case <-time.After(pollSleepAfterTick):
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
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			*cached = running.Snapshot{}
			return
		}
		notification.SendErrorLog(ctx, "orderbook-poll-stat-load-failed", zap.Error(err))
		return
	}
	if cached.Fingerprint != snap.Fingerprint && len(snap.Groups) > 0 {
		logger.InfoZ(ctx, "orderbook-poll-targets-reloaded",
			zap.Int("groups", len(snap.Groups)),
			zap.Int("outcomes", len(snap.Outcomes)))
	}
	*cached = snap
	if len(cached.Groups) == 0 || len(cached.Outcomes) == 0 {
		return
	}

	kalshiMarkets, polyMarkets, predictMarkets := splitOutcomesByPlatform(cached.Outcomes)
	// 与 orderbook_sync 相同：每请求多盘口（Poly /books、Kalshi orderbooks、Predict orderbooks），批间并发
	fetchCfg := j.resolveFetchConfig(ctx)
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