package sxbet

import (
	"context"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"go.uber.org/zap"
)

// Session 由 watch 提供换代与 Apply。
type Session struct {
	Begin    func() (context.Context, int)
	Apply    func(Message)
	Snapshot func(gen int)
}

// Run 维持 SX Bet orderbook_v3 WS，断线重连。
func Run(ctx context.Context, api httprepo.SxbetRepo, outcomes []persistententity.MarketOutcome, mapKey string, s Session) {
	hashes := MarketHashes(outcomes)
	if len(hashes) == 0 || api == nil || strings.TrimSpace(api.APIKey()) == "" {
		return
	}
	for {
		if ctx.Err() != nil {
			return
		}
		connCtx, gen := s.Begin()
		ws.NotifyReconnect(ctx, string(appcommon.PlatformSxbet), mapKey, gen)
		s.Apply(Message{Gen: gen, Kind: "pause"})
		if ctx.Err() != nil {
			return
		}
		if s.Snapshot != nil {
			s.Snapshot(gen)
		}
		err := Stream(connCtx, api, hashes, gen, s.Apply)
		s.Apply(Message{Gen: gen, Kind: "pause"})
		if ctx.Err() != nil {
			return
		}
		if connCtx.Err() != nil {
			continue
		}
		notification.SendErrorLog(ctx, "strategy_entry-sxbet-disconnected",
			zap.String("map_key", mapKey), zap.Int("gen", gen), zap.Error(err))
		if !ws.Sleep(ctx, time.Second) {
			return
		}
	}
}