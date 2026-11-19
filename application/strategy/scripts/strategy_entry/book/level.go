package book

import (
	"math"
	"sort"
	"strconv"
)

// Level 盘口一档。Price 是 0~1 的美元价格，Size 是份额。
type Level struct {
	Price float64 `json:"price"`
	Size  float64 `json:"size"`
}

func Clone(in []Level) []Level {
	if len(in) == 0 {
		return nil
	}
	out := make([]Level, len(in))
	copy(out, in)
	return out
}

func ValidPrice(p float64) bool {
	return p > 0 && p < 1 && !math.IsNaN(p) && !math.IsInf(p, 0)
}

func ParseNum(s string) float64 {
	p, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(p) || math.IsInf(p, 0) {
		return 0
	}
	return p
}

// ParsePrice 解析美元价格。大于 1 时按美分折算。
func ParsePrice(s string) float64 {
	p := ParseNum(s)
	if p > 1 {
		p = p / 100
	}
	return p
}

func PriceOnTick(price, tick float64) bool {
	if tick <= 0 {
		return true
	}
	n := math.Round(price / tick)
	return math.Abs(price-n*tick) <= tick*0.05+1e-8
}

// OnTick 丢掉无效档和不合最小价格档的档位。
func OnTick(in []Level, tick float64) []Level {
	if len(in) == 0 {
		return nil
	}
	out := make([]Level, 0, len(in))
	for _, lv := range in {
		if lv.Size <= 1e-9 || !ValidPrice(lv.Price) || !PriceOnTick(lv.Price, tick) {
			continue
		}
		out = append(out, lv)
	}
	return out
}

func SortAsc(in []Level) []Level {
	out := Clone(in)
	sort.Slice(out, func(i, j int) bool { return out[i].Price < out[j].Price })
	return out
}

func SortDesc(in []Level) []Level {
	out := Clone(in)
	sort.Slice(out, func(i, j int) bool { return out[i].Price > out[j].Price })
	return out
}

// AsksFromOppositeBids Kalshi 只有买盘。对手方买价 p、数量 s 对应本方卖价 1-p、数量 s。
func AsksFromOppositeBids(bids []Level) []Level {
	out := make([]Level, 0, len(bids))
	for _, b := range bids {
		px := 1 - b.Price
		if b.Size <= 1e-9 || !ValidPrice(b.Price) || !ValidPrice(px) {
			continue
		}
		out = append(out, Level{Price: px, Size: b.Size})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Price < out[j].Price })
	return out
}

// BidsFromOppositeAsks 对手方卖价 p、数量 s 对应本方买价 1-p、数量 s（买盘从高到低）。
func BidsFromOppositeAsks(asks []Level) []Level {
	out := make([]Level, 0, len(asks))
	for _, a := range asks {
		px := 1 - a.Price
		if a.Size <= 1e-9 || !ValidPrice(a.Price) || !ValidPrice(px) {
			continue
		}
		out = append(out, Level{Price: px, Size: a.Size})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Price > out[j].Price })
	return out
}

// NoBookFromYesBook Predict 仅推 YES 价：NO token 买盘由 YES 卖盘互补，NO 卖盘由 YES 买盘互补。
func NoBookFromYesBook(yesBids, yesAsks []Level) (noBids, noAsks []Level) {
	return BidsFromOppositeAsks(yesAsks), AsksFromOppositeBids(yesBids)
}

// Upsert 把某一价位的数量设成绝对值。size<=0 删除该档。higherFirst 为买盘从高到低。
func Upsert(levels []Level, price, size float64, higherFirst bool) []Level {
	if !ValidPrice(price) {
		return levels
	}
	for i := range levels {
		if math.Abs(levels[i].Price-price) < 1e-8 {
			if size <= 1e-9 {
				return append(levels[:i], levels[i+1:]...)
			}
			levels[i].Size = size
			return levels
		}
	}
	if size <= 1e-9 {
		return levels
	}
	levels = append(levels, Level{Price: price, Size: size})
	if higherFirst {
		sort.Slice(levels, func(i, j int) bool { return levels[i].Price > levels[j].Price })
	} else {
		sort.Slice(levels, func(i, j int) bool { return levels[i].Price < levels[j].Price })
	}
	return levels
}

// AddSize 按增量改某一价位。Kalshi orderbook_delta 的 delta 是增减，不是绝对值。
func AddSize(levels []Level, price, delta float64, higherFirst bool) []Level {
	cur := 0.0
	for _, lv := range levels {
		if math.Abs(lv.Price-price) < 1e-8 {
			cur = lv.Size
			break
		}
	}
	return Upsert(levels, price, cur+delta, higherFirst)
}
