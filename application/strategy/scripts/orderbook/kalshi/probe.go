package kalshi

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	"go.uber.org/zap"
)

// RunProbe 单市场探针，仅日志。
func RunProbe(ctx context.Context, api httprepo.KalshiRepo) {
	if api == nil {
		notification.SendErrorLog(ctx, "orderbook-kalshi-skipped", zap.String("reason", "kalshi repo nil"))
		return
	}
	markets, err := api.GetMarkets(ctx, httprepo.KalshiListParams{
		Limit:     5,
		Status:    "open",
		MveFilter: "exclude",
	})
	if err != nil {
		notification.SendErrorLog(ctx, "orderbook-kalshi-GetMarkets-failed", zap.Error(err))
		return
	}
	if markets == nil || len(markets.Markets) == 0 {
		logger.InfoZ(ctx, "orderbook-kalshi-no-markets")
		return
	}
	ticker := markets.Markets[0].Ticker
	book, err := api.GetOrderbook(ctx, ticker, 5)
	if err != nil {
		notification.SendErrorLog(ctx, "orderbook-kalshi-GetOrderbook-failed",
			zap.String("ticker", ticker), zap.Error(err))
		return
	}
	yesN, noN := 0, 0
	if book != nil {
		yesN, noN = len(book.Yes), len(book.No)
	}
	logger.InfoZ(ctx, "orderbook-kalshi",
		zap.String("ticker", ticker),
		zap.String("title", markets.Markets[0].Title),
		zap.Int("yes_levels", yesN),
		zap.Int("no_levels", noN),
		zap.Any("orderbook", book))
}
