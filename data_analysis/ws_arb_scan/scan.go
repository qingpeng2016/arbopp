package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type platTrack struct {
	q             legQuote
	gotYes, gotNo bool
}

type yesnoHit struct {
	Mode        string  `json:"mode"`
	GroupKey    string  `json:"group_key"`
	YesPlatform string  `json:"yes_platform"`
	NoPlatform  string  `json:"no_platform"`
	Per         float64 `json:"per"`
	Qty         float64 `json:"qty"`
	SnapshotMs  int64   `json:"snapshot_at_ms,omitempty"`
	// TriggerWSID 仅 replay：本条命中由该 WS 行 id 触发，SnapshotMs 为该行毫秒（对侧多为更早缓存）。
	TriggerWSID int64 `json:"trigger_ws_id,omitempty"`
}

type tripleHit struct {
	Mode      string  `json:"mode"`
	Kind      string  `json:"kind"`
	DrawPlat  string  `json:"draw_plat"`
	TeamAPlat string  `json:"team_a_plat"`
	TeamBPlat string  `json:"team_b_plat"`
	Per       float64 `json:"per"`
	Qty       float64 `json:"qty"`
}

type fixtureReport struct {
	MapKey            string         `json:"map_key"`
	Platforms         []string       `json:"platforms"`
	OutcomeCount      int            `json:"outcome_count"`
	OutcomesByPlatform map[string]int `json:"outcomes_by_platform"`
	WSByPlatform      map[string]int `json:"ws_by_platform"`
	WSSnapshotRows    int            `json:"ws_rows"`
	OutcomeGroups     int            `json:"outcome_groups"`
	YesNoLatest    []yesnoHit  `json:"yesno_latest,omitempty"`
	YesNoHistMin   []yesnoHit  `json:"yesno_hist_min,omitempty"`
	YesNoSnapshot  []yesnoHit  `json:"yesno_snapshot,omitempty"`
	YesNoReplay    []yesnoHit  `json:"yesno_replay,omitempty"`
	TripleYes      []tripleHit `json:"triple_yes_latest,omitempty"`
	TripleNo       []tripleHit `json:"triple_no_latest,omitempty"`
	Note           string      `json:"note,omitempty"`
	BestYesNoPer   float64     `json:"best_yesno_per_latest,omitempty"`
	CrossPlatGroups int        `json:"cross_platform_groups,omitempty"`
}

// setFromRow 用本行 WS 覆盖对应腿（回放用，取 min 会错）。
func (lq *legQuote) setFromRow(plat, tokenID string, mo outcomeRec, asks, bids string) {
	plat = strings.ToUpper(strings.TrimSpace(plat))
	yesTok := strings.TrimSpace(mo.YesTokenID)
	noTok := strings.TrimSpace(mo.NoTokenID)
	tok := strings.TrimSpace(tokenID)
	switch plat {
	case platPoly:
		if yesTok != "" && tok == yesTok {
			p, s := bestAsk(parseLevels(asks))
			if p > 0 {
				lq.YesPx, lq.YesSz = p, s
			}
		}
		if noTok != "" && tok == noTok {
			p, s := bestAsk(parseLevels(asks))
			if p > 0 {
				lq.NoPx, lq.NoSz = p, s
			}
		}
	default:
		ap, asz := bestAsk(parseLevels(asks))
		bid, bsz := bestBid(parseLevels(bids))
		if ap > 0 {
			lq.YesPx, lq.YesSz = ap, asz
		}
		if bid > 0 {
			np := 1 - bid
			if validPrice(np) {
				lq.NoPx, lq.NoSz = np, bsz
			}
		}
	}
}

func (lq *legQuote) absorbRow(plat, tokenID string, mo outcomeRec, asks, bids string) {
	plat = strings.ToUpper(strings.TrimSpace(plat))
	yesTok := strings.TrimSpace(mo.YesTokenID)
	noTok := strings.TrimSpace(mo.NoTokenID)
	tok := strings.TrimSpace(tokenID)

	switch plat {
	case platPoly:
		if yesTok != "" && tok == yesTok {
			p, s := bestAsk(parseLevels(asks))
			if p > 0 && (lq.YesPx <= 0 || p < lq.YesPx) {
				lq.YesPx, lq.YesSz = p, s
			}
		}
		if noTok != "" && tok == noTok {
			p, s := bestAsk(parseLevels(asks))
			if p > 0 && (lq.NoPx <= 0 || p < lq.NoPx) {
				lq.NoPx, lq.NoSz = p, s
			}
		}
	default:
		ap, asz := bestAsk(parseLevels(asks))
		bid, bsz := bestBid(parseLevels(bids))
		if ap > 0 && (lq.YesPx <= 0 || ap < lq.YesPx) {
			lq.YesPx, lq.YesSz = ap, asz
		}
		if bid > 0 {
			np := 1 - bid
			if validPrice(np) && (lq.NoPx <= 0 || np < lq.NoPx) {
				lq.NoPx, lq.NoSz = np, bsz
			}
		}
	}
}

