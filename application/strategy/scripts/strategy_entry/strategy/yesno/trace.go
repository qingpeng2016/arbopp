package yesno

import (
	"fmt"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

func newArbID() string { return NewArbID() }

// NewArbID 跨腿 / 对端 HEDGE 关联 id。
func NewArbID() string {
	return fmt.Sprintf("%s-%d", conf.GetRunNode(), time.Now().UnixNano())
}

// FireTraceBase 1X2 等策略的耗时/盘口年龄（不含双腿归因）。
func FireTraceBase(
	arbID, triggerPlatform string,
	wsStart, computeStart, fireStart, fireEnd time.Time,
	platforms []string,
	bookAgeMs func(string) int64,
) map[string]any {
	out := map[string]any{
		"arb_id":           arbID,
		"node":             conf.GetRunNode(),
		"trigger_platform": strings.ToUpper(strings.TrimSpace(triggerPlatform)),
	}
	if !wsStart.IsZero() && !fireEnd.IsZero() {
		out["ws_to_fire_ms"] = roundMs(float64(fireEnd.Sub(wsStart)) / float64(time.Millisecond))
	}
	if !computeStart.IsZero() && !fireStart.IsZero() {
		out["compute_ms"] = roundMs(float64(fireStart.Sub(computeStart)) / float64(time.Millisecond))
	}
	if !fireStart.IsZero() && !fireEnd.IsZero() {
		out["fire_ms"] = roundMs(float64(fireEnd.Sub(fireStart)) / float64(time.Millisecond))
	}
	if bookAgeMs != nil {
		ages := map[string]int64{}
		for _, p := range platforms {
			p = strings.ToUpper(strings.TrimSpace(p))
			if p == "" {
				continue
			}
			if ms := bookAgeMs(p); ms >= 0 {
				ages[p] = ms
			}
		}
		if len(ages) > 0 {
			out["book_age_ms"] = ages
		}
	}
	return out
}

func fireTraceMap(
	arbID, triggerPlatform string,
	wsStart, computeStart, fireStart, fireEnd time.Time,
	platforms []string,
	bookAgeMs func(string) int64,
	yesSend, noSend watchgroup.SendResult,
	yesErr, noErr error,
) map[string]any {
	out := map[string]any{
		"arb_id":           arbID,
		"node":             conf.GetRunNode(),
		"trigger_platform": strings.ToUpper(strings.TrimSpace(triggerPlatform)),
	}
	if !wsStart.IsZero() && !fireEnd.IsZero() {
		out["ws_to_fire_ms"] = roundMs(float64(fireEnd.Sub(wsStart)) / float64(time.Millisecond))
	}
	if !computeStart.IsZero() && !fireStart.IsZero() {
		out["compute_ms"] = roundMs(float64(fireStart.Sub(computeStart)) / float64(time.Millisecond))
	}
	if !fireStart.IsZero() && !fireEnd.IsZero() {
		out["fire_ms"] = roundMs(float64(fireEnd.Sub(fireStart)) / float64(time.Millisecond))
	}
	if bookAgeMs != nil {
		ages := map[string]int64{}
		for _, p := range platforms {
			p = strings.ToUpper(strings.TrimSpace(p))
			if p == "" {
				continue
			}
			if ms := bookAgeMs(p); ms >= 0 {
				ages[p] = ms
			}
		}
		if len(ages) > 0 {
			out["book_age_ms"] = ages
		}
	}
	out["leg_yes_send"] = sendLegMeta(yesSend.RoutedPeer, yesSend.Fill, yesErr)
	out["leg_no_send"] = sendLegMeta(noSend.RoutedPeer, noSend.Fill, noErr)
	out["leg_risk_cause"] = classifyLegRisk(yesSend.Fill, noSend.Fill, yesErr, noErr, yesSend.RoutedPeer, noSend.RoutedPeer)
	return out
}

func sendLegMeta(routed bool, fill float64, err error) map[string]any {
	m := map[string]any{"routed_peer": routed, "fill": fill}
	if err != nil {
		m["err"] = err.Error()
	}
	return m
}

func classifyLegRisk(yesFill, noFill float64, yesErr, noErr error, yesRouted, noRouted bool) string {
	if yesErr != nil {
		if yesRouted || strings.Contains(strings.ToLower(yesErr.Error()), "relay") || strings.Contains(strings.ToLower(yesErr.Error()), "peer") {
			return "PEER_HEDGE_FAIL"
		}
		return "EXCHANGE_ERR_YES"
	}
	if noErr != nil {
		if noRouted || strings.Contains(strings.ToLower(noErr.Error()), "relay") || strings.Contains(strings.ToLower(noErr.Error()), "peer") {
			return "PEER_HEDGE_FAIL"
		}
		return "EXCHANGE_ERR_NO"
	}
	if yesRouted && noRouted {
		return "BOTH_ROUTED_PEER"
	}
	if yesFill > 0 && noFill <= 0 && !noRouted {
		return "LEG_RISK_NO_FILL"
	}
	if noFill > 0 && yesFill <= 0 && !yesRouted {
		return "LEG_RISK_YES_FILL"
	}
	if yesFill <= 0 && noFill <= 0 && !yesRouted && !noRouted {
		return "EXCHANGE_NO_FILL"
	}
	if yesFill > 0 && noFill > 0 {
		return "OK_BOTH"
	}
	if yesRouted || noRouted {
		return "ROUTED_AWAIT_PEER"
	}
	return "UNKNOWN"
}

func roundMs(v float64) float64 {
	if v < 0 {
		return 0
	}
	return float64(int64(v*100+0.5)) / 100
}