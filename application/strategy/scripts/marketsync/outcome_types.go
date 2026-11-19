package marketsync

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/leaguecfg"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const configKeySyncOutcomeTypes = "sync_outcome_types"

func syncOutcomeTypeSetFrom(types []cryptoPredictionEnum.OutcomeMarketType) syncopt.OutcomeTypeSet {
	if len(types) == 0 {
		return syncopt.DefaultOutcomeTypeSet()
	}
	var s syncopt.OutcomeTypeSet
	for _, t := range types {
		switch t {
		case cryptoPredictionEnum.OutcomeMarketTypeMoneyline:
			s.Moneyline = true
		case cryptoPredictionEnum.OutcomeMarketTypeHandicap:
			s.Handicap = true
		case cryptoPredictionEnum.OutcomeMarketTypeTotal:
			s.Total = true
		}
	}
	if s.Empty() {
		return syncopt.DefaultOutcomeTypeSet()
	}
	return s
}

func (j *MarketSync) resolveSyncOutcomeTypes(ctx context.Context) cryptoPredictionEnum.SyncOutcomeTypeConfig {
	if j.marketCategoryRepo != nil {
		if parsed, ok := leaguecfg.SyncOutcomeTypes(ctx, j.marketCategoryRepo); ok {
			for sub, types := range parsed.BySubCategory {
				set := syncOutcomeTypeSetFrom(types)
				logger.InfoZ(ctx, "market_sync-outcome-types-catalog",
					zap.String("sport_cn", parsed.SportCN[sub]),
					zap.String("sub_category", string(sub)),
					zap.Bool("moneyline", set.Moneyline),
					zap.Bool("handicap", set.Handicap),
					zap.Bool("total", set.Total))
			}
			return parsed
		}
	}
	if j.configStrategyRepo == nil {
		return cryptoPredictionEnum.SyncOutcomeTypeConfig{}
	}
	cfg, err := j.configStrategyRepo.FindOne(ctx, nil, map[string]interface{}{
		"group_name = ?": configGroupMarketSync,
		"item_key = ?":   configKeySyncOutcomeTypes,
		"is_enabled = ?": 1,
	}, "")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			notification.SendErrorLog(ctx, "market_sync-outcome-types-read-failed", zap.Error(err))
		}
		return cryptoPredictionEnum.SyncOutcomeTypeConfig{}
	}
	parsed := cryptoPredictionEnum.ParseSyncOutcomeTypes(cfg.ItemValue, cfg.ValueType)
	if len(parsed.BySubCategory) == 0 {
		set := syncOutcomeTypeSetFrom(parsed.LegacyAll)
		logger.InfoZ(ctx, "market_sync-outcome-types-loaded",
			zap.String("sub_category", "*"),
			zap.Bool("moneyline", set.Moneyline),
			zap.Bool("handicap", set.Handicap),
			zap.Bool("total", set.Total))
		return parsed
	}
	for sub, types := range parsed.BySubCategory {
		set := syncOutcomeTypeSetFrom(types)
		logger.InfoZ(ctx, "market_sync-outcome-types-loaded",
			zap.String("sport_cn", parsed.SportCN[sub]),
			zap.String("sub_category", string(sub)),
			zap.Bool("moneyline", set.Moneyline),
			zap.Bool("handicap", set.Handicap),
			zap.Bool("total", set.Total))
	}
	return parsed
}

func outcomeTypesForSub(cfg cryptoPredictionEnum.SyncOutcomeTypeConfig, sub cryptoPredictionEnum.SubMarketCategory) syncopt.OutcomeTypeSet {
	return syncOutcomeTypeSetFrom(cfg.For(sub))
}
