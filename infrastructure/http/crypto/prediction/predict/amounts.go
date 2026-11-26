package predict

import (
	"math"
	"math/big"
	"sort"
)

type depthLevel struct {
	priceWei *big.Int
	qtyWei   *big.Int
}

type marketAmounts struct {
	pricePerShare  *big.Int
	makerAmount    *big.Int
	takerAmount    *big.Int
	slippageBps    int64
	isMinAmountOut bool
}

func floatSharesToWei(qty float64) *big.Int {
	if qty <= 0 {
		return big.NewInt(0)
	}
	f := new(big.Float).Mul(big.NewFloat(qty), big.NewFloat(weiPrecision))
	i, _ := f.Int(nil)
	return retainSignificantDigits(i, 5)
}

func retainSignificantDigits(num *big.Int, significantDigits int) *big.Int {
	if num == nil || num.Sign() == 0 {
		return big.NewInt(0)
	}
	neg := num.Sign() < 0
	abs := new(big.Int).Abs(num)
	str := abs.String()
	magnitude := len(str)
	excess := magnitude - significantDigits
	if excess <= 0 {
		return new(big.Int).Set(num)
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(excess)), nil)
	quotient := new(big.Int).Div(abs, divisor)
	quotient.Mul(quotient, divisor)
	if neg {
		quotient.Neg(quotient)
	}
	return quotient
}

func asksFromYesBids(bids [][]float64) []depthLevel {
	out := make([]depthLevel, 0, len(bids))
	for _, lv := range bids {
		if len(lv) < 2 {
			continue
		}
		px := lv[0]
		sz := lv[1]
		if px <= 0 || px >= 1 || sz <= 0 {
			continue
		}
		noPx := 1 - px
		if noPx <= 0 || noPx >= 1 {
			continue
		}
		out = append(out, depthLevel{
			priceWei: etherFloatToWei(noPx),
			qtyWei:   floatSharesToWei(sz),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].priceWei.Cmp(out[j].priceWei) < 0
	})
	return out
}

func bidsFromBook(bids [][]float64) []depthLevel {
	out := make([]depthLevel, 0, len(bids))
	for _, lv := range bids {
		if len(lv) < 2 {
			continue
		}
		px, sz := lv[0], lv[1]
		if px <= 0 || sz <= 0 {
			continue
		}
		out = append(out, depthLevel{
			priceWei: etherFloatToWei(px),
			qtyWei:   floatSharesToWei(sz),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].priceWei.Cmp(out[j].priceWei) > 0
	})
	return out
}

func bidsFromYesAsks(asks [][]float64) []depthLevel {
	out := make([]depthLevel, 0, len(asks))
	for _, lv := range asks {
		if len(lv) < 2 {
			continue
		}
		px, sz := lv[0], lv[1]
		if px <= 0 || px >= 1 || sz <= 0 {
			continue
		}
		noPx := 1 - px
		out = append(out, depthLevel{
			priceWei: etherFloatToWei(noPx),
			qtyWei:   floatSharesToWei(sz),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].priceWei.Cmp(out[j].priceWei) > 0
	})
	return out
}

func levelsFromBook(asks [][]float64) []depthLevel {
	out := make([]depthLevel, 0, len(asks))
	for _, lv := range asks {
		if len(lv) < 2 {
			continue
		}
		px, sz := lv[0], lv[1]
		if px <= 0 || sz <= 0 {
			continue
		}
		out = append(out, depthLevel{
			priceWei: etherFloatToWei(px),
			qtyWei:   floatSharesToWei(sz),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].priceWei.Cmp(out[j].priceWei) < 0
	})
	return out
}

// filterAsksAtOrBelow 买单限价：只吃价 ≤ limit 的卖档（FAK/市价穿透上限）。
func filterAsksAtOrBelow(asks []depthLevel, limit float64) []depthLevel {
	if !(limit > 0 && limit < 1) || len(asks) == 0 {
		return asks
	}
	capWei := etherFloatToWei(limit)
	out := make([]depthLevel, 0, len(asks))
	for _, lv := range asks {
		if lv.priceWei != nil && lv.priceWei.Cmp(capWei) <= 0 {
			out = append(out, lv)
		}
	}
	return out
}

