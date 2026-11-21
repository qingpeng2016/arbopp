package marketsync

import (
	"context"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/kalshi"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/leaguecfg"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/leagueopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/predict"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/polymarket"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/sxbet"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	configGroupMarketSync     = "market_sync"
	configKeyKalshiGameSeries = "kalshi_game_series"
	configKeyPolymarketSports = "polymarket_sports"
	configKeyPredictSports    = "predict_sports"
	configKeySxbetLeagues     = "sxbet_leagues"
	configKeyConcurrency = "concurrency"
)

// resolveSyncPlatforms 使用 platform 表中 status=ACTIVE 的 slug；仅保留已实现拉数的平台
func (j *MarketSync) resolveSyncPlatforms(ctx context.Context) (map[string]struct{}, error) {
	active, err := j.activePlatformSlugs(ctx)
	if err != nil {
		return nil, err
	}
	if len(active) == 0 {
		notification.SendErrorLog(ctx, "market_sync-no-active-platforms")
		return nil, nil
	}

	supported := marketSyncSupportedPlatforms()
	out := make(map[string]struct{}, len(active))
	for slug := range active {
		if _, ok := supported[slug]; !ok {
			notification.SendErrorLog(ctx, "market_sync-platform-skipped",
				zap.String("slug", slug),
				zap.String("reason", "market_sync has no fetcher for this platform yet"))
			continue
		}
		out[slug] = struct{}{}
	}
	if len(out) == 0 {
		notification.SendErrorLog(ctx, "market_sync-no-supported-active-platforms",
			zap.Strings("active", mapKeysSorted(active)))
		return nil, nil
	}
	logger.InfoZ(ctx, "market_sync-platforms-resolved",
		zap.Strings("active", mapKeysSorted(active)),
		zap.Strings("effective", mapKeysSorted(out)))
	return out, nil
}

func (j *MarketSync) activePlatformSlugs(ctx context.Context) (map[string]struct{}, error) {
	_, list, err := j.platformRepo.List(ctx, nil, map[string]interface{}{
		"status = ?": "ACTIVE",
	}, "sort asc", "")
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(list))
	for _, p := range list {
		slug := strings.ToUpper(strings.TrimSpace(p.Slug))
		if slug != "" {
			out[slug] = struct{}{}
		}
	}
	return out, nil
}

func mapKeysSorted(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// small n, simple sort
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

// resolveKalshiGameSeriesConfig 读取 kalshi_game_series（JSON 含中文 name）。
func (j *MarketSync) resolveKalshiGameSeriesConfig(ctx context.Context) kalshi.SeriesConfig {
	if j.marketCategoryRepo != nil {
		if cfg, ok := leaguecfg.KalshiSeriesConfig(ctx, j.marketCategoryRepo); ok {
			logger.InfoZ(ctx, "market_sync-kalshi-series-config-catalog",
				zap.Int("sub_categories", len(cfg.BySubCategory)))
			return cfg
		}
	}
	cfg, err := j.configStrategyRepo.FindOne(ctx, nil, map[string]interface{}{
		"group_name = ?": configGroupMarketSync,
		"item_key = ?":   configKeyKalshiGameSeries,
		"is_enabled = ?": 1,
	}, "")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			notification.SendErrorLog(ctx, "market_sync-kalshi-series-config-read-failed", zap.Error(err))
		}
		return kalshi.SeriesConfig{}
	}
	loaded := kalshi.LoadSeriesConfig(cfg.ItemValue, cfg.ValueType)
	if len(loaded.BySubCategory) == 0 {
		return kalshi.SeriesConfig{}
	}
	logger.InfoZ(ctx, "market_sync-kalshi-series-config-loaded", zap.Int("sub_categories", len(loaded.BySubCategory)))
	return loaded
}

func (j *MarketSync) resolvePolymarketSportsConfig(ctx context.Context) polymarket.SportsConfig {
	if j.marketCategoryRepo != nil {
		if cfg, ok := leaguecfg.PolymarketSportsConfig(ctx, j.marketCategoryRepo); ok {
			logger.InfoZ(ctx, "market_sync-polymarket-sports-config-catalog",
				zap.Int("sub_categories", len(cfg.BySubCategory)),
				zap.Bool("sync_all_soccer_games", cfg.SyncAllSoccerGames),
				zap.Int("soccer_game_tags", len(cfg.SoccerGameTagIDs)))
			return cfg
		}
	}
	cfg, err := j.configStrategyRepo.FindOne(ctx, nil, map[string]interface{}{
		"group_name = ?": configGroupMarketSync,
		"item_key = ?":   configKeyPolymarketSports,
		"is_enabled = ?": 1,
	}, "")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			notification.SendErrorLog(ctx, "market_sync-polymarket-sports-config-read-failed", zap.Error(err))
		}
		return polymarket.SportsConfig{}
	}
	loaded := polymarket.LoadSportsConfig(cfg.ItemValue, cfg.ValueType)
	if len(loaded.BySubCategory) == 0 {
		return polymarket.SportsConfig{}
	}
	logger.InfoZ(ctx, "market_sync-polymarket-sports-config-loaded",
		zap.Int("sub_categories", len(loaded.BySubCategory)),
		zap.Bool("sync_all_soccer_games", loaded.SyncAllSoccerGames),
		zap.Int("soccer_game_tags", len(loaded.SoccerGameTagIDs)))
	return loaded
}

