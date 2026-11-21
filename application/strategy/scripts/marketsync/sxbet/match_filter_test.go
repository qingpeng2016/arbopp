package sxbet

import (
	"testing"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func TestSxbetMarketForSync(t *testing.T) {
	mainTrue := true
	m := httpentity.SxbetMarket{Type: 226, Status: "ACTIVE"}
	ok, kind := sxbetMarketForSync(m, syncopt.OutcomeTypeSet{Moneyline: true})
	if !ok || kind != cryptoPredictionEnum.OutcomeMarketTypeMoneyline {
		t.Fatalf("moneyline: ok=%v kind=%s", ok, kind)
	}
	spread := httpentity.SxbetMarket{Type: 3, MainLine: &mainTrue}
	ok, kind = sxbetMarketForSync(spread, syncopt.OutcomeTypeSet{Handicap: true})
	if !ok || kind != cryptoPredictionEnum.OutcomeMarketTypeHandicap {
		t.Fatalf("handicap: ok=%v kind=%s", ok, kind)
	}
	mainFalse := false
	alt := httpentity.SxbetMarket{Type: 2, MainLine: &mainFalse}
	ok, _ = sxbetMarketForSync(alt, syncopt.OutcomeTypeSet{Total: true})
	if ok {
		t.Fatal("expected non-main total skipped")
	}
}