package yesno

import (
	"encoding/json"
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// sxbetBinaryArbRank SX 跨所二元 YES+NO 应优先「队 vs Not 队 [1X2]」，勿用「队 vs 对手 [Moneyline]」。
func sxbetBinaryArbRank(oc *persistententity.MarketOutcome) int {
	if oc == nil {
		return 0
	}
	name := ""
	if oc.OutcomeName != nil {
		name = strings.TrimSpace(*oc.OutcomeName)
	}
	if strings.Contains(name, "1X2") {
		return 100
	}
	if strings.Contains(name, "[Moneyline]") {
		return 10
	}
	if no := sxbetConfigLabel(oc.Config, "no_label"); strings.HasPrefix(strings.ToLower(no), "not ") {
		return 80
	}
	return 50
}

func sxbetConfigLabel(cfg *string, key string) string {
	if cfg == nil {
		return ""
	}
	raw := strings.TrimSpace(*cfg)
	if raw == "" {
		return ""
	}
	var m map[string]string
	if json.Unmarshal([]byte(raw), &m) != nil {
		return ""
	}
	return strings.TrimSpace(m[key])
}

// AddMarketOutcome 同 OutcomeGroup 注册平台腿；SX 多条时保留 sxbetBinaryArbRank 更高者。
func (g *OutcomeGroup) AddMarketOutcome(oc *persistententity.MarketOutcome, outcomeIndex int, prev func(int) *persistententity.MarketOutcome) {
	if g.byPlatform == nil {
		g.byPlatform = map[string]int{}
	}
	if oc == nil {
		return
	}
	platform := strings.ToUpper(strings.TrimSpace(oc.Platform))
	if platform == "" {
		return
	}
	if platform != appcommon.PlatformSxbet {
		if _, exists := g.byPlatform[platform]; !exists {
			g.byPlatform[platform] = outcomeIndex
		}
		return
	}
	if cur, exists := g.byPlatform[platform]; exists && prev != nil {
		if sxbetBinaryArbRank(oc) <= sxbetBinaryArbRank(prev(cur)) {
			return
		}
	}
	g.byPlatform[platform] = outcomeIndex
}