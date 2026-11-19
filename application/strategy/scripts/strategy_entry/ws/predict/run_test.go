package predict

import (
	"testing"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestMarketIDsCollectsEveryPredictOutcome(t *testing.T) {
	ids := MarketIDs([]persistententity.MarketOutcome{
		{Platform: appcommon.PlatformPredict, PlatformMarketOutcomeID: "4211675"},
		{Platform: appcommon.PlatformPredict, PlatformMarketOutcomeID: "4211677"},
		{Platform: appcommon.PlatformPredict, PlatformMarketOutcomeID: "4212047"},
		{Platform: appcommon.PlatformPolymarket, PlatformMarketOutcomeID: "other"},
		{Platform: appcommon.PlatformPredict, PlatformMarketOutcomeID: "4211675"},
	})
	if len(ids) != 3 || ids[0] != "4211675" || ids[2] != "4212047" {
		t.Fatalf("ids=%v", ids)
	}
}
