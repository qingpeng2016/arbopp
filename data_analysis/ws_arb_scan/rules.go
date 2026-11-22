package main

import "math"

// 与 strategy_entry 默认 config 对齐（见 groups.go / yesno.entry limits）。
type scanLimits struct {
	maxCostYesNo     float64 // binary_yes_no_max_cost
	orderAmount      float64 // binary_yes_no_order_amount
	levelFactor      float64 // binary_yes_no_level_factor
	maxCostTripleYes float64 // 1x2_yes_triple_max_cost
	maxCostTripleNo  float64 // 1x2_no_triple_max_cost
}

func defaultScanLimits() scanLimits {
	return scanLimits{
		maxCostYesNo:     0.95,
		orderAmount:      100,
		levelFactor:      2,
		maxCostTripleYes: 0.95,
		maxCostTripleNo:  1.90,
	}
}

type platformFee struct {
	sportsCoef float64 // 吃单费系数：fee = coef * p * (1-p)
}

func defaultFees() map[string]platformFee {
	return map[string]platformFee{
		platPoly:    {0.05},
		platKalshi:  {0.07},
		platPredict: {0.02},
		platSxbet:   {0.02},
	}
}

func validPrice(p float64) bool {
	return p > 0 && p < 1 && !math.IsNaN(p) && !math.IsInf(p, 0)
}

func tradeFee(p, coef float64) float64 {
	if coef <= 0 || !validPrice(p) {
		return 0
	}
	return coef * p * (1 - p)
}

// crossYesNoHit 复刻 yesno.TopHit：甲所 YES 卖一 + 乙所 NO 卖一，含双费、深度校验。
func crossYesNoHit(yesPx, yesSz, noPx, noSz, feeYes, feeNo float64, lim scanLimits) (qty, per float64, ok bool) {
	if yesSz <= 0 || noSz <= 0 || lim.maxCostYesNo <= 0 {
		return 0, 0, false
	}
	if !validPrice(yesPx) || !validPrice(noPx) {
		return 0, 0, false
	}
	per = yesPx + noPx + tradeFee(yesPx, feeYes) + tradeFee(noPx, feeNo)
	if per >= lim.maxCostYesNo {
		return 0, 0, false
	}
	pair := yesPx + noPx
	if pair <= 0 {
		return 0, 0, false
	}
	qty = lim.orderAmount / pair
	avail := yesSz
	if noSz < avail {
		avail = noSz
	}
	if avail*lim.levelFactor+1e-9 < qty {
		return 0, 0, false
	}
	return qty, per, true
}

// crossYesNoPer 只算成本（含费），不做深度与 maxCost 门槛；用于看「离套利还差多少」。
func crossYesNoPer(yesPx, yesSz, noPx, noSz, feeYes, feeNo float64) (per float64, ok bool) {
	if yesSz <= 0 || noSz <= 0 || !validPrice(yesPx) || !validPrice(noPx) {
		return 0, false
	}
	per = yesPx + noPx + tradeFee(yesPx, feeYes) + tradeFee(noPx, feeNo)
	return per, true
}

// tripleLegCost 单腿：买 YES 或 NO 的卖一价 + 手续费。
func tripleLegCost(l legQuote, buyYes bool, feeCoef float64) (px, sz float64, ok bool) {
	if buyYes {
		px, sz = l.YesPx, l.YesSz
	} else {
		px, sz = l.NoPx, l.NoSz
	}
	if !validPrice(px) || sz <= 0 {
		return 0, 0, false
	}
	return px + tradeFee(px, feeCoef), sz, true
}

// tripleHitCheck 三路 moneyline：各腿最低价之和 < 阈值，深度同 TopHit。
func tripleHitCheck(a, b, c legQuote, fa, fb, fc float64, buyYes bool, maxCost float64, lim scanLimits) (per, qty float64, ok bool) {
	pa, sa, oka := tripleLegCost(a, buyYes, fa)
	pb, sb, okb := tripleLegCost(b, buyYes, fb)
	pc, sc, okc := tripleLegCost(c, buyYes, fc)
	if !oka || !okb || !okc {
		return 0, 0, false
	}
	per = pa + pb + pc
	if per >= maxCost {
		return 0, 0, false
	}
	minSz := sa
	if sb < minSz {
		minSz = sb
	}
	if sc < minSz {
		minSz = sc
	}
	qty = lim.orderAmount / per
	if minSz*lim.levelFactor+1e-9 < qty {
		return 0, 0, false
	}
	return per, qty, true
}