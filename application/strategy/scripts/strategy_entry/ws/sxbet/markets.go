package sxbet

import (
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// MarketHashes 本组 SX 盘口 marketHash（去重）。
func MarketHashes(outcomes []persistententity.MarketOutcome) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, oc := range outcomes {
		if !strings.EqualFold(oc.Platform, appcommon.PlatformSxbet) {
			continue
		}
		h := strings.TrimSpace(oc.PlatformMarketOutcomeID)
		if h == "" {
			h = strings.TrimSpace(oc.YesTokenID)
		}
		if h == "" {
			continue
		}
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		out = append(out, h)
	}
	return out
}