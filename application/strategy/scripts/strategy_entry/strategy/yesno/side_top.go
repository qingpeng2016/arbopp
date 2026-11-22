package yesno

// SideTopAskSz TopHit 判定用的卖一价与挂单量（落库 leg_*_best_ask / arb_qty 同源）。
func SideTopAskSz(s Side, buyYes bool) (px, sz float64) {
	if buyYes {
		return s.YesPx, s.YesSz
	}
	return s.NoPx, s.NoSz
}