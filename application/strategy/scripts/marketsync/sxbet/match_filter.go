package sxbet

import (
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

// 全场盘口 type id（排除半场/节、球员 props、 outright 等）。
var (
	sxbetMoneylineTypes = map[int]struct{}{
		1:   {}, // 1X2
		52:  {}, // 12
		226: {}, // 12 incl OT
		88:  {}, // to qualify
	}
	sxbetHandicapTypes = map[int]struct{}{
		3:    {}, // Asian Handicap
		342:  {}, // AH incl OT
		201:  {}, // AH games (tennis)
		835:  {}, // Asian O/U (some sports)
	}
	sxbetTotalTypes = map[int]struct{}{
		2:    {}, // Under/Over
		28:   {}, // U/O incl OT
		166:  {}, // U/O games
		1536: {}, // U/O maps
	}
)

func sxbetIsMainLine(m httpentity.SxbetMarket) bool {
	if m.MainLine == nil {
		return true
	}
	return *m.MainLine
}

func sxbetMarketForSync(m httpentity.SxbetMarket, types syncopt.OutcomeTypeSet) (ok bool, kind cryptoPredictionEnum.OutcomeMarketType) {
	if types.Empty() {
		return false, ""
	}
	t := m.Type
	if types.Handicap {
		if _, hit := sxbetHandicapTypes[t]; hit && sxbetIsMainLine(m) {
			return true, cryptoPredictionEnum.OutcomeMarketTypeHandicap
		}
	}
	if types.Total {
		if _, hit := sxbetTotalTypes[t]; hit && sxbetIsMainLine(m) {
			return true, cryptoPredictionEnum.OutcomeMarketTypeTotal
		}
	}
	if types.Moneyline {
		if _, hit := sxbetMoneylineTypes[t]; hit {
			return true, cryptoPredictionEnum.OutcomeMarketTypeMoneyline
		}
	}
	return false, ""
}