package yesno

import (
	"strings"
	"testing"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestTopHitAndBestFilledPick(t *testing.T) {
	yesTok, noTok := "poly-yes-1", "poly-no-1"
	kTicker := "kx-test-ars"
	predictID := "pred-mkt-1"
	store := book.NewStore()
	store.ApplyPolyBook(yesTok,
		[]book.Level{{Price: 0.36, Size: 500}}, []book.Level{{Price: 0.38, Size: 800}},
		0.01, 1, time.Now().UTC())
	store.ApplyPolyBook(noTok,
		[]book.Level{{Price: 0.60, Size: 500}}, []book.Level{{Price: 0.62, Size: 800}},
		0.01, 1, time.Now().UTC())
	store.ApplyKalshiSnapshot(kTicker,
		[]book.Level{{Price: 0.56, Size: 800}}, []book.Level{{Price: 0.30, Size: 50}},
		1, time.Now().UTC())
	store.ApplyPredictBook(predictID,
		[]book.Level{{Price: 0.40, Size: 500}}, []book.Level{{Price: 0.48, Size: 500}},
		time.Now().UTC())

	wg := &watchgroup.Watch{
		Books: store,
		Outcomes: []persistententity.MarketOutcome{
			{Platform: appcommon.PlatformPolymarket, PlatformMarketOutcomeID: "pm-1", YesTokenID: yesTok, NoTokenID: noTok},
			{Platform: appcommon.PlatformKalshi, PlatformMarketOutcomeID: kTicker},
			{Platform: appcommon.PlatformPredict, PlatformMarketOutcomeID: predictID},
		},
	}
	wg.Ops.StrategyParams = func() (float64, float64, float64) { return 0.95, 100, 1 }
	wg.Ops.TradeFee = func(platform, _, _ string) float64 {
		switch strings.ToUpper(strings.TrimSpace(platform)) {
		case appcommon.PlatformKalshi:
			return 0.07
		case appcommon.PlatformPredict:
			return 0.02
		default:
			return 0.05
		}
	}

	r := &Runner{w: wg}
	group := NewOutcomeGroup()
	group.Add(appcommon.PlatformPolymarket, 0)
	group.Add(appcommon.PlatformKalshi, 1)
	group.Add(appcommon.PlatformPredict, 2)

	pick, ok := r.bestFilledPick(group)
	if !ok || pick.Per >= 0.95 {
		t.Fatalf("%+v", pick)
	}
	want := 0.38 + 0.44 + Fee(0.38, 0.05) + Fee(0.44, 0.07)
	if pick.Per < want-0.001 || pick.Per > want+0.001 {
		t.Fatalf("per=%f want≈%f", pick.Per, want)
	}
	got := map[string]bool{pick.Yes.Platform: true, pick.No.Platform: true}
	if !got[appcommon.PlatformPolymarket] || !got[appcommon.PlatformKalshi] {
		t.Fatalf("want Poly+Kalshi best pair, got yes=%s no=%s", pick.Yes.Platform, pick.No.Platform)
	}
	if got[appcommon.PlatformPredict] {
		t.Fatal("Predict prices are worse; must not appear in best pair")
	}
}