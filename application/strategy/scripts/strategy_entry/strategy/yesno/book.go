package yesno

import (
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
)

// BookOutcome 读盘所需字段。
type BookOutcome struct {
	Platform, MarketID, YesToken, NoToken string
}

func OutcomeFields(platform, marketID, yesTok, noTok string) BookOutcome {
	return BookOutcome{Platform: platform, MarketID: marketID, YesToken: yesTok, NoToken: noTok}
}

// Side 一家交易所 YES/NO 卖一（跨所时：A 所 YES 卖一 + B 所 NO 卖一组合）。
type Side struct {
	Index                    int
	YesPx, YesSz, NoPx, NoSz float64
	Fee                      float64
}

// TopFromStore 读卖一。
func TopFromStore(store *book.Store, oc BookOutcome, index int, fee float64) (Side, bool) {
	st := Side{Index: index, Fee: fee}
	switch strings.ToUpper(strings.TrimSpace(oc.Platform)) {
	case appcommon.PlatformPolymarket:
		yesID, noID := strings.TrimSpace(oc.YesToken), strings.TrimSpace(oc.NoToken)
		if yesID == "" || noID == "" || yesID == noID {
			return Side{}, false
		}
		var ok1, ok2 bool
		st.YesPx, st.YesSz, ok1 = store.Ask(appcommon.PlatformPolymarket, yesID)
		st.NoPx, st.NoSz, ok2 = store.Ask(appcommon.PlatformPolymarket, noID)
		if !ok1 || !ok2 || !book.ValidPrice(st.YesPx) || !book.ValidPrice(st.NoPx) {
			return Side{}, false
		}
		return st, true
	case appcommon.PlatformKalshi, appcommon.PlatformPredict, appcommon.PlatformSxbet:
		bookID := strings.TrimSpace(oc.MarketID)
		var ok1, ok2 bool
		st.YesPx, st.YesSz, ok1 = store.Ask(oc.Platform, bookID)
		bid, bidSz, ok2 := store.Bid(oc.Platform, bookID)
		if !ok1 || !ok2 || !book.ValidPrice(st.YesPx) {
			return Side{}, false
		}
		st.NoPx, st.NoSz = 1-bid, bidSz
		if !book.ValidPrice(st.NoPx) {
			return Side{}, false
		}
		return st, true
	default:
		return Side{}, false
	}
}

// SidePx 下单方向卖一价。
func SidePx(s Side, buyYes bool) float64 {
	if buyYes {
		return s.YesPx
	}
	return s.NoPx
}

// OutcomeGroup 同一归一化赛果上各平台 market_outcome 在 watch 中的下标。
type OutcomeGroup struct {
	byPlatform map[string]int
}

func NewOutcomeGroup() OutcomeGroup { return OutcomeGroup{byPlatform: map[string]int{}} }

func (g *OutcomeGroup) Add(platform string, outcomeIndex int) {
	if g.byPlatform == nil {
		g.byPlatform = map[string]int{}
	}
	platform = strings.ToUpper(strings.TrimSpace(platform))
	if platform == "" {
		return
	}
	if _, ok := g.byPlatform[platform]; !ok {
		g.byPlatform[platform] = outcomeIndex
	}
}

func (g OutcomeGroup) PlatformIndex(platform string) (int, bool) {
	platform = strings.ToUpper(strings.TrimSpace(platform))
	i, ok := g.byPlatform[platform]
	return i, ok
}

func (g OutcomeGroup) PlatformCount() int { return len(g.byPlatform) }

func (g OutcomeGroup) PlatformList() []string {
	out := make([]string, 0, len(g.byPlatform))
	for p := range g.byPlatform {
		out = append(out, p)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func (g OutcomeGroup) Indices() []int {
	out := make([]int, 0, len(g.byPlatform))
	for _, outcomeIndex := range g.byPlatform {
		out = append(out, outcomeIndex)
	}
	return out
}

// ExecSide 卖一 + 发单标识。
type ExecSide struct {
	Side
	OutcomeID                             uint
	Platform, MarketID, YesToken, NoToken string
}

func ExecSideAt(store *book.Store, outcomeIndex int, oc BookOutcome, fee float64) (ExecSide, bool) {
	s, ok := TopFromStore(store, oc, outcomeIndex, fee)
	if !ok {
		return ExecSide{}, false
	}
	return ExecSide{
		Side: s, Platform: oc.Platform, MarketID: oc.MarketID,
		YesToken: oc.YesToken, NoToken: oc.NoToken,
	}, true
}