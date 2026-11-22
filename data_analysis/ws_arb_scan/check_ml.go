package main

import (
	"fmt"
	"strings"
)

// checkMoneylineAtMs 只核对该毫秒、独赢 outcome 的 WS，打印两边顶价与成本。
func checkMoneylineAtMs(mapKey string, ms int64) error {
	outcomes, err := loadOutcomesDB(mapKey)
	if err != nil {
		return err
	}
	var ml []outcomeRec
	for _, o := range outcomes {
		if strings.EqualFold(strings.TrimSpace(o.OutcomeType), "moneyline") {
			ml = append(ml, o)
		}
	}
	if len(ml) == 0 {
		return fmt.Errorf("无 moneyline outcome")
	}
	ids := make([]string, 0, len(ml))
	want := map[string]outcomeRec{}
	for _, o := range ml {
		id := strings.TrimSpace(o.PlatformMarketOutcomeID)
		ids = append(ids, id)
		want[strings.ToUpper(o.Platform)+"|"+id] = o
	}
	rows, err := loadWSDBAtMs(mapKey, ms, ids)
	if err != nil {
		return err
	}
	fmt.Printf("map_key=%s 毫秒=%d 独赢WS行数=%d\n", mapKey, ms, len(rows))
	byPlat := map[string]map[string]*legQuote{}
	for _, row := range rows {
		hk := strings.ToUpper(row.Platform) + "|" + strings.TrimSpace(row.PlatformMarketOutcomeID)
		mo, ok := want[hk]
		if !ok || !useWSRowForTopBook(row.Platform, row.TokenID, mo) {
			continue
		}
		plat := strings.ToUpper(strings.TrimSpace(row.Platform))
		gk := outcomeGroupKey(mo)
		if byPlat[gk] == nil {
			byPlat[gk] = map[string]*legQuote{}
		}
		if byPlat[gk][plat] == nil {
			byPlat[gk][plat] = &legQuote{}
		}
		byPlat[gk][plat].setFromRow(plat, row.TokenID, mo, row.Asks, row.Bids)
	}
	fees := defaultFees()
	lim := defaultScanLimits()
	for gk, pl := range byPlat {
		fmt.Printf("赛果组: %s\n", gk)
		for p, q := range pl {
			if q == nil {
				continue
			}
			fmt.Printf("  %s yes=%.4f/%.0f no=%.4f/%.0f ready=%v\n",
				platformLabel(p), q.YesPx, q.YesSz, q.NoPx, q.NoSz, q.ready())
		}
		hits := scanYesNo(map[string]map[string]*legQuote{gk: pl}, fees, lim, "check")
		if len(hits) == 0 {
			fmt.Println("  结论: 无套利（独赢口径）")
		} else {
			for _, h := range hits {
				fmt.Printf("  结论: 有套利 YES@%s NO@%s 成本%.4f\n",
					platformLabel(h.YesPlatform), platformLabel(h.NoPlatform), h.Per)
			}
		}
	}
	return nil
}