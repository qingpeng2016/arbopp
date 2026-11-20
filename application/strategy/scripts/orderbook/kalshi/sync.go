package kalshi

import (
	"context"
	"encoding/json"
	"fmt"
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

// Syncer Kalshi 盘口拉取并写入 market_order_book。
type Syncer struct {
	API   httprepo.KalshiRepo
	Books persistentrepo.OrderBookRepo
}

func (s *Syncer) Sync(ctx context.Context, markets []persistententity.MarketOutcome, fixtures map[uint]persistententity.Market, fetch fetchopt.PlatformFetch) (upserted, missed int, err error) {
	if s.API == nil {
		return 0, len(markets), nil
	}
	if s.Books == nil || len(markets) == 0 {
		return 0, 0, nil
	}
	fetch = fetch.Normalized(appcommon.PlatformKalshi)
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
	books, err := s.API.GetOrderbooks(ctx, ch.tickers)
	if err != nil {
		return 0, len(ch.markets), err
	}
	snapMs := common.RestFetchSnapshotAtMs()
	rows, batchMissed := booksToRows(ch.markets, books, fixtures, snapMs)
	missed = batchMissed
	if len(rows) == 0 {
		return 0, missed, nil
	}
	if err := s.Books.BatchCreate(ctx, nil, rows); err != nil {
		return 0, missed, err
	}
	return len(rows), missed, nil
}

func booksToRows(markets []persistententity.MarketOutcome, books map[string]*httpentity.KalshiOrderbook, fixtures map[uint]persistententity.Market, snapshotAtMs int64) ([]*persistententity.OrderBook, int) {
	var out []*persistententity.OrderBook
	missed := 0
	for i := range markets {
		m := markets[i]
		ticker := strings.TrimSpace(m.PlatformMarketOutcomeID)
		book := books[ticker]
		if book == nil {
			missed++
			continue
		}
		bidsJSON, asksJSON, err := encodeOrderbook(book)
		if err != nil {
			missed++
			continue
		}
		var fx *persistententity.Market
		if f, ok := fixtures[m.MarketRowID]; ok {
			fx = &f
		}
		out = append(out, common.RowFromOutcomeAt(&m, fx, bidsJSON, asksJSON, ticker, snapshotAtMs))
	}
	return out, missed
}

// encodeOrderbook：yes_dollars 为 YES 买盘；卖盘由 NO 买盘换算 ask ≈ 1 - no_price。
func encodeOrderbook(book *httpentity.KalshiOrderbook) (bidsJSON, asksJSON string, err error) {
	if book == nil {
		return "[]", "[]", nil
	}
	bids, err := json.Marshal(book.Yes)
	if err != nil {
		return "", "", err
	}
	asks := make([][]string, 0, len(book.No))
	for _, lvl := range book.No {
		if len(lvl) < 2 {
			continue
		}
		p, perr := strconv.ParseFloat(strings.TrimSpace(lvl[0]), 64)
		if perr != nil {
			continue
		}
		asks = append(asks, []string{fmt.Sprintf("%.4f", 1-p), lvl[1]})
	}
	asksBytes, err := json.Marshal(asks)
	if err != nil {
		return "", "", err
	}
	return string(bids), string(asksBytes), nil
}
