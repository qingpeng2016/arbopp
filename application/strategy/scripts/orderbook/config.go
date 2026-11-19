package orderbook

import (
	"context"
	"encoding/json"
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/fetchopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	configGroupOrderbookSync = "orderbook_sync"
	configKeyOrderbookFetch  = "orderbook_fetch"
)

func (j *OrderBook) resolveFetchConfig(ctx context.Context) fetchopt.FetchConfig {
	def := fetchopt.DefaultFetchConfig()
	if j.configStrategyRepo == nil {
		return def
	}
	cfg, err := j.configStrategyRepo.FindOne(ctx, nil, map[string]interface{}{
		"group_name = ?": configGroupOrderbookSync,
		"item_key = ?":   configKeyOrderbookFetch,
		"is_enabled = ?": 1,
	}, "")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			notification.SendErrorLog(ctx, "orderbook_sync-fetch-config-read-failed", zap.Error(err))
		}
		return def
	}
	raw := strings.TrimSpace(cfg.ItemValue)
	if raw == "" {
		return def
	}
	var parsed fetchopt.FetchConfig
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		notification.SendErrorLog(ctx, "orderbook_sync-fetch-config-parse-failed", zap.Error(err))
		return def
	}
	poly := parsed.Polymarket.Normalized(appcommon.PlatformPolymarket)
	kal := parsed.Kalshi.Normalized(appcommon.PlatformKalshi)
	logger.InfoZ(ctx, "orderbook_sync-fetch-config-loaded",
		zap.Int("poly_batch_size", poly.BatchSize),
		zap.Int("poly_batch_concurrency", poly.BatchConcurrency),
		zap.Int("kalshi_batch_size", kal.BatchSize),
		zap.Int("kalshi_batch_concurrency", kal.BatchConcurrency))
	return fetchopt.FetchConfig{Polymarket: poly, Kalshi: kal}
}
