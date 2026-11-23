// The offline audit overlays this file into the strategy package to exercise
// the current pricing and persistence paths.
package onextwo

import (
	"math"
	"testing"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	sxbet "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/sxbet"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func probeLevels(price float64) []book.Level {
	return []book.Level{{Price: price, Size: 1000}}
}

func TestOpportunityAuditProbes(t *testing.T) {
	t.Run("binary_order_depth", func(t *testing.T) {
		qty, _, hit := yesno.TopHit(.4, 60, .4, 60, 0, 0, 90, 2, .95)
		if !hit || qty != 60 {
			t.Fatalf("expected depth-capped 60 shares, qty=%v hit=%v", qty, hit)
		}
	})

	t.Run("predict_buy_fee_and_frozen_quotes", func(t *testing.T) {
		store := book.NewStore()
		now := time.Now()
		store.ApplyPredictBook("p", probeLevels(.3), probeLevels(.5), now)
		store.ApplyPolyBook("no", nil, probeLevels(.493), .01, 1, now)
		config := `{"feeRateBps":200}`
		w := &watchgroup.Watch{Books: store, Outcomes: []entity.MarketOutcome{
			{Platform: "PREDICT", PlatformMarketOutcomeID: "p", Config: &config},
			{Platform: "POLYMARKET", NoTokenID: "no"},
		}}
		var group yesno.OutcomeGroup
		group.Add("PREDICT", 0)
		group.Add("POLYMARKET", 1)
		if _, _, hit := yesno.Entry(w).AllFilledPicks(group, yesno.Limits{MaxCost: .999}); hit {
			t.Fatal("Predict fees should turn this candidate unprofitable")
		}
		store.ApplyPolyBook("no", nil, probeLevels(.4), .01, 1, now)
		_, pick, hit := yesno.Entry(w).AllFilledPicks(group, yesno.Limits{MaxCost: .999})
		if !hit || math.Abs(pick.Per-(.5/.98+.4)) > 1e-9 || pick.Sizes[0] != 980 {
			t.Fatalf("expected equal net payout and adjusted depth, pick=%+v", pick)
		}
		store.ApplyPolyLevel("no", "SELL", .4, 1, now)
		store.ApplyPolyLevel("no", "SELL", .3, 2, now)
		snap := pick.No.PersistQuote(false, pick.Costs[1], pick.Sizes[1])
		if !snap.OK || snap.TopAsk != .4 || snap.TopSz != 1000 {
			t.Fatalf("saved quote must match the quote used at selection: %+v", snap)
		}
	})

	t.Run("sxbet_size_units", func(t *testing.T) {
		_, asks := sxbet.LevelsToYesBook(nil, []httpentity.SxbetBookLevel{{
			PercentageOdds: "52000000000000000000", Size: "52000000",
		}})
		store := book.NewStore()
		store.ApplySxbetBook("sx", nil, asks, time.Now())
		w := &watchgroup.Watch{Books: store, Outcomes: []entity.MarketOutcome{
			{Platform: "SXBET", PlatformMarketOutcomeID: "sx"},
		}}
		w.Ops.TradeFee = func(string, string, string) float64 { return .01 }
		view, at := w.StrategySnapshot()
		side, _ := yesno.QuoteExecSide(w, view, 0)
		cost, depth, ok := side.PayoutQuote(true, at)
		if !ok || math.Abs(depth-99.48) > 1e-9 || math.Abs(cost-.48/.9948) > 1e-9 {
			t.Fatalf("48 USDC at .48 should cover 99.48 net payout units, got %f @ %f, ok=%v", depth, cost, ok)
		}
		if yesno.SameSettlementScope(w, side, yesno.ExecSide{Platform: "POLYMARKET"}) {
			t.Fatal("cross-venue SX void policy has no verified equivalent")
		}
	})

	t.Run("reject_stale_and_mixed_quotes", func(t *testing.T) {
		store := book.NewStore()
		w := &watchgroup.Watch{Books: store, Outcomes: []entity.MarketOutcome{
			{Platform: "POLYMARKET", YesTokenID: "a"},
			{Platform: "PREDICT", PlatformMarketOutcomeID: "b"},
		}}
		now := time.Now()
		store.ApplyPolyBook("a", nil, probeLevels(.4), .01, 1, now.Add(-24*time.Hour))
		store.ApplyPredictBook("b", probeLevels(.6), probeLevels(.95), now)
		var group yesno.OutcomeGroup
		group.Add("POLYMARKET", 0)
		group.Add("PREDICT", 1)
		run := yesno.Entry(w)
		if _, _, hit := run.AllFilledPicks(group, yesno.Limits{MaxCost: .95}); hit {
			t.Fatal("old Poly book must not be combined with a current Predict book")
		}

		store.ApplyPredictBook("b", probeLevels(.3), probeLevels(.95), now)
		store.ApplyPolyBook("a", nil, probeLevels(.4), .01, 1, now)
		view, at := w.StrategySnapshot()
		store.ApplyPolyBook("a", nil, probeLevels(.7), .01, 1, now)
		store.ApplyPredictBook("b", probeLevels(.6), probeLevels(.95), now)
		a, _ := yesno.QuoteExecSide(w, view, 0)
		b, _ := yesno.QuoteExecSide(w, view, 1)
		ac, _, _ := a.PayoutQuote(true, at)
		bc, _, _ := b.PayoutQuote(false, at)
		if math.Abs(ac+bc-1.1) > 1e-9 {
			t.Fatalf("immutable snapshot mixed quotes, got %f", ac+bc)
		}
	})

	t.Run("triple_persistence", func(t *testing.T) {
		store := book.NewStore()
		w := &watchgroup.Watch{Books: store, Outcomes: []entity.MarketOutcome{
			{Platform: "PREDICT", PlatformMarketOutcomeID: "draw"},
			{Platform: "POLYMARKET", YesTokenID: "team1"},
			{Platform: "PREDICT", PlatformMarketOutcomeID: "team2"},
		}}
		now := time.Now()
		store.ApplyPredictBook("draw", nil, probeLevels(.2), now)
		store.ApplyPolyBook("team1", nil, probeLevels(.3), .01, 1, now)
		store.ApplyPredictBook("team2", nil, probeLevels(.4), now)
		w.Ops.TradeFee = func(string, string, string) float64 { return .02 }
		r := Runner{w: w}
		lim := Limits{YesTripleMax: .95, LevelFactor: 1, OrderAmount: 90}
		view, at := w.StrategySnapshot()
		var legs [3]pricedLeg
		for i := range legs {
			side, _ := yesno.QuoteExecSide(w, view, i)
			cost, size, ok := side.PayoutQuote(true, at)
			if !ok {
				t.Fatalf("missing quote %d", i)
			}
			legs[i] = pricedLeg{side: side, cost: cost, size: size}
		}
		pick, hit := r.priceTriple(legs, at, true, lim)
		if !hit {
			t.Fatal("valid cross-venue triple did not trigger")
		}
		store.ApplyPredictBook("draw", nil, probeLevels(.8), now)
		row := buildOppBacktest(persistDeps{}, "audit-fixture", GrabHit{Pick: pick, Legs: r.snapTripleLegs(pick), Limits: lim})
		want := .2/.98 + .3 + .02*.3*.7 + .4/.98
		if row == nil || row.Leg1BestAsk == nil || row.Leg1BestAsk.String() != "0.2" ||
			math.Abs(row.TotalCost.InexactFloat64()-want) > 1e-8 ||
			row.SnapshotAtMs != at.UnixMilli() {
			t.Fatalf("triple must save the raw price and original scan time: %+v", row)
		}
		legs[1].side.Platform = "PREDICT"
		if _, hit := r.priceTriple(legs, at, true, lim); hit {
			t.Fatal("same-venue triple must not be reported as a cross-venue opportunity")
		}
	})
}
