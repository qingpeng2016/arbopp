package backtest

import (
	"context"

	strategyentry "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

func loadStrategyParams(ctx context.Context, thresholdRepo persistentrepo.ConfigStrategyThresholdRepo, feeRepo persistentrepo.ConfigStrategyFeeRepo) strategyentry.Params {
	return strategyentry.LoadParams(ctx, thresholdRepo, feeRepo)
}