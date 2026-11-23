package yesno

import (
	"context"
	"testing"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

func TestPersistableBinarySnap(t *testing.T) {
	if persistableBinarySnap(0.94, 0.95) {
		// ok
	} else {
		t.Fatal("expected persistable")
	}
	if persistableBinarySnap(1.076557, 0.95) {
		t.Fatal("no arb when perShare above 1")
	}
	if persistableBinarySnap(0.96, 0.95) {
		t.Fatal("no arb when perShare >= maxCost")
	}
}

type captureOpportunityRepo struct {
	repository.ArbBinaryOpportunityRepo
	rows []*entity.ArbBinaryOpportunity
}

func (r *captureOpportunityRepo) BatchCreate(_ context.Context, _ *gorm.DB, rows []*entity.ArbBinaryOpportunity) error {
	r.rows = append(r.rows, rows...)
	return nil
}

type captureBacktestOpportunityRepo struct {
	repository.BacktestArbBinaryOpportunityRepo
	rows []*entity.BacktestArbBinaryOpportunity
}

func (r *captureBacktestOpportunityRepo) BatchCreate(_ context.Context, _ *gorm.DB, rows []*entity.BacktestArbBinaryOpportunity) error {
	r.rows = append(r.rows, rows...)
	return nil
}

func TestSaveGrabHitPersistsBinaryLegPlatforms(t *testing.T) {
	hit := GrabHit{
		Pick: FilledPick{
			Yes:      ExecSide{Platform: " kalshi ", Side: Side{NoPx: 0.4, NoSz: 10}},
			No:       ExecSide{Platform: "POLYMARKET", Side: Side{YesPx: 0.5, YesSz: 10}},
			NoBuyYes: true,
			Per:      0.9,
		},
		Leg1:    watchgroup.LegPersistSnap{OrderBooks: `{"platform":"KALSHI"}`},
		Leg2:    watchgroup.LegPersistSnap{OrderBooks: `{"platform":"POLYMARKET"}`},
		MaxCost: 0.95,
		DryRun:  true,
	}
	checkLegs := func(t *testing.T, yes, no, unused *string, yesBook, noBook string) {
		t.Helper()
		if yes == nil || *yes != "POLYMARKET" || yesBook != hit.Leg2.OrderBooks {
			t.Error("YES leg should carry the POLYMARKET platform and snapshot")
		}
		if no == nil || *no != "KALSHI" || noBook != hit.Leg1.OrderBooks {
			t.Error("NO leg should carry the KALSHI platform and snapshot")
		}
		if unused != nil {
			t.Errorf("leg 3 platform = %q, want nil", *unused)
		}
	}
	t.Run("live", func(t *testing.T) {
		repo := &captureOpportunityRepo{}
		SaveGrabHit(t.Context(), PersistGrab{OppRepo: repo}, "test-map", hit)
		if len(repo.rows) != 1 {
			t.Fatalf("saved %d opportunities, want 1", len(repo.rows))
		}
		row := repo.rows[0]
		checkLegs(t, row.Leg1Platform, row.Leg2Platform, row.Leg3Platform, row.Leg1OrderBooks, row.Leg2OrderBooks)
	})
	t.Run("backtest", func(t *testing.T) {
		repo := &captureBacktestOpportunityRepo{}
		SaveGrabHit(t.Context(), PersistGrab{BacktestTaskID: 1, BacktestOppRepo: repo}, "test-map", hit)
		if len(repo.rows) != 1 {
			t.Fatalf("saved %d opportunities, want 1", len(repo.rows))
		}
		row := repo.rows[0]
		checkLegs(t, row.Leg1Platform, row.Leg2Platform, row.Leg3Platform, row.Leg1OrderBooks, row.Leg2OrderBooks)
	})
}
