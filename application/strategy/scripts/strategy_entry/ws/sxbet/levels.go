package sxbet

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

const oddsDenom = 1e20

// LevelsToYesBook outcomeOne → YES 卖盘，outcomeTwo → YES 买盘（与 Predict 单簿模型一致）。
func LevelsToYesBook(one, two []httpentity.SxbetBookLevel) (bids, asks []book.Level) {
	tick := 0.01
	bids = book.SortDesc(book.OnTick(levelsToYesBids(two), tick))
	asks = book.SortAsc(book.OnTick(levelsToYesAsks(one), tick))
	return bids, asks
}

func levelsToYesAsks(one []httpentity.SxbetBookLevel) []book.Level {
	out := make([]book.Level, 0, len(one))
	for _, lv := range one {
		p := makerImplied(lv.PercentageOdds)
		sz := levelSize(lv.Size)
		if !book.ValidPrice(p) || sz <= 0 {
			continue
		}
		out = append(out, book.Level{Price: p, Size: sz})
	}
	return out
}

func levelsToYesBids(two []httpentity.SxbetBookLevel) []book.Level {
	out := make([]book.Level, 0, len(two))
	for _, lv := range two {
		p := 1 - makerImplied(lv.PercentageOdds)
		sz := levelSize(lv.Size)
		if !book.ValidPrice(p) || sz <= 0 {
			continue
		}
		out = append(out, book.Level{Price: p, Size: sz})
	}
	return out
}

func makerImplied(percentageOdds string) float64 {
	percentageOdds = strings.TrimSpace(percentageOdds)
	if percentageOdds == "" {
		return math.NaN()
	}
	r := new(big.Rat)
	if _, ok := r.SetString(percentageOdds); !ok {
		f, err := strconv.ParseFloat(percentageOdds, 64)
		if err != nil {
			return math.NaN()
		}
		return f / oddsDenom
	}
	den := new(big.Rat).SetFloat64(oddsDenom)
	v := new(big.Rat).Quo(r, den)
	f, _ := v.Float64()
	return f
}

func levelSize(size string) float64 {
	size = strings.TrimSpace(size)
	if size == "" {
		return 0
	}
	f, err := strconv.ParseFloat(size, 64)
	if err != nil {
		return 0
	}
	// SX base units → 与 Poly 档位同量级展示（÷ 1e6 近似 USDC 份）。
	return f / 1e6
}