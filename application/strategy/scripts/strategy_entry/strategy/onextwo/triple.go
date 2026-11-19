package onextwo

import (
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
)

// TriplePick 三路 YES 或三路 NO 组合。
type TriplePick struct {
	Kind     string // OppKindYesTriple | OppKindNoTriple
	BuyYes   bool
	Legs     [3]yesno.ExecSide
	Prices   [3]float64
	Qty, Per float64
}

func outcomeNormFromGroupKey(groupKey string) string {
	parts := strings.Split(groupKey, "|")
	if len(parts) < 4 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func (r *Runner) moneylineTripleGroups() (draw, teamA, teamB yesno.OutcomeGroup, ok bool) {
	var teamKeys []string
	prefix := r.fixtureGroupPrefix()
	for gk, g := range r.outcomeGroups {
		if prefix != "" && !strings.HasPrefix(gk, prefix) {
			continue
		}
		if !r.groupIsMoneyline(g) {
			continue
		}
		norm := outcomeNormFromGroupKey(gk)
		if norm == "" {
			continue
		}
		if norm == "draw" {
			draw = g
			continue
		}
		teamKeys = append(teamKeys, gk)
		if len(teamKeys) == 1 {
			teamA = g
		}
		if len(teamKeys) == 2 {
			teamB = g
		}
	}
	if draw.PlatformCount() < 1 || len(teamKeys) != 2 {
		return yesno.OutcomeGroup{}, yesno.OutcomeGroup{}, yesno.OutcomeGroup{}, false
	}
	if draw.PlatformCount() < 1 || teamA.PlatformCount() < 1 || teamB.PlatformCount() < 1 {
		return yesno.OutcomeGroup{}, yesno.OutcomeGroup{}, yesno.OutcomeGroup{}, false
	}
	return draw, teamA, teamB, true
}

func (r *Runner) fixtureGroupPrefix() string {
	mapKey := strings.TrimSpace(r.w.MapKey)
	if mapKey == "" {
		return ""
	}
	return mapKey + "|"
}

func (r *Runner) groupIsMoneyline(g yesno.OutcomeGroup) bool {
	idxs := g.Indices()
	if len(idxs) == 0 {
		return false
	}
	for _, idx := range idxs {
		oc := r.w.OutcomeAt(idx)
		if oc == nil {
			return false
		}
		if !strings.EqualFold(strings.TrimSpace(oc.OutcomeType), "moneyline") {
			return false
		}
	}
	return true
}