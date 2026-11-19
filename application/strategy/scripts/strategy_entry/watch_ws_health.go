package strategyentry

import (
	"strings"
	"sync"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	kalshiws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/kalshi"
	polyws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/polymarket"
	predictws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/predict"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

const (
	wsStaleAfter   = 4 * time.Minute
	wsConnectGrace = 90 * time.Second
)

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
	default:
		return false
	}
}

func (w *watch) platformExpected(platform string) bool {
	if w == nil {
		return false
	}
	return outcomesPlatformWSExpected(w.entry, w.Outcomes, platform)
}

// expectedWSCounts 按库内应对场次统计各平台应开的 WS 条数（每组每平台最多 1 条）。
func expectedWSCounts(e *Entry, groups []group) map[string]int {
	platforms := []string{
		appcommon.PlatformPolymarket,
		appcommon.PlatformKalshi,
		appcommon.PlatformPredict,
	}
	out := make(map[string]int, len(platforms))
	for _, p := range platforms {
		out[p] = 0
	}
	for _, g := range groups {
		for _, p := range platforms {
			if outcomesPlatformWSExpected(e, g.Outcomes, p) {
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