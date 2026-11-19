package backtest

import (
	"strings"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// replayFlags 按 backtest_task.strategy_code 决定回放哪些策略。
func replayFlags(strategyCode string) (runYesno, runOneX2 bool) {
	c := strings.ToLower(strings.TrimSpace(strategyCode))
	switch c {
	case "", persistententity.BacktestStrategyStrategyEntry, persistententity.BacktestStrategyYesNo:
		return true, true
	case persistententity.BacktestStrategyOneX2, "1x2":
		return false, true
	case "binary_yes_no", "binary":
		return true, false
	default:
		return true, true
	}
}