package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

type row struct {
	MapKey         string  `gorm:"column:map_key"`
	EventStart     *string `gorm:"column:event_start_time"`
	Platform       string  `gorm:"column:platform"`
	MarketURL      string  `gorm:"column:market_url"`
	Name           *string `gorm:"column:name"`
	Slug           *string `gorm:"column:slug"`
	PlatformMarket string  `gorm:"column:platform_market_id"`
	MapVerified    int     `gorm:"column:map_verified"`
}

func parseFixtureDay(mapKey string, ev *string) (time.Time, bool) {
	if ev != nil {
		s := strings.TrimSpace(*ev)
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
			if t, e := time.ParseInLocation(layout, s, time.UTC); e == nil {
				return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), true
			}
		}
	}
	mapKey = strings.TrimSpace(mapKey)
	if len(mapKey) >= 10 {
		suffix := mapKey[len(mapKey)-10:]
		if suffix[4] == '-' && suffix[7] == '-' {
			if t, e := time.ParseInLocation("2006-01-02", suffix, time.UTC); e == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

func main() {
	_ = conf.NewCfg()
	db := boot.NewDBClient()
	ageCutoff := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -21)
	fmt.Println("free PMData: fixture UTC day must be on or before", ageCutoff.Format("2006-01-02"))

	const q = `
SELECT TRIM(map_key) AS map_key,
       UPPER(TRIM(platform)) AS platform,
       MAX(map_verified) AS map_verified
FROM market
WHERE status = 'ACTIVE'
  AND map_key IS NOT NULL AND TRIM(map_key) <> ''
  AND (sub_category = 'Football' OR sport_cn = '足球' OR category = 'Sports')
  AND UPPER(TRIM(platform)) IN ('POLYMARKET', 'PREDICT')
GROUP BY TRIM(map_key), UPPER(TRIM(platform))`

	var parts []struct {
		MapKey      string `gorm:"column:map_key"`
		Platform    string `gorm:"column:platform"`
		MapVerified int    `gorm:"column:map_verified"`
	}
	if err := db.Raw(q).Scan(&parts).Error; err != nil {
		fmt.Println("ERR", err)
		return
	}

	type pair struct {
		verified bool
	}
	byKey := map[string]*pair{}
	for _, p := range parts {
		if byKey[p.MapKey] == nil {
			byKey[p.MapKey] = &pair{}
		}
		if p.MapVerified == 1 {
			byKey[p.MapKey].verified = true
		}
	}

	var candidates []string
	for k := range byKey {
		hasP, hasPoly := false, false
		for _, p := range parts {
			if p.MapKey != k {
				continue
			}
			if p.Platform == "PREDICT" {
				hasP = true
			}
			if p.Platform == "POLYMARKET" {
				hasPoly = true
			}
		}
		if !hasP || !hasPoly {
			continue
		}
		var sample row
		_ = db.Raw(`SELECT map_key, event_start_time FROM market WHERE TRIM(map_key)=? LIMIT 1`, k).Scan(&sample)
		day, ok := parseFixtureDay(k, sample.EventStart)
		if !ok || !day.Before(ageCutoff) {
			continue
		}
		candidates = append(candidates, k)
	}

	fmt.Printf("Poly+Predict football with fixture day <= cutoff: %d\n", len(candidates))

	// 没有配对时：分别列最老 Poly / Predict 足球日
	if len(candidates) == 0 {
		fmt.Println("\n-- 库内没有任何「满 3 周」的 Poly+Predict 同 map_key 足球；最老单项：")
		for _, plat := range []string{"POLYMARKET", "PREDICT"} {
			var olds []row
			_ = db.Raw(`
SELECT TRIM(map_key) map_key, event_start_time, market_url, name, slug
FROM market
WHERE status='ACTIVE' AND UPPER(TRIM(platform))=?
  AND (sub_category='Football' OR sport_cn='足球')
  AND map_key IS NOT NULL AND TRIM(map_key)<>''
ORDER BY event_start_time ASC, id ASC LIMIT 5`, plat).Scan(&olds)
			fmt.Printf("\n%s oldest:\n", plat)
			for _, m := range olds {
				d, _ := parseFixtureDay(m.MapKey, m.EventStart)
				fmt.Printf("  day=%s map_key=%s url=%s\n", d.Format("2006-01-02"), m.MapKey, m.MarketURL)
			}
		}
		fmt.Println("\n要到免费 PMData：需先 sync/map 出 10 月及更早的 Poly+Predict 同场，或用 PMData catalog 里 slug+market_id 手工落库后再插 data_task。")
		return
	}

	picked := candidates[0]
	// 取最早一场
	var pickDay time.Time
	for _, k := range candidates {
		var sample row
		_ = db.Raw(`SELECT map_key, event_start_time FROM market WHERE TRIM(map_key)=? LIMIT 1`, k).Scan(&sample)
		day, _ := parseFixtureDay(k, sample.EventStart)
		if picked == candidates[0] || day.Before(pickDay) {
			picked, pickDay = k, day
		}
	}

	var markets []row
	_ = db.Raw(`
SELECT TRIM(map_key) map_key, event_start_time, UPPER(TRIM(platform)) platform,
       market_url, name, slug, platform_market_id, map_verified
FROM market
WHERE TRIM(map_key) = ? AND status = 'ACTIVE'
  AND UPPER(TRIM(platform)) IN ('POLYMARKET','PREDICT')
ORDER BY platform`, picked).Scan(&markets)

	urls := map[string]string{}
	var ev *string
	for _, m := range markets {
		urls[m.Platform] = strings.TrimSpace(m.MarketURL)
		if ev == nil {
			ev = m.EventStart
		}
	}
	dMatch, _ := parseFixtureDay(picked, ev)
	dates := []string{
		dMatch.AddDate(0, 0, -1).Format("2006-01-02"),
		dMatch.Format("2006-01-02"),
		dMatch.AddDate(0, 0, 1).Format("2006-01-02"),
	}
	p := map[string]interface{}{
		"min_utc_age_days": 21,
		"utc_days_before":  1,
		"utc_days_after":   1,
		"utc_dates":        dates,
		"outcome_types":    []string{"moneyline"},
		"market_url":       urls,
		"note":             fmt.Sprintf("fixture_utc=%s; free tier OK", dMatch.Format("2006-01-02")),
	}
	pj, _ := json.Marshal(p)

	fmt.Println("\nmap_key:", picked, "fixture:", dMatch.Format("2006-01-02"))
	fmt.Println("INSERT INTO data_task (ingest_source, map_key, params, status) VALUES")
	fmt.Printf("('pmdata', %q, %q, 'pending');\n", picked, string(pj))
	for _, m := range markets {
		n := ""
		if m.Name != nil {
			n = *m.Name
		}
		fmt.Printf("-- %s verified=%d %s | %s\n", m.Platform, m.MapVerified, m.MarketURL, n)
	}
}