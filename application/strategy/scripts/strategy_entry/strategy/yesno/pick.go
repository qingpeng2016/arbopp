package yesno

// Pick 跨平台两两套利候选。
type Pick struct {
	YesSide, NoSide           Side
	YesBuyYes, NoBuyYes       bool
	Qty, Per                  float64
}

// FilledPick 带平台信息的成交候选。
type FilledPick struct {
	Yes, No                   ExecSide
	YesBuyYes, NoBuyYes       bool
	Qty, Per                  float64
}

func considerPick(best *Pick, found *bool, pick Pick, hit bool) {
	if !hit {
		return
	}
	if !*found || pick.Per < best.Per {
		*best = pick
		*found = true
	}
}

// bestFilledPick 同一 OutcomeGroup 上最优跨所 YES+NO 组合（Runner.grabOne 调用）。
func (r *Runner) bestFilledPick(group OutcomeGroup) (FilledPick, bool) {
	lim := r.limits()
	p, ok := r.bestOutcomeGroupPick(group, lim)
	if !ok {
		return FilledPick{}, false
	}
	return r.filledPickFrom(p)
}

func (r *Runner) bestOutcomeGroupPick(group OutcomeGroup, lim Limits) (Pick, bool) {
	if group.PlatformCount() < 2 {
		return Pick{}, false
	}
	plats := group.PlatformList()
	var best Pick
	found := false
	for i := 0; i < len(plats); i++ {
		for j := i + 1; j < len(plats); j++ {
			ia, okA := group.PlatformIndex(plats[i])
			ib, okB := group.PlatformIndex(plats[j])
			if !okA || !okB {
				continue
			}
			sa, oka := r.sideTop(ia)
			sb, okb := r.sideTop(ib)
			if !oka || !okb {
				continue
			}
			// 每对平台：i YES+j NO 与 j YES+i NO；不试 i NO+j YES（与 i YES+j NO 镜像重复）。
			qtyA, perA, hitA := TopHit(sa.YesPx, sa.YesSz, sb.NoPx, sb.NoSz, sa.Fee, sb.Fee, lim.OrderAmount, lim.LevelFactor, lim.MaxCost)
			considerPick(&best, &found, Pick{YesSide: sa, NoSide: sb, YesBuyYes: true, NoBuyYes: false, Qty: qtyA, Per: perA}, hitA)
			qtyB, perB, hitB := TopHit(sb.YesPx, sb.YesSz, sa.NoPx, sa.NoSz, sb.Fee, sa.Fee, lim.OrderAmount, lim.LevelFactor, lim.MaxCost)
			considerPick(&best, &found, Pick{YesSide: sb, NoSide: sa, YesBuyYes: true, NoBuyYes: false, Qty: qtyB, Per: perB}, hitB)
		}
	}
	return best, found
}

func (r *Runner) filledPickFrom(p Pick) (FilledPick, bool) {
	yes, okA := r.execSide(p.YesSide.Index)
	no, okB := r.execSide(p.NoSide.Index)
	if !okA || !okB {
		return FilledPick{}, false
	}
	return FilledPick{
		Yes: yes, No: no,
		YesBuyYes: p.YesBuyYes, NoBuyYes: p.NoBuyYes,
		Qty: p.Qty, Per: p.Per,
	}, true
}