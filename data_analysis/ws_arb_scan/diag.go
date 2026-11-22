package main

import (
	"fmt"
	"sort"
	"strings"
)

func printDiag(mapKey string, outcomes []outcomeRec, latestWS []wsRow) {
	byHash := outcomeIndex(outcomes)
	latest := map[string]map[string]*platTrack{}
	for _, row := range latestWS {
		hk := strings.ToUpper(row.Platform) + "|" + strings.TrimSpace(row.PlatformMarketOutcomeID)
		mo, ok := byHash[hk]
		if !ok || !useWSRowForTopBook(row.Platform, row.TokenID, mo) {
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
	type ginfo struct {
		gk    string
		plats []string
	}
	var ml []ginfo
	for _, o := range outcomes {
		if !strings.EqualFold(o.OutcomeType, "moneyline") {
			continue
		}
		gk := outcomeGroupKey(o)
		if gk == "" {
			fmt.Printf("diag: no group key platform=%s comp=%q slug=%q name=%.80s\n",
				o.Platform, o.Competition, o.Slug, o.OutcomeName)
			continue
		}
		found := false
		for i := range ml {
			if ml[i].gk == gk {
				found = true
				break
			}
		}
		if !found {
			pl := []string{}
			if latest[gk] != nil {
				for p, tr := range latest[gk] {
					q := tr.q
					rd := q.ready()
					pl = append(pl, fmt.Sprintf("%s(yes=%.3f/%.0f no=%.3f/%.0f ready=%v)",
						p, q.YesPx, q.YesSz, q.NoPx, q.NoSz, rd))
				}
			}
			sort.Strings(pl)
			ml = append(ml, ginfo{gk, pl})
		}
	}
	fmt.Printf("--- diag %s moneyline groups=%d ---\n", mapKey, len(ml))
	for _, g := range ml {
		fmt.Println(g.gk)
		for _, p := range g.plats {
			fmt.Println(" ", p)
		}
	}
}