package backtest

import (
	"testing"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestReplayFlags(t *testing.T) {
	tests := []struct {
		code        string
		yesno, oneX2 bool
	}{
		{"", true, true},
		{persistententity.BacktestStrategyStrategyEntry, true, true},
		{persistententity.BacktestStrategyYesNo, true, true},
		{persistententity.BacktestStrategyOneX2, false, true},
		{"binary_yes_no", true, false},
	}
	for _, tc := range tests {
		y, o := replayFlags(tc.code)
		if y != tc.yesno || o != tc.oneX2 {
			t.Fatalf("replayFlags(%q) = %v,%v want %v,%v", tc.code, y, o, tc.yesno, tc.oneX2)
		}
	}
}