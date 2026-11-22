package main

import (
	"strings"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func outcomeToEntity(o outcomeRec) *persistententity.MarketOutcome {
	mo := &persistententity.MarketOutcome{
		Platform:                o.Platform,
		PlatformMarketOutcomeID: o.PlatformMarketOutcomeID,
		OutcomeType:             o.OutcomeType,
		YesTokenID:              o.YesTokenID,
		NoTokenID:               o.NoTokenID,
	}
	if s := strings.TrimSpace(o.OutcomeName); s != "" {
		mo.OutcomeName = &s
	}
	if s := strings.TrimSpace(o.Slug); s != "" {
		mo.Slug = &s
	}
	if s := strings.TrimSpace(o.Competition); s != "" {
		mo.Competition = &s
	}
	if s := strings.TrimSpace(o.MapKey); s != "" {
		mo.MapKey = &s
	}
	return mo
}