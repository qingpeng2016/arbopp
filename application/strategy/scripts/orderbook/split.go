package orderbook

import (
	"strings"

	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func splitOutcomesByPlatform(rows []persistententity.MarketOutcome) (kalshi, poly, predict, sxbet []persistententity.MarketOutcome) {
	for i := range rows {
		m := rows[i]
		platform := strings.ToUpper(strings.TrimSpace(m.Platform))
		switch platform {
		case string(cryptoPredictionEnum.PlatformKalshi):
			if strings.TrimSpace(m.PlatformMarketOutcomeID) == "" {
				continue
			}
			if strings.TrimSpace(m.YesTokenID) == "" {
				m.YesTokenID = m.PlatformMarketOutcomeID
			}
			kalshi = append(kalshi, m)
		case string(cryptoPredictionEnum.PlatformPolymarket):
			if strings.TrimSpace(m.PlatformMarketOutcomeID) == "" {
				continue
			}
			token := strings.TrimSpace(m.YesTokenID)
			if token == "" {
				continue
			}
			m.YesTokenID = token
			poly = append(poly, m)
		case string(cryptoPredictionEnum.PlatformPredict):
			if strings.TrimSpace(m.PlatformMarketOutcomeID) == "" {
				continue
			}
			predict = append(predict, m)
		case string(cryptoPredictionEnum.PlatformSxbet):
			if marketHashSX(m) == "" {
				continue
			}
			sxbet = append(sxbet, m)
		}
	}
	return kalshi, poly, predict, sxbet
}

func marketHashSX(m persistententity.MarketOutcome) string {
	h := strings.TrimSpace(m.PlatformMarketOutcomeID)
	if h == "" {
		h = strings.TrimSpace(m.YesTokenID)
	}
	return h
}
