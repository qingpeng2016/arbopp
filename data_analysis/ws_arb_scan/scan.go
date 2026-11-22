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
	MapKey         string      `json:"map_key"`
	Platforms      []string    `json:"platforms"`
	WSSnapshotRows int         `json:"ws_rows"`
	OutcomeGroups  int         `json:"outcome_groups"`
	YesNoLatest    []yesnoHit  `json:"yesno_latest,omitempty"`
	YesNoHistMin   []yesnoHit  `json:"yesno_hist_min,omitempty"`
	TripleYes      []tripleHit `json:"triple_yes_latest,omitempty"`
	TripleNo       []tripleHit `json:"triple_no_latest,omitempty"`
	Note           string      `json:"note,omitempty"`
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
	if !tr.gotYes || !tr.gotNo {
		tr.q.absorbRow(plat, tokenID, mo, asks, bids)
		tr.gotYes, tr.gotNo = true, true
	}
}

func scanFixture(mapKey string, outcomes []outcomeRec, latestWS, histWS []wsRow) fixtureReport {
	lim := defaultScanLimits()
	fees := defaultFees()
	rep := fixtureReport{MapKey: mapKey, WSSnapshotRows: len(histWS)}

	byHash := map[string]outcomeRec{}
	gkMoneyline := map[string]bool{}
	for _, o := range outcomes {
		hk := strings.ToUpper(o.Platform) + "|" + strings.TrimSpace(o.PlatformMarketOutcomeID)
		byHash[hk] = o
		if gk := outcomeGroupKey(o); gk != "" &&
			strings.EqualFold(strings.TrimSpace(o.OutcomeType), "moneyline") {
			gkMoneyline[gk] = true
		}
	}

	latest := map[string]map[string]*platTrack{}
	histMin := map[string]map[string]*legQuote{}

	processWS := func(row wsRow, isLatest bool) {
		hk := strings.ToUpper(row.Platform) + "|" + strings.TrimSpace(row.PlatformMarketOutcomeID)
		mo, ok := byHash[hk]
		if !ok {
			return
		}
		gk := outcomeGroupKey(mo)
		if gk == "" {
			return
		}
		plat := strings.ToUpper(strings.TrimSpace(row.Platform))
		if latest[gk] == nil {
			latest[gk] = map[string]*platTrack{}
		}
		if histMin[gk] == nil {
			histMin[gk] = map[string]*legQuote{}
		}
		if latest[gk][plat] == nil {
			latest[gk][plat] = &platTrack{}
		}
		if histMin[gk][plat] == nil {
			histMin[gk][plat] = &legQuote{}
		}
		if isLatest {
			absorbLatestPoly(latest[gk][plat], plat, row.TokenID, mo, row.Asks, row.Bids)
		}
		histMin[gk][plat].absorbRow(plat, row.TokenID, mo, row.Asks, row.Bids)
	}

	for _, row := range latestWS {
		processWS(row, true)
	}
	for _, row := range histWS {
		processWS(row, false)
	}

	latestPlain := map[string]map[string]*legQuote{}
	for gk, pl := range latest {
		latestPlain[gk] = map[string]*legQuote{}
		for p, tr := range pl {
			latestPlain[gk][p] = &tr.q
		}
	}

	platSet := map[string]struct{}{}
	for gk := range latestPlain {
		rep.OutcomeGroups++
		for plat := range latestPlain[gk] {
			platSet[plat] = struct{}{}
		}
	}
	for p := range platSet {
		rep.Platforms = append(rep.Platforms, p)
	}
	sort.Strings(rep.Platforms)

	rep.YesNoLatest = scanYesNo(latestPlain, fees, lim, "latest")
	rep.YesNoHistMin = scanYesNo(histMin, fees, lim, "hist_min")
	if len(rep.YesNoHistMin) > 0 {
		rep.Note = "hist_min=各腿独立最低卖价，未时间对齐，仅参考"
	}

	draw, tA, tB, ok := moneylineTriples(latestPlain, gkMoneyline)
	if ok {
		rep.TripleYes = scanTriples(draw, tA, tB, fees, lim, true)
		rep.TripleNo = scanTriples(draw, tA, tB, fees, lim, false)
	}
	if len(latestWS) == 0 && len(histWS) == 0 {
		rep.Note = "无 WS 盘口行"
	}
	return rep
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
	var ynL, ynH, ty, tn int
	var bestYN, bestTriple float64 = math.MaxFloat64, math.MaxFloat64
	for _, r := range reports {
		ynL += len(r.YesNoLatest)
		ynH += len(r.YesNoHistMin)
		ty += len(r.TripleYes)
		tn += len(r.TripleNo)
		for _, h := range r.YesNoLatest {
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
	return fmt.Sprintf("fixtures=%d yesno_latest_hits=%d yesno_hist_hits=%d triple_yes=%d triple_no=%d best_yesno_per=%.4f best_triple_yes=%.4f",
		len(reports), ynL, ynH, ty, tn, bestYN, bestTriple)
}