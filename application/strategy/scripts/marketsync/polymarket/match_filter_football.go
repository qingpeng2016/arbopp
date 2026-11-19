package polymarket

import (
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

type footballMatchFilter struct{}

func (footballMatchFilter) MarketForSync(m *httpentity.Market, types syncopt.OutcomeTypeSet) (bool, cryptoPredictionEnum.OutcomeMarketType) {
	if m == nil || types.Empty() {
		return false, ""
	}
	if types.Handicap && footballIsHandicap(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeHandicap
	}
	if types.Total && footballIsTotal(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeTotal
	}
	if types.Moneyline && footballIsMoneyline(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeMoneyline
	}
	return false, ""
}

func footballCrossSportSlugPrefixes() []string {
	return []string{"crint", "criclcl", "cric", "odi", "t20", "nba", "nfl", "mlb", "nhl"}
}

func footballJunkSlugFragments() []string {
	return []string{"-exact-score-", "-btts", "-toss-winner-", "-completed-match"}
}

func footballJunkQuestionKeywords() []string {
	return []string{
		"champion", "championship", "qualify", "relegate", "relegation",
		"win the 20", "finish first", "top goalscorer", "golden boot",
		"make the playoffs", "win the league",
		"exact score:", "both teams to score",
		"who wins the toss", "completed match?",
	}
}

func footballPassesMatchdayBase(m *httpentity.Market) bool {
	if !hasPolymarketScheduledKickoff(m) {
		return false
	}
	slug := polySlugLower(m)
	q := polyQuestionLower(m)
	for _, prefix := range footballCrossSportSlugPrefixes() {
		if strings.HasPrefix(slug, prefix) || strings.HasPrefix(slug, prefix+"-") {
			return false
		}
	}
	if polySlugContainsAny(slug, footballJunkSlugFragments()) {
		return false
	}
	if polyQuestionContainsAny(q, footballJunkQuestionKeywords()) {
		return false
	}
	return true
}

func footballIsMoneyline(m *httpentity.Market) bool {
	if !footballPassesMatchdayBase(m) {
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
	if strings.Contains(q, "end in a draw") {
		return true
	}
	return false
}

func footballIsHandicap(m *httpentity.Market) bool {
	if !footballPassesMatchdayBase(m) {
		return false
	}
	slug := polySlugLower(m)
	q := polyQuestionDescLower(m)
	if polySlugIsSpread(slug) {
		return true
	}
	return polyQuestionLooksLikeHandicap(q)
}

func footballIsTotal(m *httpentity.Market) bool {
	if !footballPassesMatchdayBase(m) {
		return false
	}
	slug := polySlugLower(m)
	q := polyQuestionLower(m)
	if strings.Contains(q, "1st half") || strings.Contains(q, "first half") || strings.Contains(slug, "-1h-") || strings.Contains(slug, "team-total") || strings.Contains(q, "team total") {
		return false
	}
	if polySlugIsTotal(slug) {
		return true
	}
	return polyQuestionLooksLikeTotal(q)
}