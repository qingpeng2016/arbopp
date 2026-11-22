package main

import (
	"context"
	"math"
	"sort"
	"strings"

	strategyentry "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
)

func loadScanParams() strategyentry.Params {
	return strategyentry.LoadParams(context.Background(), nil, nil)
}

func scanFixture(mapKey string, outcomes []outcomeRec, latestWS, histWS []wsRow) fixtureReport {
	params := loadScanParams()
	byHash := outcomeIndex(outcomes)

	rep := fixtureReport{MapKey: mapKey, WSSnapshotRows: len(histWS)}

	// —— 最新盘口（strategy_entry yesno）——
	wLatest, _ := buildScanWatch(mapKey, outcomes, params)
	for _, row := range latestWS {
		if mo, ok := byHash[strings.ToUpper(row.Platform)+"|"+strings.TrimSpace(row.PlatformMarketOutcomeID)]; ok {
			applyBookRow(wLatest.Books, row, mo)
		}
	}
	runLatest := yesno.Entry(wLatest)
	rep.Platforms = platformListFromWatch(wLatest)
	rep.OutcomeGroups = len(runLatest.OutcomeGroupsMap())
	rep.YesNoLatest = collectYesNoHits(runLatest, wLatest, "latest", 0, 0)
	rep.CrossPlatGroups, rep.BestYesNoPer = crossPlatStats(runLatest, wLatest)
	rep.YesNoReplay = scanYesNoReplayStrategy(mapKey, outcomes, histWS, byHash, params)
	rep.YesNoSnapshot = scanYesNoSnapshotStrategy(mapKey, outcomes, histWS, byHash, params)
	rep.YesNoHistMin = scanYesNoHistMinLegacy(outcomes, histWS, byHash)

	gkMoneyline := map[string]bool{}
	for _, o := range outcomes {
		if gk := outcomeGroupKey(o); gk != "" && strings.EqualFold(o.OutcomeType, "moneyline") {
			gkMoneyline[gk] = true
		}
	}
	latestPlain := buildLatestPlainFromWS(latestWS, byHash)
	draw, tA, tB, ok := moneylineTriples(latestPlain, gkMoneyline)
	if ok {
		lim := defaultScanLimits()
		fees := defaultFees()
		rep.TripleYes = scanTriples(draw, tA, tB, fees, lim, true)
		rep.TripleNo = scanTriples(draw, tA, tB, fees, lim, false)
	}

	if len(rep.YesNoReplay) > 0 {
		if orderBookFromWS() {
			rep.Note = "yesno_replay=按 WS id 回放盘口（等同线上单腿更新触发）"
		} else {
			rep.Note = "yesno_replay=按 REST 落库 id 回放（orderbook_sync 每次插入触发试算）"
		}
	} else if len(rep.YesNoHistMin) > 0 {
		rep.Note = "hist_min=各腿独立最低价（未同毫秒），勿当真"
	}
	if len(latestWS) == 0 && len(histWS) == 0 {
		rep.Note = "无订单簿行（表 " + orderBookTable + "）"
	}
	return rep
}

