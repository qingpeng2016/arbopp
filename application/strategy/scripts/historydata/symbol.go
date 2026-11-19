package historydata

import (
	"strings"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// PolymarketVendorSlug PMData / 伪造任务用 market slug。
func PolymarketVendorSlug(oc persistententity.MarketOutcome) string {
	if oc.Slug != nil {
		s := strings.TrimSpace(*oc.Slug)
		if s != "" {
			return strings.ToLower(s)
		}
	}
	return strings.ToLower(strings.TrimSpace(oc.PlatformMarketOutcomeID))
}

func predictVendorMarketID(oc persistententity.MarketOutcome) string {
	return strings.TrimSpace(oc.PlatformMarketOutcomeID)
}