func absorbLatestPoly(tr *platTrack, plat, tokenID string, mo outcomeRec, asks, bids string) {
	yesTok := strings.TrimSpace(mo.YesTokenID)
	noTok := strings.TrimSpace(mo.NoTokenID)
	tok := strings.TrimSpace(tokenID)
	if strings.EqualFold(plat, platPoly) {
		if yesTok != "" && tok == yesTok && !tr.gotYes {
			p, s := bestAsk(parseLevels(asks))
			if p > 0 {
				tr.q.YesPx, tr.q.YesSz = p, s
				tr.gotYes = true
			}
		}
		if noTok != "" && tok == noTok && !tr.gotNo {
			p, s := bestAsk(parseLevels(asks))
			if p > 0 {
				tr.q.NoPx, tr.q.NoSz = p, s
				tr.gotNo = true
			}
		}
		return
	}
	tr.q.absorbRow(plat, tokenID, mo, asks, bids)
	if validPrice(tr.q.YesPx) && tr.q.YesSz > 0 {
		tr.gotYes = true
	}
	if validPrice(tr.q.NoPx) && tr.q.NoSz > 0 {
		tr.gotNo = true
	}
}

func buildLatestPlainFromWS(rows []wsRow, byHash map[string]outcomeRec) map[string]map[string]*legQuote {
	latest := map[string]map[string]*platTrack{}
	for _, row := range rows {
		hk := strings.ToUpper(row.Platform) + "|" + strings.TrimSpace(row.PlatformMarketOutcomeID)
		mo, ok := byHash[hk]
		if !ok || !strings.EqualFold(strings.TrimSpace(mo.OutcomeType), "moneyline") {
			continue
		}
		if !useWSRowForTopBook(row.Platform, row.TokenID, mo) {
			continue
		}
		gk := outcomeGroupKey(mo)
		if gk == "" {
			continue
		}
		plat := strings.ToUpper(strings.TrimSpace(row.Platform))
		if latest[gk] == nil {
			latest[gk] = map[string]*platTrack{}
		}
		if latest[gk][plat] == nil {
			latest[gk][plat] = &platTrack{}
		}
		absorbLatestPoly(latest[gk][plat], plat, row.TokenID, mo, row.Asks, row.Bids)
	}
	plain := map[string]map[string]*legQuote{}
	for gk, pl := range latest {
		plain[gk] = map[string]*legQuote{}
		for p, tr := range pl {
			plain[gk][p] = &tr.q
		}
	}
	return plain
}

func scanYesNoHistMinLegacy(_ []outcomeRec, hist []wsRow, byHash map[string]outcomeRec) []yesnoHit {
	histMin := buildHistMinQuotes(hist, byHash)
	return scanYesNo(histMin, defaultFees(), defaultScanLimits(), "hist_min")
}

func buildHistMinQuotes(hist []wsRow, byHash map[string]outcomeRec) map[string]map[string]*legQuote {
	histMin := map[string]map[string]*legQuote{}
	for _, row := range hist {
		hk := strings.ToUpper(row.Platform) + "|" + strings.TrimSpace(row.PlatformMarketOutcomeID)
		mo, ok := byHash[hk]
		if !ok || !strings.EqualFold(strings.TrimSpace(mo.OutcomeType), "moneyline") {
			continue
		}
		if !useWSRowForTopBook(row.Platform, row.TokenID, mo) {
			continue
		}
		gk := outcomeGroupKey(mo)
		if gk == "" {
			continue
		}
		plat := strings.ToUpper(strings.TrimSpace(row.Platform))
		if histMin[gk] == nil {
			histMin[gk] = map[string]*legQuote{}
		}
		if histMin[gk][plat] == nil {
			histMin[gk][plat] = &legQuote{}
		}
		histMin[gk][plat].absorbRow(plat, row.TokenID, mo, row.Asks, row.Bids)
	}
	return histMin
}

