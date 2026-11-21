package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	msynccommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/mysql"
)

const (
	maxUnlinkedSamples   = 40
	maxWrongLinkSamples  = 40
	maxStartDeltaHours   = 18
)

func main() {
	_ = conf.NewCfg()
	db := boot.NewDBClient()
	repo := mysql.NewMarketImpl(db)
	ctx := context.Background()

	_, rows, err := repo.List(ctx, nil, map[string]interface{}{
		"status = ?": string(cryptoPredictionEnum.MarketStatusActive),
	}, "competition_en asc, platform asc, id asc", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "list market: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("ACTIVE markets: %d\n\n", len(rows))

	unlinked := findUnlinkedCrossPlatform(rows)
	wrong := findSuspectWrongMapKey(rows)

	fmt.Printf("=== 可能应关联但未关联（同桶+同开赛日+队名能对上，map_key 不一致或缺）: %d 对 ===\n", len(unlinked))
	limit := maxUnlinkedSamples
	if len(unlinked) < limit {
		limit = len(unlinked)
	}
	for i := 0; i < limit; i++ {
		u := unlinked[i]
		fmt.Printf("[%d] %s | %s vs %s\n", i+1, u.Scope, u.PlatformA, u.PlatformB)
		fmt.Printf("    A id=%d map_key=%q verified=%d start=%s\n       name=%s\n", u.IDA, nullStr(u.MapKeyA), u.VerifiedA, u.StartA, trunc(u.NameA, 120))
		fmt.Printf("    B id=%d map_key=%q verified=%d start=%s\n       name=%s\n", u.IDB, nullStr(u.MapKeyB), u.VerifiedB, u.StartB, trunc(u.NameB, 120))
	}
	if len(unlinked) > maxUnlinkedSamples {
		fmt.Printf("... and %d more\n", len(unlinked)-maxUnlinkedSamples)
	}

	fmt.Printf("\n=== 同一 map_key 但疑似错链（队名对不上或开赛差>%dh）: %d 对 ===\n", maxStartDeltaHours, len(wrong))
	limit = maxWrongLinkSamples
	if len(wrong) < limit {
		limit = len(wrong)
	}
	for i := 0; i < limit; i++ {
		w := wrong[i]
		fmt.Printf("[%d] map_key=%q reason=%s\n", i+1, w.MapKey, w.Reason)
		fmt.Printf("    %s id=%d start=%s name=%s\n", w.PlatformA, w.IDA, w.StartA, trunc(w.NameA, 100))
		fmt.Printf("    %s id=%d start=%s name=%s\n", w.PlatformB, w.IDB, w.StartB, trunc(w.NameB, 100))
	}
	if len(wrong) > maxWrongLinkSamples {
		fmt.Printf("... and %d more\n", len(wrong)-maxWrongLinkSamples)
	}

	fmt.Printf("\n=== 汇总 ===\n")
	fmt.Printf("无 map_key 的 ACTIVE 场次: %d\n", countNoMapKey(rows))
	fmt.Printf("map_verified=0 但有 map_key: %d\n", countHasKeyUnverified(rows))
}

type unlinkedPair struct {
	Scope     string
	PlatformA string
	PlatformB string
	IDA, IDB  uint
	MapKeyA   *string
	MapKeyB   *string
	VerifiedA int
	VerifiedB int
	NameA     string
	NameB     string
	StartA    string
	StartB    string
}

type wrongPair struct {
	MapKey    string
	Reason    string
	PlatformA string
	PlatformB string
	IDA, IDB  uint
	NameA     string
	NameB     string
	StartA    string
	StartB    string
}

func marketScope(m *persistententity.Market) string {
	if m == nil {
		return ""
	}
	cat, sub := "", ""
	if m.Category != nil {
		cat = strings.TrimSpace(*m.Category)
	}
	if m.SubCategory != nil {
		sub = strings.TrimSpace(*m.SubCategory)
	}
	en := ""
	if m.CompetitionEn != nil {
		en = strings.TrimSpace(*m.CompetitionEn)
	}
	return msynccommon.MapMatchScopeKey(cat, sub, en)
}

func mapKeyOf(m *persistententity.Market) string {
	if m == nil || m.MapKey == nil {
		return ""
	}
	return strings.TrimSpace(*m.MapKey)
}

func nameOf(m *persistententity.Market) string {
	if m == nil || m.Name == nil {
		return ""
	}
	return strings.TrimSpace(*m.Name)
}

func startOf(m *persistententity.Market) string {
	if m == nil || m.EventStartTime == nil {
		return ""
	}
	return strings.TrimSpace(*m.EventStartTime)
}

func eventDay(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 10 {
		return s[:10]
	}
	return ""
}

func startDeltaHours(a, b string) (int, bool) {
	ta, oka := parseEventStart(a)
	tb, okb := parseEventStart(b)
	if !oka || !okb {
		return 0, false
	}
	d := ta.Sub(tb)
	if d < 0 {
		d = -d
	}
	return int(d.Hours()), true
}

func parseEventStart(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func sameMapKeyLinked(a, b *persistententity.Market) bool {
	ka, kb := mapKeyOf(a), mapKeyOf(b)
	return ka != "" && ka == kb
}

func findUnlinkedCrossPlatform(rows []persistententity.Market) []unlinkedPair {
	byScope := map[string][]*persistententity.Market{}
	for i := range rows {
		m := &rows[i]
		scope := marketScope(m)
		if scope == "" {
			continue
		}
		byScope[scope] = append(byScope[scope], m)
	}
	var out []unlinkedPair
	scopes := make([]string, 0, len(byScope))
	for s := range byScope {
		scopes = append(scopes, s)
	}
	sort.Strings(scopes)
	for _, scope := range scopes {
		list := byScope[scope]
		for i := 0; i < len(list); i++ {
			for j := i + 1; j < len(list); j++ {
				a, b := list[i], list[j]
				if strings.EqualFold(a.Platform, b.Platform) {
					continue
				}
				if sameMapKeyLinked(a, b) {
					continue
				}
				na, nb := nameOf(a), nameOf(b)
				if na == "" || nb == "" {
					continue
				}
				if !fixtureNamesMatchExport(na, nb) {
					continue
				}
				dayA, dayB := eventDay(startOf(a)), eventDay(startOf(b))
				if dayA == "" || dayB == "" || dayA != dayB {
					continue
				}
				if dh, ok := startDeltaHours(startOf(a), startOf(b)); ok && dh > maxStartDeltaHours {
					continue
				}
				out = append(out, unlinkedPair{
					Scope:     scope,
					PlatformA: strings.ToUpper(strings.TrimSpace(a.Platform)),
					PlatformB: strings.ToUpper(strings.TrimSpace(b.Platform)),
					IDA:       a.ID, IDB: b.ID,
					MapKeyA:   a.MapKey, MapKeyB:   b.MapKey,
					VerifiedA: a.MapVerified, VerifiedB: b.MapVerified,
					NameA:     na, NameB: nb,
					StartA:    startOf(a), StartB: startOf(b),
				})
			}
		}
	}
	return out
}

// fixtureNamesMatchExport 复用 marketmap 队名匹配（同包不可导出，此处复制调用入口）。
func fixtureNamesMatchExport(a, b string) bool {
	return marketmap.FixtureNamesMatchForAudit(a, b)
}

func findSuspectWrongMapKey(rows []persistententity.Market) []wrongPair {
	byKey := map[string][]*persistententity.Market{}
	for i := range rows {
		m := &rows[i]
		k := mapKeyOf(m)
		if k == "" {
			continue
		}
		byKey[k] = append(byKey[k], m)
	}
	var out []wrongPair
	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		list := byKey[k]
		if len(list) < 2 {
			continue
		}
		for i := 0; i < len(list); i++ {
			for j := i + 1; j < len(list); j++ {
				a, b := list[i], list[j]
				if strings.EqualFold(a.Platform, b.Platform) {
					continue
				}
				na, nb := nameOf(a), nameOf(b)
				reason := ""
				if na != "" && nb != "" && !fixtureNamesMatchExport(na, nb) {
					reason = "队名不匹配"
				}
				if dh, ok := startDeltaHours(startOf(a), startOf(b)); ok && dh > maxStartDeltaHours {
					if reason != "" {
						reason += " + "
					}
					reason += fmt.Sprintf("开赛差%dh", dh)
				}
				if marketScope(a) != "" && marketScope(b) != "" && marketScope(a) != marketScope(b) {
					if reason != "" {
						reason += " + "
					}
					reason += "联赛桶不一致"
				}
				if reason == "" {
					continue
				}
				out = append(out, wrongPair{
					MapKey:    k,
					Reason:    reason,
					PlatformA: strings.ToUpper(a.Platform),
					PlatformB: strings.ToUpper(b.Platform),
					IDA: a.ID, IDB: b.ID,
					NameA: na, NameB: nb,
					StartA: startOf(a), StartB: startOf(b),
				})
			}
		}
	}
	return out
}

func countNoMapKey(rows []persistententity.Market) int {
	n := 0
	for i := range rows {
		if mapKeyOf(&rows[i]) == "" {
			n++
		}
	}
	return n
}

func countHasKeyUnverified(rows []persistententity.Market) int {
	n := 0
	for i := range rows {
		if mapKeyOf(&rows[i]) != "" && rows[i].MapVerified == 0 {
			n++
		}
	}
	return n
}

func nullStr(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return strings.TrimSpace(*p)
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}