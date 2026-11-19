package orderbook

import (
	"strings"

	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func splitOutcomesByPlatform(rows []persistententity.MarketOutcome) (kalshi, poly, predict []persistententity.MarketOutcome) {
	for i := range rows {
		m := rows[i]
		platform := strings.ToUpper(strings.TrimSpace(m.Platform))
		if strings.TrimSpace(m.PlatformMarketOutcomeID) == "" {
			continue
		}
		switch platform {
		case string(cryptoPredictionEnum.PlatformKalshi):
			if strings.TrimSpace(m.YesTokenID) == "" {
				m.YesTokenID = m.PlatformMarketOutcomeID
			}
			kalshi = append(kalshi, m)
		case string(cryptoPredictionEnum.PlatformPolymarket):
			token := strings.TrimSpace(m.YesTokenID)
			if token == "" {
				continue
			}
			m.YesTokenID = token
			poly = append(poly, m)
		case string(cryptoPredictionEnum.PlatformPredict):
			predict = append(predict, m)
		}
	}
	return kalshi, poly, predict
}