func bestCrossYesNoPer(groups map[string]map[string]*legQuote, fees map[string]platformFee) (best float64, crossGroups int) {
	best = math.MaxFloat64
	for _, plats := range groups {
		if len(plats) < 2 {
			continue
		}
		type side struct {
			plat string
			q    legQuote
		}
		list := make([]side, 0, len(plats))
		for plat, q := range plats {
			if q == nil || !q.ready() {
				continue
			}
			list = append(list, side{plat, *q})
		}
		if len(list) < 2 {
			continue
		}
		crossGroups++
		for i := 0; i < len(list); i++ {
			for j := i + 1; j < len(list); j++ {
				a, b := list[i], list[j]
				for _, pair := range []struct{ y, n side }{{a, b}, {b, a}} {
					per, ok := crossYesNoPer(pair.y.q.YesPx, pair.y.q.YesSz, pair.n.q.NoPx, pair.n.q.NoSz,
						fees[pair.y.plat].sportsCoef, fees[pair.n.plat].sportsCoef)
					if ok && per < best {
						best = per
					}
				}
			}
		}
	}
	if best == math.MaxFloat64 {
		return 0, crossGroups
	}
	return best, crossGroups
}

func scanYesNo(groups map[string]map[string]*legQuote, fees map[string]platformFee, lim scanLimits, mode string) []yesnoHit {
	var out []yesnoHit
	for gk, plats := range groups {
		if len(plats) < 2 {
			continue
		}
		type side struct {
			plat string
			q    legQuote
		}
		list := make([]side, 0, len(plats))
		for plat, q := range plats {
			if q == nil || !q.ready() {
				continue
			}
			list = append(list, side{plat, *q})
		}
		if len(list) < 2 {
			continue
		}
		for i := 0; i < len(list); i++ {
			for j := i + 1; j < len(list); j++ {
				a, b := list[i], list[j]
				try := func(y, n side) {
					fy := fees[y.plat].sportsCoef
					fn := fees[n.plat].sportsCoef
					qty, per, hit := crossYesNoHit(y.q.YesPx, y.q.YesSz, n.q.NoPx, n.q.NoSz, fy, fn, lim)
					if !hit {
						return
					}
					out = append(out, yesnoHit{
						Mode: mode, GroupKey: gk,
						YesPlatform: y.plat, NoPlatform: n.plat,
						Per: round4(per), Qty: round4(qty),
					})
				}
				try(a, b)
				try(b, a)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Per < out[j].Per })
	return out
}

func buildPlainFromWSRows(rows []wsRow, byHash map[string]outcomeRec) map[string]map[string]*legQuote {
	latest := map[string]map[string]*platTrack{}
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
		gk := outcomeGroupKey(mo)
		if gk == "" {
			continue
		}
		plat := strings.ToUpper(strings.TrimSpace(row.Platform))
		if latest[gk] == nil {
			latest[gk] = map[string]*platTrack{}
		}
		if latest[gk][plat] == nil {
			latest[gk][plat] = &platTrack{}
		}
		absorbLatestPoly(latest[gk][plat], plat, row.TokenID, mo, row.Asks, row.Bids)
	}
	plain := map[string]map[string]*legQuote{}
	for gk, pl := range latest {
		plain[gk] = map[string]*legQuote{}
		for p, tr := range pl {
			plain[gk][p] = &tr.q
		}
	}
	return plain
}

