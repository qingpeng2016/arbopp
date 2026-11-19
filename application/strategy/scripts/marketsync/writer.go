package marketsync

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/batch"
)

// PersistFixtureOutcomes 实现 batch.Writer：先 market 场次，再 market_outcome 盘口。
func (j *MarketSync) PersistFixtureOutcomes(ctx context.Context, b *batch.FixtureOutcome) error {
	return j.batchUpsertFixtureOutcomes(ctx, b)
}

// CloseStaleAfterSync 实现 batch.Writer。
func (j *MarketSync) CloseStaleAfterSync(
	ctx context.Context,
	platform, category, subCategory string,
	seenFixtures, seenOutcomes map[string]struct{},
) (fixturesClosed, outcomesClosed int, err error) {
	return j.closeStaleAfterSync(ctx, platform, category, subCategory, seenFixtures, seenOutcomes)
}