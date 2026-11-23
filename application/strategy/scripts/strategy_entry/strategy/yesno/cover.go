// Package yesno 实现 5.2 跨所二元事件全覆盖：YES+NO 总成本 < 可配置上限（默认 0.95），结算恒为 1。
package yesno

import (
	"math"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
)

// Limits 门槛（与 strategy_entry.Params 同源）。
type Limits struct {
	MaxCost     float64
	OrderAmount float64
	LevelFactor float64
}

// Quote 单所深度摊薄 YES+NO 的成本（Cover 返回值；线上 5.2 只做跨所卖一 TopHit，不用本结构做发现）。
type Quote struct {
	YesAvg, NoAvg float64
	YesFee, NoFee float64
	Qty, PerShare float64
}

// Fee 每份手续费，coef 为 p(1−p) 系数。
func Fee(px, coef float64) float64 {
	if coef <= 0 || px <= 0 || px >= 1 {
		return 0
	}
	return coef * px * (1 - px)
}

// Cover 卖盘逐级同时买 YES、NO，边际价达 maxCost 即停。
func Cover(yesAsks, noAsks []book.Level, yesFee, noFee, minQty, maxQty, maxCost float64) (Quote, bool) {
	if minQty <= 0 {
		minQty = 1
	}
	if maxQty < minQty || maxCost <= 0 {
		return Quote{}, false
	}
	y, n := positiveAsc(yesAsks), positiveAsc(noAsks)
	i, j := 0, 0
	yLeft, nLeft := 0.0, 0.0
	var yesSpend, noSpend, yesFeeTot, noFeeTot, qty float64
	for qty+1e-9 < maxQty && i < len(y) && j < len(n) {
		if yLeft <= 1e-12 {
			yLeft = y[i].Size
		}
		if nLeft <= 1e-12 {
			nLeft = n[j].Size
		}
		yp, np := y[i].Price, n[j].Price
		if yp+np+Fee(yp, yesFee)+Fee(np, noFee) >= maxCost {
			break
		}
		take := math.Min(yLeft, math.Min(nLeft, maxQty-qty))
		if take <= 1e-12 {
			break
		}
		yesSpend += take * yp
		noSpend += take * np
		yesFeeTot += take * Fee(yp, yesFee)
		noFeeTot += take * Fee(np, noFee)
		qty += take
		yLeft -= take
		nLeft -= take
		if yLeft <= 1e-12 {
			i++
			yLeft = 0
		}
		if nLeft <= 1e-12 {
			j++
			nLeft = 0
		}
	}
	if qty+1e-9 < minQty {
		return Quote{}, false
	}
	q := Quote{
		YesAvg: yesSpend / qty, NoAvg: noSpend / qty,
		YesFee: yesFeeTot / qty, NoFee: noFeeTot / qty,
		Qty: qty, PerShare: (yesSpend + noSpend + yesFeeTot + noFeeTot) / qty,
	}
	if q.PerShare >= maxCost {
		return Quote{}, false
	}
	return q, true
}

// TopHit 卖一套利：跨所两两组合时各买一边的 YES/NO 卖一。
func TopHit(yesPx, yesSz, noPx, noSz, yesFee, noFee, orderAmount, levelFactor, maxCost float64) (qty, per float64, ok bool) {
	if !book.ValidPrice(yesPx) || !book.ValidPrice(noPx) {
		return 0, 0, false
	}
	per = yesPx + noPx + Fee(yesPx, yesFee) + Fee(noPx, noFee)
	if per >= maxCost {
		return 0, 0, false
	}
	qty, ok = SizeForBudget(per, math.Min(yesSz, noSz), orderAmount, levelFactor)
	return qty, per, ok
}

// SizeForBudget keeps the configured depth gate and caps the accepted order at
// the quantity actually offered at the selected price.
func SizeForBudget(cost, depth, orderAmount, levelFactor float64) (float64, bool) {
	if orderAmount <= 0 {
		orderAmount = 100
	}
	if levelFactor <= 0 {
		levelFactor = 2
	}
	target := orderAmount / cost
	if !(cost > 0 && depth > 0 && target > 0 && depth*levelFactor >= target) {
		return 0, false
	}
	return math.Min(target, depth), true
}

func positiveAsc(in []book.Level) []book.Level {
	out := make([]book.Level, 0, len(in))
	for _, lv := range in {
		if lv.Size > 1e-9 && book.ValidPrice(lv.Price) {
			out = append(out, lv)
		}
	}
	return book.SortAsc(out)
}
