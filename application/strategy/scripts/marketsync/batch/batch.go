package batch

import (
	"context"
	"strings"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// FixtureOutcome 一次同步写入：场次 + 盘口。
type FixtureOutcome struct {
	Fixtures []*persistententity.Market
	Outcomes []*persistententity.MarketOutcome
}

func (b *FixtureOutcome) AddFixture(fx *persistententity.Market) {
	if b == nil || fx == nil || strings.TrimSpace(fx.PlatformMarketID) == "" {
		return
	}
	b.Fixtures = append(b.Fixtures, fx)
}

func (b *FixtureOutcome) AddOutcome(oc *persistententity.MarketOutcome) {
	if b == nil || oc == nil || strings.TrimSpace(oc.PlatformMarketOutcomeID) == "" {
		return
	}
	b.Outcomes = append(b.Outcomes, oc)
}

// Writer 平台同步落库（由 MarketSync 实现，避免 import 环）。
type Writer interface {
	PersistFixtureOutcomes(ctx context.Context, b *FixtureOutcome) error
	CloseStaleAfterSync(ctx context.Context, platform, category, subCategory string, seenFixtures, seenOutcomes map[string]struct{}) (fixturesClosed, outcomesClosed int, err error)
}