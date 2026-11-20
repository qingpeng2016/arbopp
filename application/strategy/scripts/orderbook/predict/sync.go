package predict

import (
	"context"
	"strconv"
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/fetchopt"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// Syncer Predict REST 批量订单簿写入 market_order_book。
type Syncer struct {
	API   httprepo.PredictRepo
	Books persistentrepo.OrderBookRepo
}

func (s *Syncer) Sync(ctx context.Context, markets []persistententity.MarketOutcome, fixtures map[uint]persistententity.Market, fetch fetchopt.PlatformFetch) (upserted, missed int, err error) {
	if s.API == nil {
		return 0, len(markets), nil
	}
	if s.Books == nil || len(markets) == 0 {
		return 0, 0, nil
	}
	fetch = fetch.Normalized(appcommon.PlatformPredict)
	chunks := chunkOutcomes(markets, fetch.BatchSize)
	if len(chunks) == 0 {
		return 0, len(markets), nil
	}
	tasks := make([]fetchopt.BatchTask, len(chunks))
	for i := range chunks {
		ch := chunks[i]
		tasks[i] = func(ctx context.Context) (int, int, error) {
			return s.syncChunk(ctx, ch, fixtures)
		}
	}
	return fetchopt.RunConcurrent(ctx, fetch.BatchConcurrency, tasks)
}

func (s *Syncer) syncChunk(ctx context.Context, ch fetchChunk, fixtures map[uint]persistententity.Market) (upserted, missed int, err error) {
	books, err := s.API.GetOrderbooks(ctx, ch.ids)
	if err != nil {
		return 0, len(ch.markets), err
	}
	snap := common.RestFetchSnapshotSecond()
	rows, batchMissed := booksToRows(ch.markets, books, fixtures, snap)
	missed = batchMissed
	if len(rows) == 0 {
		return 0, missed, nil
	}
	if err := s.Books.BatchUpsert(ctx, nil, rows); err != nil {
		return 0, missed, err
	}
	return len(rows), missed, nil
}

func booksToRows(markets []persistententity.MarketOutcome, books []httpentity.PredictOrderbook, fixtures map[uint]persistententity.Market, snapshotSecond int64) ([]*persistententity.OrderBook, int) {
	byID := make(map[string]httpentity.PredictOrderbook, len(books))
	for i := range books {
		id := strconv.FormatInt(books[i].MarketID, 10)
		if id == "" || id == "0" {
			continue
		}
		byID[id] = books[i]
	}
	var out []*persistententity.OrderBook
	missed := 0
	for i := range markets {
		m := markets[i]
		id := strings.TrimSpace(m.PlatformMarketOutcomeID)
		book, ok := byID[id]
		if !ok {
			missed++
			continue
		}
		yesBids := common.LevelsFromPairs(book.Bids)
		yesAsks := common.LevelsFromPairs(book.Asks)
		var fx *persistententity.Market
		if f, ok := fixtures[m.MarketRowID]; ok {
			fx = &f
		}
		out = append(out, common.PredictBookRowsWithFixture(&m, fx, yesBids, yesAsks, snapshotSecond)...)
	}
	return out, missed
}
