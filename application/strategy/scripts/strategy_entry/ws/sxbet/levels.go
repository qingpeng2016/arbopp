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

// LevelsToYesBook 将 SX orderbook_v3（maker frame）合成 YES= outcomeOne 视角的档位簿。
//
// 挂单挂在「自己押的结果」一侧；taker 买 outcomeOne 吃 outcomeTwo，卖 YES/买 outcomeTwo 吃 outcomeOne。
// taker 价 = 1 − makerImplied；size 为 taker 可成交 stake（非 maker resting size）。
// https://docs.sx.bet/developers/order-book
func LevelsToYesBook(one, two []httpentity.SxbetBookLevel) (bids, asks []book.Level) {
	tick := 0.01
	bids = book.SortDesc(book.OnTick(levelsToYesBids(one), tick))
	asks = book.SortAsc(book.OnTick(levelsToYesAsks(two), tick))
	return bids, asks
}

// levelsToYesAsks 买 YES：消费 outcomeTwo 挂单。
func levelsToYesAsks(two []httpentity.SxbetBookLevel) []book.Level {
	out := make([]book.Level, 0, len(two))
	for _, lv := range two {
		m := makerImplied(lv.PercentageOdds)
		p := 1 - m
		sz := takerStake(lv.Size, lv.PercentageOdds)
		if !book.ValidPrice(p) || sz <= 0 {
			continue
		}
		out = append(out, book.Level{Price: p, Size: sz})
	}
	return out
}

// levelsToYesBids 卖 YES：消费 outcomeOne 挂单；YES 买价 = maker 隐含概率。
func levelsToYesBids(one []httpentity.SxbetBookLevel) []book.Level {
	out := make([]book.Level, 0, len(one))
	for _, lv := range one {
		p := makerImplied(lv.PercentageOdds)
		sz := takerStake(lv.Size, lv.PercentageOdds)
		if !book.ValidPrice(p) || sz <= 0 {
			continue
		}
		out = append(out, book.Level{Price: p, Size: sz})
	}
	return out
}

// takerStake size × (10^20 − percentageOdds) / percentageOdds。
func takerStake(size, percentageOdds string) float64 {
	ms := levelSize(size)
	if ms <= 0 {
		return 0
	}
	m := makerImplied(percentageOdds)
	if math.IsNaN(m) || m <= 0 {
		return 0
	}
	return ms * (1 - m) / m
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