// filterBidsAtOrAbove 卖单限价：只吃价 ≥ limit 的买档。
func filterBidsAtOrAbove(bids []depthLevel, limit float64) []depthLevel {
	if !(limit > 0 && limit < 1) || len(bids) == 0 {
		return bids
	}
	floorWei := etherFloatToWei(limit)
	out := make([]depthLevel, 0, len(bids))
	for _, lv := range bids {
		if lv.priceWei != nil && lv.priceWei.Cmp(floorWei) >= 0 {
			out = append(out, lv)
		}
	}
	return out
}

func etherFloatToWei(px float64) *big.Int {
	f := new(big.Float).Mul(big.NewFloat(px), big.NewFloat(weiPrecision))
	i, _ := f.Int(nil)
	return i
}

func processBook(levels []depthLevel, qtyWei *big.Int) (priceWei, quantityWei, lastPriceWei *big.Int) {
	remaining := new(big.Int).Set(qtyWei)
	priceAcc := big.NewInt(0)
	qtyAcc := big.NewInt(0)
	last := big.NewInt(0)
	prec := big.NewInt(int64(weiPrecision))
	for _, lv := range levels {
		if remaining.Sign() <= 0 {
			break
		}
		take := new(big.Int).Set(lv.qtyWei)
		if take.Cmp(remaining) > 0 {
			take = new(big.Int).Set(remaining)
		}
		// priceWei += price * take (without intermediate div)
		term := new(big.Int).Mul(lv.priceWei, take)
		priceAcc.Add(priceAcc, term)
		qtyAcc.Add(qtyAcc, take)
		last = lv.priceWei
		remaining.Sub(remaining, take)
	}
	if qtyAcc.Sign() == 0 {
		return big.NewInt(0), big.NewInt(0), big.NewInt(0)
	}
	avg := new(big.Int).Div(priceAcc, qtyAcc)
	_ = prec
	return avg, qtyAcc, last
}

// marketAmountsAtLimit 用策略限价+数量直接算签名金额，不再 REST 拉盘口。
// 套利路径：内存簿 SizeCheck 已通过后并发发单；此处按限价封顶/托底授权成交。
// isMinAmountOut（买）：maker 固定为限价×数量，taker 按 slippage 下调（少收份额）；卖始终下调 taker USDT。
func marketAmountsAtLimit(qty, priceLimit float64, sell bool, slippageBps int64, isMinAmountOut bool) (marketAmounts, bool) {
	if !(qty > 0) || !(priceLimit > 0 && priceLimit < 1) {
		return marketAmounts{}, false
	}
	if slippageBps < 0 {
		slippageBps = 0
	}
	if slippageBps > 10_000 {
		slippageBps = 10_000
	}
	qtyWei := floatSharesToWei(qty)
	if qtyWei.Cmp(big.NewInt(int64(1e16))) < 0 {
		return marketAmounts{}, false
	}
	qtyWei = retainSignificantDigits(qtyWei, 5)
	priceWei := etherFloatToWei(priceLimit)
	prec := big.NewInt(int64(weiPrecision))
	if sell {
		baseTaker := new(big.Int).Mul(priceWei, qtyWei)
		baseTaker.Div(baseTaker, prec)
		takerAmount := new(big.Int).Set(baseTaker)
		if slippageBps > 0 {
			takerAmount.Mul(baseTaker, big.NewInt(10_000-slippageBps))
			takerAmount.Div(takerAmount, big.NewInt(10_000))
		}
		return marketAmounts{
			pricePerShare: priceWei,
			makerAmount:   new(big.Int).Set(qtyWei),
			takerAmount:   takerAmount,
			slippageBps:   slippageBps,
		}, true
	}
	baseMaker := new(big.Int).Mul(priceWei, qtyWei)
	baseMaker.Div(baseMaker, prec)
	if isMinAmountOut {
		takerAmount := new(big.Int).Set(qtyWei)
		if slippageBps > 0 {
			takerAmount.Mul(qtyWei, big.NewInt(10_000-slippageBps))
			takerAmount.Div(takerAmount, big.NewInt(10_000))
		}
		return marketAmounts{
			pricePerShare:  priceWei,
			makerAmount:    new(big.Int).Set(baseMaker),
			takerAmount:    takerAmount,
			slippageBps:    slippageBps,
			isMinAmountOut: true,
		}, true
	}
	makerAmount := new(big.Int).Set(baseMaker)
	if slippageBps > 0 {
		num := new(big.Int).Mul(baseMaker, big.NewInt(10_000+slippageBps))
		num.Div(num, big.NewInt(10_000))
		if num.Cmp(qtyWei) < 0 {
			makerAmount = num
		} else {
			makerAmount = new(big.Int).Set(qtyWei)
		}
	}
	return marketAmounts{
		pricePerShare: priceWei,
		makerAmount:   makerAmount,
		takerAmount:   new(big.Int).Set(qtyWei),
		slippageBps:   slippageBps,
	}, true
}

