package polymarket

import (
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

// defaultMatchFilter 未单独建文件的子类：保守单场规则，后续新增子类请新建 match_filter_<sub>.go。
type defaultMatchFilter struct{}

func (defaultMatchFilter) MarketForSync(m *httpentity.Market, types syncopt.OutcomeTypeSet) (bool, cryptoPredictionEnum.OutcomeMarketType) {
	if m == nil || types.Empty() {
		return false, ""
	}
	if types.Handicap && defaultIsHandicap(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeHandicap
	}
	if types.Total && defaultIsTotal(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeTotal
	}
	if types.Moneyline && defaultIsMoneyline(m) {
		return true, cryptoPredictionEnum.OutcomeMarketTypeMoneyline
	}
	return false, ""
}

func defaultPassesMatchdayBase(m *httpentity.Market) bool {
	if !hasPolymarketScheduledKickoff(m) {
		return false
	}
	q := polyQuestionLower(m)
	if polyQuestionContainsAny(q, []string{"champion", "championship"}) {
		return false
	}
	return true
}

func defaultIsMoneyline(m *httpentity.Market) bool {
	if !defaultPassesMatchdayBase(m) {
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

func defaultIsHandicap(m *httpentity.Market) bool {
	if !defaultPassesMatchdayBase(m) {
		return false
	}
	slug := polySlugLower(m)
	q := polyQuestionDescLower(m)
	if polySlugIsSpread(slug) {
		return true
	}
	return polyQuestionLooksLikeHandicap(q)
}

func defaultIsTotal(m *httpentity.Market) bool {
	if !defaultPassesMatchdayBase(m) {
		return false
	}
	slug := polySlugLower(m)
	q := polyQuestionLower(m)
	if polySlugIsTotal(slug) {
		return true
	}
	return polyQuestionLooksLikeTotal(q)
}