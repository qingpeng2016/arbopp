package kalshi

import (
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func kalshiMarketTitleBlob(m httpentity.KalshiMarket) string {
	title := strings.TrimSpace(m.Title)
	if sub := strings.TrimSpace(m.Subtitle); sub != "" {
		title = title + " " + sub
	}
	return strings.ToLower(title)
}

func kalshiIsMoneylineMarket(m httpentity.KalshiMarket) bool {
	t := kalshiMarketTitleBlob(m)
	if t == "" {
		return false
	}
	if strings.Contains(t, "spread") || strings.Contains(t, "wins by") || strings.Contains(t, "margin") {
		return false
	}
	if strings.Contains(t, "tie is the result") || strings.Contains(t, "draw") {
		return true
	}
	return strings.Contains(t, " wins") || strings.Contains(t, " win ")
}

func kalshiIsHandicapMarket(m httpentity.KalshiMarket) bool {
	t := kalshiMarketTitleBlob(m)
	if t == "" {
		return false
	}
	if strings.Contains(t, "spread") {
		return true
	}
	return strings.Contains(t, "wins by") || strings.Contains(t, "win by") || strings.Contains(t, "margin")
}

func kalshiIsTotalMarket(m httpentity.KalshiMarket) bool {
	t := kalshiMarketTitleBlob(m)
	if t == "" {
		return false
	}
	if strings.Contains(t, "1st half") || strings.Contains(t, "first half") || strings.Contains(t, "team total") {
		return false
	}
	if strings.Contains(t, "spread") || strings.Contains(t, "wins by") || strings.Contains(t, "win by") || strings.Contains(t, "margin") {
		return false
	}
	return strings.Contains(t, "over ") && strings.Contains(t, "goal")
}

func kalshiMarketForSync(m httpentity.KalshiMarket, types syncopt.OutcomeTypeSet) (ok bool, kind cryptoPredictionEnum.OutcomeMarketType) {
	if types.Empty() {
		return false, ""
	}
	if types.Handicap && kalshiIsHandicapMarket(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeHandicap
	}
	if types.Total && kalshiIsTotalMarket(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeTotal
	}
	if types.Moneyline && kalshiIsMoneylineMarket(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeMoneyline
	}
	return false, ""
}