// marketBuyByQuantity 对齐 SDK 默认 MARKET BUY（非 isMinAmountOut）。
func marketBuyByQuantity(asks []depthLevel, qtyWei *big.Int, slippageBps int64) (marketAmounts, bool) {
	if qtyWei == nil || qtyWei.Cmp(big.NewInt(int64(1e16))) < 0 || len(asks) == 0 {
		return marketAmounts{}, false
	}
	qty := retainSignificantDigits(qtyWei, 5)
	priceWei, quantityWei, lastPriceWei := processBook(asks, qty)
	if quantityWei.Sign() == 0 || lastPriceWei.Sign() == 0 {
		return marketAmounts{}, false
	}
	prec := big.NewInt(int64(weiPrecision))
	baseMaker := new(big.Int).Mul(lastPriceWei, quantityWei)
	baseMaker.Div(baseMaker, prec)
	makerAmount := new(big.Int).Set(baseMaker)
	if slippageBps > 0 {
		num := new(big.Int).Mul(baseMaker, big.NewInt(10_000+slippageBps))
		num.Div(num, big.NewInt(10_000))
		if num.Cmp(quantityWei) < 0 {
			makerAmount = num
		} else {
			makerAmount = new(big.Int).Set(quantityWei)
		}
	}
	return marketAmounts{
		pricePerShare: priceWei,
		makerAmount:   makerAmount,
		takerAmount:   new(big.Int).Set(quantityWei),
		slippageBps:   slippageBps,
	}, true
}

func marketSellByQuantity(bids []depthLevel, qtyWei *big.Int, slippageBps int64) (marketAmounts, bool) {
	if qtyWei == nil || qtyWei.Cmp(big.NewInt(int64(1e16))) < 0 || len(bids) == 0 {
		return marketAmounts{}, false
	}
	qty := retainSignificantDigits(qtyWei, 5)
	priceWei, quantityWei, lastPriceWei := processBook(bids, qty)
	if quantityWei.Sign() == 0 || lastPriceWei.Sign() == 0 {
		return marketAmounts{}, false
	}
	prec := big.NewInt(int64(weiPrecision))
	baseTaker := new(big.Int).Mul(lastPriceWei, quantityWei)
	baseTaker.Div(baseTaker, prec)
	takerAmount := new(big.Int).Set(baseTaker)
	if slippageBps > 0 {
		takerAmount.Mul(baseTaker, big.NewInt(10_000-slippageBps))
		takerAmount.Div(takerAmount, big.NewInt(10_000))
	}
	return marketAmounts{
		pricePerShare: priceWei,
		makerAmount:   new(big.Int).Set(quantityWei),
		takerAmount:   takerAmount,
		slippageBps:   slippageBps,
	}, true
}

func weiToFloatShares(w *big.Int) float64 {
	if w == nil || w.Sign() <= 0 {
		return 0
	}
	f, _ := new(big.Float).Quo(new(big.Float).SetInt(w), big.NewFloat(weiPrecision)).Float64()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return f
}