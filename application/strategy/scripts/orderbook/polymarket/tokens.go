package polymarket

import (
	"strings"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// tokenIDsForOutcome CLOB 拉盘：Yes + No（两者皆非空且不同）。
func tokenIDsForOutcome(m *persistententity.MarketOutcome) []string {
	if m == nil {
		return nil
	}
	yes := strings.TrimSpace(m.YesTokenID)
	no := strings.TrimSpace(m.NoTokenID)
	var out []string
	if yes != "" {
		out = append(out, yes)
	}
	if no != "" && no != yes {
		out = append(out, no)
	}
	return out
}