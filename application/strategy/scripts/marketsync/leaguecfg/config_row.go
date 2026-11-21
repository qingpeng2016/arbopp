package leaguecfg

import (
	"encoding/json"
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/leagueopt"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

const defaultLeagueFetch = `{"polymarket":{"concurrency":5},"kalshi":{"concurrency":3},"predict":{"concurrency":2},"sxbet":{"concurrency":3}}`

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func platformKeyNorm(key string) string {
	return strings.ToUpper(strings.TrimSpace(key))
}

func sportGroupFromConfig(raw string) (group string, leagueConcurrency int) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0
	}
	return sportGroupFromBlock([]byte(raw))
}

func sportGroupFromBlock(block []byte) (group string, leagueConcurrency int) {
	var m map[string]interface{}
	if err := json.Unmarshal(block, &m); err != nil {
		return "", 0
	}
	for _, key := range []string{"concurrency", "league_concurrency"} {
		v, ok := m[key]
		if !ok {
			continue
		}
		switch n := v.(type) {
		case float64:
			leagueConcurrency = int(n)
		case int:
			leagueConcurrency = n
		}
		delete(m, key)
		if leagueConcurrency > 0 {
			break
		}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", leagueConcurrency
	}
	return string(b), leagueConcurrency
}

func groupsArrayForPlatformRows(cats []persistententity.ConfigMarketCategory, platform string) string {
	plat := platformKeyNorm(platform)
	var parts []string
	for i := range cats {
		c := &cats[i]
		if platformKeyNorm(c.Platform) != plat {
			continue
		}
		grp, _ := sportGroupFromConfig(c.ConfigJSON)
		grp = strings.TrimSpace(grp)
		if grp != "" && grp != "{}" {
			parts = append(parts, grp)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func mergeLeagueFetchFromRows(cats []persistententity.ConfigMarketCategory) (leagueopt.Config, bool) {
	var merged leagueopt.Config
	got := false
	for i := range cats {
		conc := cats[i].Concurrency
		if conc <= 0 {
			_, conc = sportGroupFromConfig(cats[i].ConfigJSON)
		}
		if conc <= 0 {
			continue
		}
		got = true
		switch platformKeyNorm(cats[i].Platform) {
		case appcommon.PlatformPolymarket:
			merged.Polymarket.LeagueConcurrency = maxInt(merged.Polymarket.LeagueConcurrency, conc)
		case appcommon.PlatformKalshi:
			merged.Kalshi.LeagueConcurrency = maxInt(merged.Kalshi.LeagueConcurrency, conc)
		case appcommon.PlatformPredict:
			merged.Predict.LeagueConcurrency = maxInt(merged.Predict.LeagueConcurrency, conc)
		case appcommon.PlatformSxbet:
			merged.Sxbet.LeagueConcurrency = maxInt(merged.Sxbet.LeagueConcurrency, conc)
		}
	}
	return merged, got
}

type platformSeedRow struct {
	Platform    string
	ConfigJSON  string
	Concurrency int
	SortNo      int
}

func buildPlatformRowsFromDocs(poly, kal, prd json.RawMessage) ([]platformSeedRow, error) {
	fetch := leagueopt.Parse(defaultLeagueFetch, "json")
	polyJSON, _ := sportGroupFromConfig(string(poly))
	kalJSON, _ := sportGroupFromConfig(string(kal))
	prdJSON, _ := sportGroupFromConfig(string(prd))
	return []platformSeedRow{
		{
			Platform: appcommon.PlatformPolymarket, ConfigJSON: polyJSON,
			Concurrency: fetch.Polymarket.Concurrency(), SortNo: 1,
		},
		{
			Platform: appcommon.PlatformKalshi, ConfigJSON: kalJSON,
			Concurrency: fetch.Kalshi.Concurrency(), SortNo: 2,
		},
		{
			Platform: appcommon.PlatformPredict, ConfigJSON: prdJSON,
			Concurrency: fetch.Predict.Concurrency(), SortNo: 3,
		},
	}, nil
}