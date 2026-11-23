package onextwo

import (
	"context"
	"testing"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

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

func TestSaveGrabHitPersistsTripleLegPlatforms(t *testing.T) {
	hit := GrabHit{
		Pick: TriplePick{
			Kind:   OppKindYesTriple,
			BuyYes: true,
			Legs: [3]yesno.ExecSide{
				{Platform: "KALSHI"},
				{Platform: "PREDICT"},
				{Platform: "POLYMARKET"},
			},
			Prices:   [3]float64{0.2, 0.4, 0.3},
			LegSizes: [3]float64{8, 10, 9},
			Per:      0.9,
		},
		Legs: [3]watchgroup.LegPersistSnap{
			{OrderBooks: `{"platform":"KALSHI"}`},
			{OrderBooks: `{"platform":"PREDICT"}`},
			{OrderBooks: `{"platform":"POLYMARKET"}`},
		},
		Limits: Limits{YesTripleMax: 0.95},
		DryRun: true,
	}
	checkLegs := func(t *testing.T, platforms [3]*string, books [3]string) {
		t.Helper()
		for i, leg := range hit.Pick.Legs {
			if platforms[i] == nil || *platforms[i] != leg.Platform || books[i] != hit.Legs[i].OrderBooks {
				t.Errorf("leg %d should carry the %s platform and snapshot", i+1, leg.Platform)
			}
		}
	}
	t.Run("live", func(t *testing.T) {
		repo := &captureOpportunityRepo{}
		SaveGrabHit(t.Context(), persistDeps{opp: repo}, "test-map", hit)
		if len(repo.rows) != 1 {
			t.Fatalf("saved %d opportunities, want 1", len(repo.rows))
		}
		row := repo.rows[0]
		checkLegs(t,
			[3]*string{row.Leg1Platform, row.Leg2Platform, row.Leg3Platform},
			[3]string{row.Leg1OrderBooks, row.Leg2OrderBooks, row.Leg3OrderBooks},
		)
	})
	t.Run("backtest", func(t *testing.T) {
		repo := &captureBacktestOpportunityRepo{}
		SaveGrabHit(t.Context(), persistDeps{task: 1, btOpp: repo}, "test-map", hit)
		if len(repo.rows) != 1 {
			t.Fatalf("saved %d opportunities, want 1", len(repo.rows))
		}
		row := repo.rows[0]
		checkLegs(t,
			[3]*string{row.Leg1Platform, row.Leg2Platform, row.Leg3Platform},
			[3]string{row.Leg1OrderBooks, row.Leg2OrderBooks, row.Leg3OrderBooks},
		)
	})
}
