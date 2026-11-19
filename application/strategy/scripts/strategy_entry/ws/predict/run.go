package predict

import (
	"context"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"go.uber.org/zap"
)

// Session 由 watch 提供换代与 Apply。
type Session struct {
	Begin    func() (context.Context, int)
	Apply    func(Message)
	Snapshot func(gen int) // 订阅前 REST 拉全量，WS 快照是 best-effort
}

// Run 维持 Predict orderbook WS，断线重连。
func Run(ctx context.Context, apiKey string, outcomes []persistententity.MarketOutcome, mapKey string, s Session) {
	ids := MarketIDs(outcomes)
	if len(ids) == 0 || strings.TrimSpace(apiKey) == "" {
		return
	}
	for {
		if ctx.Err() != nil {
			return
		}
		connCtx, gen := s.Begin()
		ws.NotifyReconnect(ctx, string(appcommon.PlatformPredict), mapKey, gen)
		s.Apply(Message{Gen: gen, Kind: "pause"})
		if ctx.Err() != nil {
			return
		}
		if s.Snapshot != nil {
			s.Snapshot(gen)
		}
		err := stream(connCtx, apiKey, mapKey, gen, ids, s.Apply)
		s.Apply(Message{Gen: gen, Kind: "pause"})
		if ctx.Err() != nil {
			return
		}
		if connCtx.Err() != nil {
			continue
		}
		notification.SendErrorLog(ctx, "strategy_entry-predict-disconnected",
			zap.String("map_key", mapKey), zap.Int("gen", gen), zap.Error(err))
		if !ws.Sleep(ctx, time.Second) {
			return
		}
	}
}

func stream(ctx context.Context, apiKey, mapKey string, gen int, ids []string, apply func(Message)) error {
	return Stream(ctx, apiKey, mapKey, ids, func(msg Message) error {
		msg.Gen = gen
		apply(msg)
		return nil
	})
}

// MarketIDs 本组 Predict 盘口 id（platform_market_outcome_id，REST market id）。
func MarketIDs(outcomes []persistententity.MarketOutcome) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, oc := range outcomes {
		if !strings.EqualFold(oc.Platform, appcommon.PlatformPredict) {
			continue
		}
		id := strings.TrimSpace(oc.PlatformMarketOutcomeID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
