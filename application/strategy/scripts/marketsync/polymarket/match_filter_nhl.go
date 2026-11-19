package polymarket

import (
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

type nhlMatchFilter struct{}

func (nhlMatchFilter) MarketForSync(m *httpentity.Market, types syncopt.OutcomeTypeSet) (bool, cryptoPredictionEnum.OutcomeMarketType) {
	if m == nil || types.Empty() {
		return false, ""
	}
	if types.Handicap && nhlIsHandicap(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeHandicap
	}
	if types.Total && nhlIsTotal(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeTotal
	}
	if types.Moneyline && nhlIsMoneyline(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeMoneyline
	}
	return false, ""
}

func nhlJunkQuestionKeywords() []string {
	return []string{
		"stanley cup winner", "champion", "championship", "make the playoffs",
	}
}

func nhlPassesMatchdayBase(m *httpentity.Market) bool {
	if !hasPolymarketScheduledKickoff(m) {
		return false
	}
	q := polyQuestionLower(m)
	if polyQuestionContainsAny(q, nhlJunkQuestionKeywords()) {
		return false
	}
	return true
}

func nhlIsMoneyline(m *httpentity.Market) bool {
	if !nhlPassesMatchdayBase(m) {
		return false
	}
	slug := polySlugLower(m)
	if polySlugIsSpread(slug) || polySlugIsTotal(slug) {
		return false
	}
	q := polyQuestionDescLower(m)
	if polyQuestionLooksLikeHandicap(q) || polyQuestionLooksLikeTotal(q) {
		return false
	}
	return polyQuestionHasVs(q)
}

func nhlIsHandicap(m *httpentity.Market) bool {
	if !nhlPassesMatchdayBase(m) {
		return false
	}
	slug := polySlugLower(m)
	q := polyQuestionDescLower(m)
	if polySlugIsSpread(slug) {
		return true
	}
	return polyQuestionLooksLikeHandicap(q)
}

func nhlIsTotal(m *httpentity.Market) bool {
	if !nhlPassesMatchdayBase(m) {
		return false
	}
	slug := polySlugLower(m)
	q := polyQuestionLower(m)
	if strings.Contains(slug, "team-total") || strings.Contains(q, "team total") {
		return false
	}
	if strings.Contains(q, "1st half") || strings.Contains(q, "first half") || strings.Contains(slug, "-1h-") {
		return false
	}
	if polySlugIsTotal(slug) {
		return true
	}
	return polyQuestionLooksLikeTotal(q)
}