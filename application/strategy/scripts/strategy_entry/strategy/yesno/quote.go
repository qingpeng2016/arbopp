package yesno

import (
	"math"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
)

// maxBookAge limits the age of every selected leg at the scan time.
const maxBookAge = 5 * time.Second

// QuoteExecSide reads both sides from the same strategy snapshot.
func QuoteExecSide(w *watchgroup.Watch, view book.Snapshot, index int) (ExecSide, bool) {
	oc := w.OutcomeAt(index)
	if oc == nil {
		return ExecSide{}, false
	}
	platform := strings.ToUpper(strings.TrimSpace(oc.Platform))
	rate := TradeFeeRate(platform, deref(oc.Category), deref(oc.SubCategory), oc.Config, w.Ops.TradeFee)
	side := ExecSide{Side: Side{Index: index, Fee: rate}, OutcomeID: oc.ID, Platform: platform,
		MarketID: oc.PlatformMarketOutcomeID, YesToken: oc.YesTokenID, NoToken: oc.NoTokenID,
		yesQuote: w.Quote(view, index, true), noQuote: w.Quote(view, index, false)}
	side.YesPx, side.YesSz, _ = side.yesQuote.Top()
	side.NoPx, side.NoSz, _ = side.noQuote.Top()
	return side, side.YesPx > 0 || side.NoPx > 0
}

func (s ExecSide) quote(buyYes bool) watchgroup.LegQuote {
	if buyYes {
		return s.yesQuote
	}
	return s.noQuote
}

// PayoutQuote returns the cost and top depth per unit of net payout at a valid quote.
func (s ExecSide) PayoutQuote(buyYes bool, at time.Time) (cost, depth float64, ok bool) {
	q := s.quote(buyYes)
	price, size, ok := q.Top()
	age := at.Sub(q.UpdatedAt())
	if !ok || q.UpdatedAt().IsZero() || age < 0 || age > maxBookAge ||
		math.IsNaN(size) || math.IsInf(size, 0) || s.Fee < 0 || s.Fee >= 1 || math.IsNaN(s.Fee) {
		return 0, 0, false
	}
	cost = NetCostPerShare(s.Platform, price, s.Fee)
	if cost <= 0 {
		return 0, 0, false
	}
	switch s.Platform {
	case appcommon.PlatformPredict:
		ratio := PredictNetRatio(price, s.Fee)
		if ratio <= 0 {
			return 0, 0, false
		}
		depth = size * ratio
		return cost, depth, true
	case appcommon.PlatformSxbet:
		netRatio := 1 - s.Fee*(1-price)
		if netRatio <= 0 {
			return 0, 0, false
		}
		depth = size / price * netRatio
		return cost, depth, depth > 0 && !math.IsInf(depth, 0)
	default:
		depth = size
		return cost, depth, true
	}
}

// TopAskSz 该腿选定方向卖一价与原始挂单量（leg_*_best_ask / arb_qty 口径）。
func (s ExecSide) TopAskSz(buyYes bool) (px, sz float64, ok bool) {
	return s.quote(buyYes).Top()
}

// PredictFeeSharesFitBook 仅对 Predict：OrderShares 放大后核对内存卖一是否够量。
// 不够则整笔套利不应发单（所有平台都不发）。非 Predict 腿直接通过。
func (s ExecSide) PredictFeeSharesFitBook(buyYes bool, netQty float64) (ok bool, need, have float64) {
	if !strings.EqualFold(strings.TrimSpace(s.Platform), appcommon.PlatformPredict) {
		return true, 0, 0
	}
	need = s.OrderShares(buyYes, netQty)
	if _, sz, topOK := s.TopAskSz(buyYes); topOK {
		have = sz
	} else if buyYes {
		have = s.YesSz
	} else {
		have = s.NoSz
	}
	if !(have > 0) {
		return false, need, have
	}
	return need <= have+1e-12, need, have
}

// OrderShares 将策略净兑付目标换成平台下单量。
// Predict：扣费前份额；SX：USDC stake（totalBetSize）；其余：份额原样。
func (s ExecSide) OrderShares(buyYes bool, netQty float64) float64 {
	if netQty <= 0 {
		return 0
	}
	price := SidePx(s.Side, buyYes)
	if !book.ValidPrice(price) {
		return netQty
	}
	switch s.Platform {
	case appcommon.PlatformPredict:
		ratio := PredictNetRatio(price, s.Fee)
		if ratio <= 0 {
			return 0
		}
		return netQty / ratio
	case appcommon.PlatformSxbet:
		netRatio := 1 - s.Fee*(1-price)
		if netRatio <= 0 {
			return 0
		}
		return netQty * price / netRatio
	default:
		return netQty
	}
}

// FillToNet 将平台回报成交量折回策略净兑付单位（与 OrderShares 互逆）。
func (s ExecSide) FillToNet(buyYes bool, fill float64) float64 {
	if fill <= 0 {
		return 0
	}
	price := SidePx(s.Side, buyYes)
	if !book.ValidPrice(price) {
		return fill
	}
	switch s.Platform {
	case appcommon.PlatformPredict:
		return fill * PredictNetRatio(price, s.Fee)
	case appcommon.PlatformSxbet:
		netRatio := 1 - s.Fee*(1-price)
		if netRatio <= 0 || price <= 0 {
			return 0
		}
		return fill / price * netRatio
	default:
		return fill
	}
}

func (s ExecSide) PersistQuote(buyYes bool, cost, size float64) watchgroup.LegPersistSnap {
	return s.quote(buyYes).Persist(cost, size, s.Fee)
}

// SameSettlementScope excludes SX combinations without a common void policy and
// rejects a known disagreement in the fixture start time.
func SameSettlementScope(w *watchgroup.Watch, a, b ExecSide) bool {
	if a.Platform == b.Platform {
		return true
	}
	if a.Platform == appcommon.PlatformSxbet || b.Platform == appcommon.PlatformSxbet {
		return false
	}
	x, y := fixtureStart(w, a.Index), fixtureStart(w, b.Index)
	return x.IsZero() || y.IsZero() || x.Equal(y)
}

func fixtureStart(w *watchgroup.Watch, index int) time.Time {
	oc := w.OutcomeAt(index)
	if oc == nil {
		return time.Time{}
	}
	parent := w.MarketsByRowID[oc.MarketRowID]
	for _, raw := range []*string{parent.EventStartTime, oc.MarketStartTime} {
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
			if at, err := time.Parse(layout, strings.TrimSpace(deref(raw))); err == nil {
				return at
			}
		}
	}
	return time.Time{}
}