// scanYesNoBySnapshot 同一 snapshot_at_ms 内多平台盘口对齐后试算（接近线上 WS 触发）。
func scanYesNoBySnapshot(histWS []wsRow, byHash map[string]outcomeRec, fees map[string]platformFee, lim scanLimits) []yesnoHit {
	byMs := map[int64][]wsRow{}
	for _, r := range histWS {
		if r.SnapshotAtMs <= 0 {
			continue
		}
		byMs[r.SnapshotAtMs] = append(byMs[r.SnapshotAtMs], r)
	}
	type dedupeKey struct {
		gk, yp, np string
	}
	best := map[dedupeKey]yesnoHit{}
	for ms, rows := range byMs {
		plain := buildPlainFromWSRows(rows, byHash)
		for _, h := range scanYesNo(plain, fees, lim, "snapshot") {
			h.SnapshotMs = ms
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

// scanYesNoReplay 按 id 升序回放：每来一行 WS 更新内存盘口，对该 OutcomeGroup 试算（对齐 OnBookUpdate）。
func scanYesNoReplay(histWS []wsRow, byHash map[string]outcomeRec, fees map[string]platformFee, lim scanLimits) []yesnoHit {
	if len(histWS) == 0 {
		return nil
	}
	rows := append([]wsRow(nil), histWS...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	state := map[string]map[string]*legQuote{}
	type dedupeKey struct {
		gk, yp, np string
	}
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
		gk := outcomeGroupKey(mo)
		if gk == "" {
			continue
		}
		plat := strings.ToUpper(strings.TrimSpace(row.Platform))
		if state[gk] == nil {
			state[gk] = map[string]*legQuote{}
		}
		if state[gk][plat] == nil {
			state[gk][plat] = &legQuote{}
		}
		state[gk][plat].setFromRow(plat, row.TokenID, mo, row.Asks, row.Bids)
		plain := map[string]map[string]*legQuote{gk: {}}
		for p, q := range state[gk] {
			plain[gk][p] = q
		}
		for _, h := range scanYesNo(plain, fees, lim, "replay") {
			h.SnapshotMs = row.SnapshotAtMs
			h.TriggerWSID = row.ID
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

func moneylineTriples(groups map[string]map[string]*legQuote, gkMoneyline map[string]bool) (draw, teamA, teamB map[string]*legQuote, ok bool) {
	var teamKeys []string
	draw = map[string]*legQuote{}
	teamA = map[string]*legQuote{}
	teamB = map[string]*legQuote{}
	for gk, plats := range groups {
		if !gkMoneyline[gk] {
			continue
		}
		norm := lastSegmentGroupKey(gk)
		if norm == "draw" {
			draw = plats
			continue
		}
		teamKeys = append(teamKeys, gk)
		if len(teamKeys) == 1 {
			teamA = plats
		}
		if len(teamKeys) == 2 {
			teamB = plats
		}
	}
	if len(draw) < 1 || len(teamKeys) != 2 {
		return nil, nil, nil, false
	}
	return draw, teamA, teamB, true
}

func scanTriples(draw, tA, tB map[string]*legQuote, fees map[string]platformFee, lim scanLimits, buyYes bool) []tripleHit {
	var out []tripleHit
	maxCost := lim.maxCostTripleYes
	kind := "ONE_X2_YES_TRIPLE"
	if !buyYes {
		maxCost = lim.maxCostTripleNo
		kind = "ONE_X2_NO_TRIPLE"
	}
	for dp, dq := range draw {
		for ap, aq := range tA {
			for bp, bq := range tB {
				if dq == nil || aq == nil || bq == nil {
					continue
				}
				per, qty, hit := tripleHitCheck(*dq, *aq, *bq,
					fees[dp].sportsCoef, fees[ap].sportsCoef, fees[bp].sportsCoef,
					buyYes, maxCost, lim)
				if !hit {
					continue
				}
				out = append(out, tripleHit{
					Kind: kind, Mode: "latest",
					DrawPlat: dp, TeamAPlat: ap, TeamBPlat: bp,
					Per: round4(per), Qty: round4(qty),
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Per < out[j].Per })
	return out
}

func summarize(reports []fixtureReport) string {
	var ynL, ynH, ynSnap, ynReplay, ty, tn int
	var bestYN, bestTriple float64 = math.MaxFloat64, math.MaxFloat64
	for _, r := range reports {
		ynL += len(r.YesNoLatest)
		ynH += len(r.YesNoHistMin)
		ynSnap += len(r.YesNoSnapshot)
		ynReplay += len(r.YesNoReplay)
		ty += len(r.TripleYes)
		tn += len(r.TripleNo)
		if r.BestYesNoPer > 0 && r.BestYesNoPer < bestYN {
			bestYN = r.BestYesNoPer
		}
		for _, h := range r.YesNoLatest {
			if h.Per < bestYN {
				bestYN = h.Per
			}
		}
		for _, h := range r.YesNoSnapshot {
			if h.Per < bestYN {
				bestYN = h.Per
			}
		}
		for _, h := range r.YesNoReplay {
			if h.Per < bestYN {
				bestYN = h.Per
			}
		}
		for _, h := range r.TripleYes {
			if h.Per < bestTriple {
				bestTriple = h.Per
			}
		}
	}
	if bestYN == math.MaxFloat64 {
		bestYN = 0
	}
	if bestTriple == math.MaxFloat64 {
		bestTriple = 0
	}
	return fmt.Sprintf("fixtures=%d yesno_latest_hits=%d yesno_replay_hits=%d yesno_snapshot_hits=%d yesno_hist_hits=%d triple_yes=%d triple_no=%d best_yesno_per=%.4f best_triple_yes=%.4f",
		len(reports), ynL, ynReplay, ynSnap, ynH, ty, tn, bestYN, bestTriple)
}