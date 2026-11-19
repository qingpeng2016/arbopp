package polymarket

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	"go.uber.org/zap"
)

// RunProbe 单市场探针，仅日志。
func RunProbe(ctx context.Context, api httprepo.PolymarketRepo) {
	if api == nil {
		notification.SendErrorLog(ctx, "orderbook-polymarket-skipped", zap.String("reason", "polymarket repo nil"))
		return
	}
	closed := false
	markets, err := api.ListMarkets(ctx, httpentity.ListMarketsRequest{
		Limit:  5,
		Closed: &closed,
		Order:  "volume24hr",
	})
	if err != nil {
		notification.SendErrorLog(ctx, "orderbook-polymarket-ListMarkets-failed", zap.Error(err))
		return
	}
	tokenID := ""
	question := ""
	for _, m := range markets {
		tokenID = firstClobTokenID(m.ClobTokenIds)
		if tokenID != "" {
			question = m.Question
			break
		}
	}
	if tokenID == "" {
		logger.InfoZ(ctx, "orderbook-polymarket-no-token")
		return
	}
	book, err := api.GetOrderBook(ctx, httpentity.GetOrderBookRequest{TokenID: tokenID})
	if err != nil {
		notification.SendErrorLog(ctx, "orderbook-polymarket-GetOrderBook-failed",
			zap.String("token_id", tokenID), zap.Error(err))
		return
	}
	bids, asks := 0, 0
	if book != nil {
		bids, asks = len(book.Bids), len(book.Asks)
	}
	logger.InfoZ(ctx, "orderbook-polymarket",
		zap.String("token_id", tokenID),
		zap.String("question", question),
		zap.Int("bids", bids),
		zap.Int("asks", asks),
		zap.Any("orderbook", book))
}

func firstClobTokenID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err == nil && len(ids) > 0 {
		return strings.TrimSpace(ids[0])
	}
	parts := strings.Split(raw, ",")
	if len(parts) > 0 {
		return strings.Trim(strings.TrimSpace(parts[0]), `"'[]`)
	}
	return ""
}
