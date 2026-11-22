package main

import "strings"

func outcomeIndex(outcomes []outcomeRec) map[string]outcomeRec {
	by := make(map[string]outcomeRec, len(outcomes))
	for _, o := range outcomes {
		hk := strings.ToUpper(o.Platform) + "|" + strings.TrimSpace(o.PlatformMarketOutcomeID)
		by[hk] = o
	}
	return by
}