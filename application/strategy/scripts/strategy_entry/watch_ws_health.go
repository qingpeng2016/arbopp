package strategyentry

import (
	"strings"
	"sync"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	entrystat "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/stat"
	kalshiws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/kalshi"
	polyws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/polymarket"
	predictws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/predict"
	sxbetws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/sxbet"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

const (
	wsStaleAfter   = 4 * time.Minute
	wsConnectGrace = 90 * time.Second
)

// WSHeartbeatPlatforms strategy_entry 心跳统计的平台列表（与每场应连 WS 一致）。
func WSHeartbeatPlatforms() []string {
	return []string{
		appcommon.PlatformPolymarket,
		appcommon.PlatformKalshi,
		appcommon.PlatformPredict,
		appcommon.PlatformSxbet,
	}
}

func wsHeartbeatPlatforms() []string { return WSHeartbeatPlatforms() }

// wsHealth 单组 map_key 上各平台 WS 最近活动（book/心跳），非 pause 窗口内视为在连。
type wsHealth struct {
	mu       sync.Mutex
	lastRx   map[string]time.Time
	connAt   map[string]time.Time
	paused   map[string]bool
}

func (w *watch) wsHealthState() *wsHealth {
	if w.wsHealth == nil {
		w.wsHealth = &wsHealth{
			lastRx: map[string]time.Time{},
			connAt: map[string]time.Time{},
			paused: map[string]bool{},
		}
	}
	return w.wsHealth
}

func outcomesPlatformWSExpected(e *Entry, outcomes []persistententity.MarketOutcome, platform string) bool {
	if e == nil {
		return false
	}
	platform = strings.ToUpper(strings.TrimSpace(platform))
	switch platform {
	case appcommon.PlatformPolymarket:
		return len(polyws.Assets(outcomes)) > 0 && e.poly != nil
	case appcommon.PlatformKalshi:
		return len(kalshiws.Tickers(outcomes)) > 0 && e.kalshi != nil
	case appcommon.PlatformPredict:
		return len(predictws.MarketIDs(outcomes)) > 0 && e.predict != nil && e.predictAPIKey() != ""
	case appcommon.PlatformSxbet:
		return len(sxbetws.MarketHashes(outcomes)) > 0 && e.sxbet != nil && e.sxbetAPIKey() != ""
	default:
		return false
	}
}

func (w *watch) platformExpected(platform string) bool {
	if w == nil {
		return false
	}
	return groupHasPlatformMarket(w.MarketsByRowID, platform)
}

func groupHasPlatformMarket(markets map[uint]persistententity.Market, platform string) bool {
	by := entrystat.MarketsByPlatform(markets)
	_, ok := by[strings.ToUpper(strings.TrimSpace(platform))]
	return ok
}

// expectedWSCounts 与 stat_kv groups[].markets 一致：每组每平台至多 1 条 WS。
func expectedWSCounts(_ *Entry, groups []group) map[string]int {
	platforms := wsHeartbeatPlatforms()
	out := make(map[string]int, len(platforms))
	for _, p := range platforms {
		out[p] = 0
	}
	platSet := make(map[string]struct{}, len(platforms))
	for _, p := range platforms {
		platSet[strings.ToUpper(p)] = struct{}{}
	}
	for _, g := range groups {
		by := entrystat.MarketsByPlatform(g.MarketsByRowID)
		for p := range by {
			if _, ok := platSet[p]; ok {
				out[p]++
			}
		}
	}
	return out
}

func (w *watch) markWSConn(platform string) {
	h := w.wsHealthState()
	h.mu.Lock()
	defer h.mu.Unlock()
	platform = strings.ToUpper(strings.TrimSpace(platform))
	h.connAt[platform] = time.Now()
	h.paused[platform] = false
}

func (w *watch) markWSPause(platform string) {
	h := w.wsHealthState()
	h.mu.Lock()
	defer h.mu.Unlock()
	platform = strings.ToUpper(strings.TrimSpace(platform))
	h.paused[platform] = true
}

func (w *watch) touchWS(platform string) {
	h := w.wsHealthState()
	h.mu.Lock()
	defer h.mu.Unlock()
	platform = strings.ToUpper(strings.TrimSpace(platform))
	now := time.Now()
	h.lastRx[platform] = now
	h.connAt[platform] = now
	h.paused[platform] = false
}

// wsLinkOK 本场次是否应维持该平台 WS，以及当前是否判为心跳正常。
func (w *watch) wsLinkOK(platform string) (expected, healthy bool) {
	if !w.platformExpected(platform) {
		return false, false
	}
	platform = strings.ToUpper(strings.TrimSpace(platform))
	h := w.wsHealthState()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.paused[platform] {
		return true, false
	}
	t, has := h.lastRx[platform]
	if has && time.Since(t) <= wsStaleAfter {
		return true, true
	}
	if !has {
		if at, ok := h.connAt[platform]; ok && time.Since(at) <= wsConnectGrace {
			return true, true
		}
	}
	return true, false
}