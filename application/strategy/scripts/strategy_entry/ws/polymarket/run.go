package polymarket

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

// Session 由这组市场提供：换代，以及一条推送调用一次 Apply。
type Session struct {
	Begin func() (context.Context, int)
	Apply func(Message)
}

// Run 盘口等订阅后的 book 全量。断线后重新订阅。ctx 取消时返回。
func Run(ctx context.Context, assets []string, mapKey string, s Session) {
	for {
		if ctx.Err() != nil {
			return
		}
		connCtx, gen := s.Begin()
		ws.NotifyReconnect(ctx, string(appcommon.PlatformPolymarket), mapKey, gen)
		s.Apply(Message{Gen: gen, Kind: "pause"})
		if ctx.Err() != nil {
			return
		}
		err := stream(connCtx, mapKey, gen, assets, s.Apply)
		s.Apply(Message{Gen: gen, Kind: "pause"})
		if ctx.Err() != nil {
			return
		}
		if connCtx.Err() != nil {
			continue
		}
		notification.SendErrorLog(ctx, "strategy_entry-poly-disconnected", zap.String("map_key", mapKey), zap.Int("gen", gen), zap.Error(err))
		if !ws.Sleep(ctx, time.Second) {
			return
		}
	}
}

func stream(ctx context.Context, mapKey string, gen int, assets []string, apply func(Message)) error {
	return Stream(ctx, mapKey, assets, func(msg Message) error {
		msg.Gen = gen
		apply(msg)
		return nil
	})
}

// Assets 收集这组市场里 Polymarket 的 YES token（NO 盘口由策略内存互补维护）。
func Assets(outcomes []persistententity.MarketOutcome) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, oc := range outcomes {
		if !strings.EqualFold(oc.Platform, appcommon.PlatformPolymarket) {
			continue
		}
		id := strings.TrimSpace(oc.YesTokenID)
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