func (j *MarketSync) resolveLeagueFetchConfig(ctx context.Context) leagueopt.Config {
	if j.marketCategoryRepo != nil {
		if cfg, ok := leaguecfg.LeagueFetchConfig(ctx, j.marketCategoryRepo); ok {
			logger.InfoZ(ctx, "market_sync-league-fetch-config-catalog",
				zap.Int("polymarket", cfg.Polymarket.Concurrency()),
				zap.Int("kalshi", cfg.Kalshi.Concurrency()),
				zap.Int("predict", cfg.Predict.Concurrency()),
				zap.Int("sxbet", cfg.Sxbet.Concurrency()))
			return cfg
		}
	}
	cfg, err := j.configStrategyRepo.FindOne(ctx, nil, map[string]interface{}{
		"group_name = ?": configGroupMarketSync,
		"item_key = ?":   configKeyConcurrency,
		"is_enabled = ?": 1,
	}, "")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			notification.SendErrorLog(ctx, "market_sync-league-fetch-config-read-failed", zap.Error(err))
		}
		return leagueopt.Config{}
	}
	loaded := leagueopt.Parse(cfg.ItemValue, cfg.ValueType)
	logger.InfoZ(ctx, "market_sync-league-fetch-config-loaded",
		zap.Int("polymarket", loaded.Polymarket.Concurrency()),
		zap.Int("kalshi", loaded.Kalshi.Concurrency()),
		zap.Int("predict", loaded.Predict.Concurrency()),
		zap.Int("sxbet", loaded.Sxbet.Concurrency()))
	return loaded
}

func (j *MarketSync) resolvePredictSportsConfig(ctx context.Context) predict.SportsConfig {
	if j.marketCategoryRepo != nil {
		if cfg, ok := leaguecfg.PredictSportsConfig(ctx, j.marketCategoryRepo); ok {
			logger.InfoZ(ctx, "market_sync-predict-sports-config-catalog",
				zap.Int("sub_categories", len(cfg.BySubCategory)),
				zap.Int("sync_root_subs", len(cfg.SyncRootTagBySub)),
				zap.Int("tag_league_map", len(cfg.TagToCompetition)))
			return cfg
		}
	}
	cfg, err := j.configStrategyRepo.FindOne(ctx, nil, map[string]interface{}{
		"group_name = ?": configGroupMarketSync,
		"item_key = ?":   configKeyPredictSports,
		"is_enabled = ?": 1,
	}, "")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			notification.SendErrorLog(ctx, "market_sync-predict-sports-config-read-failed", zap.Error(err))
		}
		return predict.SportsConfig{}
	}
	loaded := predict.LoadSportsConfig(cfg.ItemValue, cfg.ValueType)
	if len(loaded.BySubCategory) == 0 {
		return predict.SportsConfig{}
	}
	logger.InfoZ(ctx, "market_sync-predict-sports-config-loaded",
		zap.Int("sub_categories", len(loaded.BySubCategory)),
		zap.Int("sync_root_subs", len(loaded.SyncRootTagBySub)),
		zap.Int("tag_league_map", len(loaded.TagToCompetition)))
	return loaded
}

func (j *MarketSync) resolveSxbetLeaguesConfig(ctx context.Context) sxbet.LeaguesConfig {
	if j.marketCategoryRepo != nil {
		if cfg, ok := leaguecfg.SxbetLeaguesConfig(ctx, j.marketCategoryRepo); ok {
			logger.InfoZ(ctx, "market_sync-sxbet-leagues-config-catalog",
				zap.Int("sub_categories", len(cfg.BySubCategory)))
			return cfg
		}
	}
	cfg, err := j.configStrategyRepo.FindOne(ctx, nil, map[string]interface{}{
		"group_name = ?": configGroupMarketSync,
		"item_key = ?":   configKeySxbetLeagues,
		"is_enabled = ?": 1,
	}, "")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			notification.SendErrorLog(ctx, "market_sync-sxbet-leagues-config-read-failed", zap.Error(err))
		}
		return sxbet.LeaguesConfig{}
	}
	loaded := sxbet.LoadLeaguesConfig(cfg.ItemValue, cfg.ValueType)
	if len(loaded.BySubCategory) == 0 {
		return sxbet.LeaguesConfig{}
	}
	logger.InfoZ(ctx, "market_sync-sxbet-leagues-config-loaded", zap.Int("sub_categories", len(loaded.BySubCategory)))
	return loaded
}

func marketSyncSupportedPlatforms() map[string]struct{} {
	return map[string]struct{}{
		string(cryptoPredictionEnum.PlatformPolymarket): {},
		string(cryptoPredictionEnum.PlatformKalshi):     {},
		string(cryptoPredictionEnum.PlatformPredict):    {},
		string(cryptoPredictionEnum.PlatformSxbet):      {},
	}
}
