package polymarket

import (
	"testing"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestAssetsYesAndNoTokens(t *testing.T) {
	out := Assets([]persistententity.MarketOutcome{
		{Platform: appcommon.PlatformPolymarket, YesTokenID: "y1", NoTokenID: "n1"},
		{Platform: appcommon.PlatformPolymarket, YesTokenID: "y2", NoTokenID: "y2"},
	})
	if len(out) != 3 {
		t.Fatalf("assets: %v", out)
	}
	seen := map[string]bool{}
	for _, a := range out {
		seen[a] = true
	}
	for _, want := range []string{"y1", "n1", "y2"} {
		if !seen[want] {
			t.Fatalf("missing %s in %v", want, out)
		}
	}
}