package onextwo

import (
	"sort"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
)

// TriplePick 三路 YES 或三路 NO 组合。
type TriplePick struct {
	Kind     string // OppKindYesTriple | OppKindNoTriple
	BuyYes   bool
	Legs     [3]yesno.ExecSide
	Prices   [3]float64 // 卖一裸价（不含费，与 leg_*_best_ask 一致）
	Costs    [3]float64 // 单位净兑付含费成本
	At       time.Time
	LegSizes    [3]float64 // 各腿卖一原始挂单量（arb_qty 取 min）
	Qty, Per    float64
	SizeCheck   yesno.OrderSizeCheckResult
	SizeCheckOK bool
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

	}
	if draw.PlatformCount() < 1 || len(teamKeys) != 2 {
		return yesno.OutcomeGroup{}, yesno.OutcomeGroup{}, yesno.OutcomeGroup{}, false
	}
	sort.Strings(teamKeys)
	return draw, r.outcomeGroups[teamKeys[0]], r.outcomeGroups[teamKeys[1]], true
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