func platformListFromWatch(w *watchgroup.Watch) []string {
	if w == nil {
		return nil
	}
	seen := map[string]struct{}{}
	for i := range w.Outcomes {
		p := strings.ToUpper(strings.TrimSpace(w.Outcomes[i].Platform))
		if p != "" {
			seen[p] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func collectYesNoHits(runner *yesno.Runner, w *watchgroup.Watch, mode string, snapMs, triggerID int64) []yesnoHit {
	var out []yesnoHit
	for gk, g := range runner.OutcomeGroupsMap() {
		if !groupIsMoneyline(w, g) {
			continue
		}
		for _, p := range runner.AllPicksInGroup(g) {
			out = append(out, hitFromPick(mode, gk, w, p, snapMs, triggerID))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Per < out[j].Per })
	return out
}

func hitFromPick(mode, gk string, w *watchgroup.Watch, p yesno.Pick, snapMs, triggerID int64) yesnoHit {
	yesPlat, noPlat := "", ""
	if oc := w.OutcomeAt(p.YesSide.Index); oc != nil {
		yesPlat = oc.Platform
	}
	if oc := w.OutcomeAt(p.NoSide.Index); oc != nil {
		noPlat = oc.Platform
	}
	return yesnoHit{
		Mode: mode, GroupKey: gk,
		YesPlatform: yesPlat, NoPlatform: noPlat,
		Per: round4(p.Per), Qty: round4(p.Qty),
		SnapshotMs: snapMs, TriggerWSID: triggerID,
	}
}

func groupIsMoneyline(w *watchgroup.Watch, g yesno.OutcomeGroup) bool {
	for _, idx := range g.Indices() {
		oc := w.OutcomeAt(idx)
		if oc != nil && strings.EqualFold(strings.TrimSpace(oc.OutcomeType), "moneyline") {
			return true
		}
	}
	return false
}

func crossPlatStats(runner *yesno.Runner, w *watchgroup.Watch) (cross int, best float64) {
	best = math.MaxFloat64
	loose := runner.LimitsForScan()
	loose.MaxCost = 2
	for gk, g := range runner.OutcomeGroupsMap() {
		if !groupIsMoneyline(w, g) {
			continue
		}
		if g.PlatformCount() < 2 {
			continue
		}
		ready := 0
		for _, plat := range g.PlatformList() {
			idx, ok := g.PlatformIndex(plat)
			if !ok {
				continue
			}
			if _, ok := runner.SideTopAt(idx); ok {
				ready++
			}
		}
		if ready >= 2 {
			cross++
		}
		for _, p := range runner.AllPicksInGroup(g) {
			if p.Per < best {
				best = p.Per
			}
		}
		// 未过深度时仍展示「最优合成成本」
		if _, ok := runner.BestPickInGroup(g); !ok {
			plats := g.PlatformList()
			for i := 0; i < len(plats); i++ {
				for j := i + 1; j < len(plats); j++ {
					ia, _ := g.PlatformIndex(plats[i])
					ib, _ := g.PlatformIndex(plats[j])
					sa, oka := runner.SideTopAt(ia)
					sb, okb := runner.SideTopAt(ib)
					if !oka || !okb {
						continue
					}
					_, perA, _ := yesno.TopHit(sa.YesPx, sa.YesSz, sb.NoPx, sb.NoSz, sa.Fee, sb.Fee,
						loose.OrderAmount, loose.LevelFactor, loose.MaxCost)
					_, perB, _ := yesno.TopHit(sb.YesPx, sb.YesSz, sa.NoPx, sa.NoSz, sb.Fee, sa.Fee,
						loose.OrderAmount, loose.LevelFactor, loose.MaxCost)
					for _, per := range []float64{perA, perB} {
						if per > 0 && per < best {
							best = per
						}
					}
				}
			}
			_ = gk
		}
	}
	if best == math.MaxFloat64 {
		return cross, 0
	}
	return cross, round4(best)
}

func scanYesNoReplayStrategy(mapKey string, outcomes []outcomeRec, hist []wsRow, byHash map[string]outcomeRec, params strategyentry.Params) []yesnoHit {
	if len(hist) == 0 {
		return nil
	}
	rows := append([]wsRow(nil), hist...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	w, _ := buildScanWatch(mapKey, outcomes, params)
	runner := yesno.Entry(w)
	type dedupeKey struct{ gk, yp, np string }
	best := map[dedupeKey]yesnoHit{}
	for _, row := range rows {
		hk := strings.ToUpper(row.Platform) + "|" + strings.TrimSpace(row.PlatformMarketOutcomeID)
		mo, ok := byHash[hk]
		if !ok {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(mo.OutcomeType), "moneyline") {
			continue
		}
		if !useWSRowForTopBook(row.Platform, row.TokenID, mo) {
			continue
		}
		applyBookRow(w.Books, row, mo)
		ocID := mo.ID
		if ocID == 0 {
			continue
		}
		fp, ok := runner.TryPickForOutcome(ocID)
		if !ok {
			continue
		}
		idx, okIdx := w.OutcomeIndex(ocID)
		if !okIdx {
			continue
		}
		gk := w.OutcomeGroupKeyAt(idx)
		h := filledPickToHit("replay", gk, fp, row.SnapshotAtMs, row.ID)
		k := dedupeKey{h.GroupKey, h.YesPlatform, h.NoPlatform}
		if prev, ok := best[k]; !ok || h.Per < prev.Per {
			best[k] = h
		}
	}
	out := make([]yesnoHit, 0, len(best))
	for _, h := range best {
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Per < out[j].Per })
	return out
}

func scanYesNoSnapshotStrategy(mapKey string, outcomes []outcomeRec, hist []wsRow, byHash map[string]outcomeRec, params strategyentry.Params) []yesnoHit {
	byMs := map[int64][]wsRow{}
	for _, r := range hist {
		byMs[r.SnapshotAtMs] = append(byMs[r.SnapshotAtMs], r)
	}
	type dedupeKey struct{ gk, yp, np string }
	best := map[dedupeKey]yesnoHit{}
	for ms, rows := range byMs {
		w, _ := buildScanWatch(mapKey, outcomes, params)
		runner := yesno.Entry(w)
		for _, row := range rows {
			if mo, ok := byHash[strings.ToUpper(row.Platform)+"|"+strings.TrimSpace(row.PlatformMarketOutcomeID)]; ok {
				if !strings.EqualFold(strings.TrimSpace(mo.OutcomeType), "moneyline") {
					continue
				}
				applyBookRow(w.Books, row, mo)
			}
		}
		for _, h := range collectYesNoHits(runner, w, "snapshot", ms, 0) {
			k := dedupeKey{h.GroupKey, h.YesPlatform, h.NoPlatform}
			if prev, ok := best[k]; !ok || h.Per < prev.Per {
				best[k] = h
			}
		}
	}
	out := make([]yesnoHit, 0, len(best))
	for _, h := range best {
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Per < out[j].Per })
	return out
}