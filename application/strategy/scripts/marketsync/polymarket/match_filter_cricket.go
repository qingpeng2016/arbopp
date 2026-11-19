package polymarket

import (
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

type cricketMatchFilter struct{}

func (cricketMatchFilter) MarketForSync(m *httpentity.Market, types syncopt.OutcomeTypeSet) (bool, cryptoPredictionEnum.OutcomeMarketType) {
	if m == nil || types.Empty() {
		return false, ""
	}
	if types.Handicap && cricketIsHandicap(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeHandicap
	}
	if types.Total && cricketIsTotal(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeTotal
	}
	if types.Moneyline && cricketIsMoneyline(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeMoneyline
	}
	return false, ""
}

func cricketJunkSlugFragments() []string {
	return []string{"-toss-winner-", "-completed-match", "-exact-score-"}
}

func cricketJunkQuestionKeywords() []string {
	return []string{
		"champion", "championship", "top run scorer", "top wicket",
		"who wins the toss", "completed match?",
	}
}

func cricketPassesMatchdayBase(m *httpentity.Market) bool {
	if !hasPolymarketScheduledKickoff(m) {
		return false
	}
	slug := polySlugLower(m)
	q := polyQuestionLower(m)
	if polySlugContainsAny(slug, cricketJunkSlugFragments()) {
		return false
	}
	if polyQuestionContainsAny(q, cricketJunkQuestionKeywords()) {
		return false
	}
	return true
}

func cricketIsMoneyline(m *httpentity.Market) bool {
	if !cricketPassesMatchdayBase(m) {
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
	if strings.Contains(q, " win on ") {
		return true
	}
	if strings.Contains(q, " beat ") || strings.Contains(q, " to win") {
		return true
	}
	return polyQuestionHasVs(q)
}

func cricketIsHandicap(m *httpentity.Market) bool {
	if !cricketPassesMatchdayBase(m) {
		return false
	}
	slug := polySlugLower(m)
	q := polyQuestionDescLower(m)
	if polySlugIsSpread(slug) {
		return true
	}
	return polyQuestionLooksLikeHandicap(q)
}

func cricketIsTotal(m *httpentity.Market) bool {
	if !cricketPassesMatchdayBase(m) {
		return false
	}
	slug := polySlugLower(m)
	q := polyQuestionLower(m)
	if polySlugIsTotal(slug) {
		return true
	}
	return polyQuestionLooksLikeTotal(q)
}