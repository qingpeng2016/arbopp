package sxbet

import (
	"context"
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/fetchopt"
	sxbetws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/sxbet"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// Syncer SX Bet GET /orderbook-v3/snapshot/market → market_order_book。
type Syncer struct {
	API   httprepo.SxbetRepo
	Books persistentrepo.OrderBookRepo
}

func (s *Syncer) Sync(ctx context.Context, markets []persistententity.MarketOutcome, fixtures map[uint]persistententity.Market, fetch fetchopt.PlatformFetch) (upserted, missed int, err error) {
	if s.API == nil {
		return 0, len(markets), nil
	}
	if s.Books == nil || len(markets) == 0 {
		return 0, 0, nil
	}
	byHash := groupByMarketHash(markets)
	if len(byHash) == 0 {
		return 0, len(markets), nil
	}
	fetch = fetch.Normalized(appcommon.PlatformSxbet)
	tasks := make([]fetchopt.BatchTask, 0, len(byHash))
	for hash, outcomes := range byHash {
		hash, outcomes := hash, outcomes
		tasks = append(tasks, func(ctx context.Context) (int, int, error) {
			return s.syncHash(ctx, hash, outcomes, fixtures)
		})
	}
	return fetchopt.RunConcurrent(ctx, fetch.BatchConcurrency, tasks)
}

func (s *Syncer) syncHash(ctx context.Context, marketHash string, outcomes []persistententity.MarketOutcome, fixtures map[uint]persistententity.Market) (upserted, missed int, err error) {
	snap, err := s.API.GetOrderbookSnapshot(ctx, marketHash)
	if err != nil || snap == nil {
		return 0, len(outcomes), err
	}
	yesBids, yesAsks := sxbetws.LevelsToYesBook(snap.OutcomeOne, snap.OutcomeTwo)
	snapMs := common.RestFetchSnapshotAtMs()
	var rows []*persistententity.OrderBook
	for i := range outcomes {
		m := outcomes[i]
		var fx *persistententity.Market
		if f, ok := fixtures[m.MarketRowID]; ok {
			fx = &f
		}
		rows = append(rows, common.PredictBookRowsWithFixture(&m, fx, yesBids, yesAsks, snapMs)...)
	}
	if len(rows) == 0 {
		return 0, len(outcomes), nil
	}
	if err := s.Books.BatchCreate(ctx, nil, rows); err != nil {
		return 0, len(outcomes), err
	}
	return len(rows), 0, nil
}

func marketHash(m persistententity.MarketOutcome) string {
	h := strings.TrimSpace(m.PlatformMarketOutcomeID)
	if h == "" {
		h = strings.TrimSpace(m.YesTokenID)
	}
	return h
}

func groupByMarketHash(markets []persistententity.MarketOutcome) map[string][]persistententity.MarketOutcome {
	out := map[string][]persistententity.MarketOutcome{}
	for i := range markets {
		h := marketHash(markets[i])
		if h == "" {
			continue
		}
		out[h] = append(out[h], markets[i])
	}
	return out
}