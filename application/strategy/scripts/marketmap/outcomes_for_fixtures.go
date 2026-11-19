package marketmap

import (
	"context"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// OutcomesForFixtureMarkets 只加载指定场次下的盘口，避免全表扫描。
func OutcomesForFixtureMarkets(ctx context.Context, outcomeRepo persistentrepo.MarketOutcomeRepo, fixtures map[uint]persistententity.Market) ([]persistententity.MarketOutcome, error) {
	if outcomeRepo == nil || len(fixtures) == 0 {
		return nil, nil
	}
	ids := make([]uint, 0, len(fixtures))
	for id := range fixtures {
		ids = append(ids, id)
	}
	_, list, err := outcomeRepo.List(ctx, nil, map[string]interface{}{
		"market_row_id IN ?": ids,
	}, "platform asc, id asc", "")
	if err != nil {
		return nil, err
	}
	return FilterOutcomesUnderVerifiedFixtures(list, fixtures), nil
}