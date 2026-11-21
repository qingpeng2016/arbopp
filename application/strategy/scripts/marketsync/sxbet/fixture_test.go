package sxbet

import (
	"testing"

	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func TestSxbetOutcomeTitle(t *testing.T) {
	line := -0.25
	tests := []struct {
		name string
		m    httpentity.SxbetMarket
		want string
	}{
		{
			name: "1x2 home",
			m: httpentity.SxbetMarket{
				Type: 1, TeamOneName: "Kilmarnock", TeamTwoName: "Aberdeen",
				OutcomeOneName: "Kilmarnock", OutcomeTwoName: "Not Kilmarnock",
			},
			want: "Kilmarnock vs Not Kilmarnock [1X2]",
		},
		{
			name: "1x2 draw",
			m: httpentity.SxbetMarket{
				Type: 1,
				OutcomeOneName: "Tie", OutcomeTwoName: "Not tie",
			},
			want: "Tie vs Not tie [1X2]",
		},
		{
			name: "spread",
			m: httpentity.SxbetMarket{
				Type: 3, Line: &line,
				OutcomeOneName: "Kilmarnock -0.25", OutcomeTwoName: "Aberdeen +0.25",
			},
			want: "Kilmarnock -0.25 vs Aberdeen +0.25 [Spread line=-0.25]",
		},
		{
			name: "total",
			m: httpentity.SxbetMarket{
				Type: 2, Line: ptrFloat(3),
				OutcomeOneName: "Over 3.0", OutcomeTwoName: "Under 3.0",
			},
			want: "Over 3.0 vs Under 3.0 [Total line=3]",
		},
		{
			name: "moneyline",
			m: httpentity.SxbetMarket{
				Type: 52,
				OutcomeOneName: "Kilmarnock", OutcomeTwoName: "Aberdeen",
			},
			want: "Kilmarnock vs Aberdeen [Moneyline]",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sxbetOutcomeTitle(tc.m, cryptoPredictionEnum.OutcomeMarketTypeMoneyline)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func ptrFloat(f float64) *float64 { return &f }