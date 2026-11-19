package kalshi

import (
	"context"
	"net/http"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"go.uber.org/zap"
)

// Books 为握手路径签名。
type Books interface {
	SignHeaders(ctx context.Context, account httpentity.KalshiAccount, method, path string) (keyID, timestamp, signature string, err error)
}

// Feed 是一条写进内存盘口的行情。Kind 为 pause、snap、delta。
type Feed struct {
	Gen    int
	Kind   string
	ID     string
	Side   string
	Price  float64
	Delta  float64
	Bids   []book.Level
	NoBids []book.Level
	Seq    int64
	TS     time.Time
}

// Session 由这组市场提供：换代，以及一条推送调用一次 Apply。
type Session struct {
	Begin func() (context.Context, int)
	Apply func(Feed)
}

// Run 没有签名账户时等待。盘口等订阅后的第一条全量，断线后重新订阅。
func Run(ctx context.Context, client Books, tickers []string, mapKey string, account func() httpentity.KalshiAccount, noteMissing func() bool, s Session) {
	for {
		if ctx.Err() != nil {
			return
		}
		acc := account()
		if acc.KeyID == "" || acc.PrivateKey == "" {
			if noteMissing() {
				notification.SendErrorLog(ctx, "strategy_entry-kalshi-no-key", zap.String("map_key", mapKey))
			}
			if !ws.Sleep(ctx, 15*time.Second) {
				return
			}
			continue
		}
		connCtx, gen := s.Begin()
		ws.NotifyReconnect(ctx, string(appcommon.PlatformKalshi), mapKey, gen)
		s.Apply(Feed{Gen: gen, Kind: "pause"})
		if ctx.Err() != nil {
			return
		}
		err := stream(connCtx, client, mapKey, gen, tickers, acc, s.Apply)
		s.Apply(Feed{Gen: gen, Kind: "pause"})
		if ctx.Err() != nil {
			return
		}
		if connCtx.Err() != nil {
			continue
		}
		notification.SendErrorLog(ctx, "strategy_entry-kalshi-disconnected", zap.String("map_key", mapKey), zap.Int("gen", gen), zap.Error(err))
		if !ws.Sleep(ctx, time.Second) {
			return
		}
	}
}

func stream(ctx context.Context, client Books, mapKey string, gen int, tickers []string, account httpentity.KalshiAccount, apply func(Feed)) error {
	key, ts, sig, err := client.SignHeaders(ctx, account, http.MethodGet, ws.KalshiPath)
	if err != nil {
		return err
	}
	header := http.Header{}
	header.Set("KALSHI-ACCESS-KEY", key)
	header.Set("KALSHI-ACCESS-TIMESTAMP", ts)
	header.Set("KALSHI-ACCESS-SIGNATURE", sig)
	return Stream(ctx, mapKey, header, tickers, func(u Update) error {
		ev := Feed{Gen: gen, ID: u.Ticker, Seq: u.Seq, TS: u.TS}
		if u.Snapshot {
			ev.Kind = "snap"
			ev.Bids = u.Yes
			ev.NoBids = u.No
		} else {
			ev.Kind = "delta"
			ev.Side = u.Side
			ev.Price = u.Price
			ev.Delta = u.Delta
		}
		apply(ev)
		return nil
	})
}

// Tickers 收集这组市场里 Kalshi 的 ticker。
func Tickers(outcomes []persistententity.MarketOutcome) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, oc := range outcomes {
		if !strings.EqualFold(oc.Platform, appcommon.PlatformKalshi) {
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
