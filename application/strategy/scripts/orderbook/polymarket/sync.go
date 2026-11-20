package polymarket

import (
	"context"
	"encoding/json"
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/fetchopt"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// Syncer Polymarket CLOB 盘口拉取并写入 market_order_book。
type Syncer struct {
	API   httprepo.PolymarketRepo
	Books persistentrepo.OrderBookRepo
}

func (s *Syncer) Sync(ctx context.Context, markets []persistententity.MarketOutcome, fixtures map[uint]persistententity.Market, fetch fetchopt.PlatformFetch) (upserted, missed int, err error) {
	if s.API == nil {
		return 0, len(markets), nil
	}
	if s.Books == nil || len(markets) == 0 {
		return 0, 0, nil
	}
	fetch = fetch.Normalized(appcommon.PlatformPolymarket)
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
	if len(ch.items) == 0 {
		return 0, 0, nil
	}
	resp, err := s.API.BatchGetOrderBooks(ctx, httpentity.BatchGetOrderBooksRequest{Items: ch.items})
	if err != nil {
		return 0, len(ch.markets), err
	}
	books := []httpentity.OrderBookResponse{}
	if resp != nil {
		books = resp.OrderBooks
	}
	rows, batchMissed := booksToRows(ch.markets, books, fixtures)
	missed = batchMissed
	if len(rows) == 0 {
		return 0, missed, nil
	}
	if err := s.Books.BatchUpsert(ctx, nil, rows); err != nil {
		return 0, missed, err
	}
	return len(rows), missed, nil
}

func booksToRows(markets []persistententity.MarketOutcome, books []httpentity.OrderBookResponse, fixtures map[uint]persistententity.Market) ([]*persistententity.OrderBook, int) {
	byAsset := make(map[string]httpentity.OrderBookResponse, len(books))
	for _, b := range books {
		id := strings.TrimSpace(b.AssetID)
		if id == "" {
			continue
		}
		byAsset[id] = b
	}
	var out []*persistententity.OrderBook
	missed := 0
	for i := range markets {
		m := markets[i]
		token := strings.TrimSpace(m.YesTokenID)
		book, ok := byAsset[token]
		if !ok {
			missed++
			continue
		}
		bidsJSON, err := json.Marshal(book.Bids)
		if err != nil {
			missed++
			continue
		}
		asksJSON, err := json.Marshal(book.Asks)
		if err != nil {
			missed++
			continue
		}
		var fx *persistententity.Market
		if f, ok := fixtures[m.MarketRowID]; ok {
			fx = &f
		}
		snap := common.SnapshotSecondOrNow(common.SnapshotSecondFromPolyTimestamp(book.Timestamp))
		out = append(out, common.RowFromOutcomeAt(&m, fx, string(bidsJSON), string(asksJSON), token, snap))
	}
	return out, missed
}
