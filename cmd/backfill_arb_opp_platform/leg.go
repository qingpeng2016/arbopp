package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
)

type legInfo struct {
	OutcomeID   uint64
	URLPlatform string
	Platform    string
}

// Each slot may contain a YES, NO or draw record, depending on the writer version.
func parseLegInfo(raw string) (legInfo, error) {
	var leg legInfo
	if strings.TrimSpace(raw) == "" {
		return leg, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return leg, err
	}
	var hasDetails bool
	for key, value := range fields {
		switch key {
		case "yes_outcome_id", "no_outcome_id", "draw_outcome_id":
			var number json.Number
			if err := json.Unmarshal(value, &number); err != nil {
				return legInfo{}, fmt.Errorf("%s: %w", key, err)
			}
			if number == "" {
				continue
			}
			id, err := strconv.ParseUint(string(number), 10, 64)
			if err != nil {
				return legInfo{}, fmt.Errorf("%s: %w", key, err)
			}
			if id == 0 {
				continue
			}
			if leg.OutcomeID != 0 && leg.OutcomeID != id {
				return legInfo{}, fmt.Errorf("multiple distinct outcome IDs in one leg")
			}
			leg.OutcomeID = id
		case "market_url", "platform":
			var s string
			if err := json.Unmarshal(value, &s); err != nil {
				return legInfo{}, fmt.Errorf("%s: %w", key, err)
			}
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			hasDetails = true
			if key == "market_url" {
				leg.URLPlatform = platformFromURL(s)
				continue
			}
			switch p := strings.ToUpper(s); p {
			case appcommon.PlatformPolymarket, appcommon.PlatformKalshi, appcommon.PlatformPredict, appcommon.PlatformSxbet:
				leg.Platform = p
			default:
				return legInfo{}, fmt.Errorf("unrecognized platform %q", s)
			}
		default:
			// The writer also emits empty, zero-valued role structs for missing outcomes.
			role, name, _ := strings.Cut(key, "_")
			var s string
			if role == "yes" || role == "no" || role == "draw" {
				switch name {
				case "market_id", "market_name", "outcome_name", "token_id", "token_name":
					if json.Unmarshal(value, &s) == nil && strings.TrimSpace(s) == "" {
						continue
					}
				}
			}
			hasDetails = true
		}
	}
	if leg == (legInfo{}) && hasDetails {
		return leg, fmt.Errorf("leg has no outcome ID or identifiable platform")
	}
	return leg, nil
}

func platformFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return ""
	}
	host := strings.TrimPrefix(strings.TrimSuffix(strings.ToLower(u.Hostname()), "."), "www.")
	switch host {
	case "polymarket.com":
		return appcommon.PlatformPolymarket
	case "kalshi.com":
		return appcommon.PlatformKalshi
	case "predict.fun":
		return appcommon.PlatformPredict
	case "sx.bet":
		return appcommon.PlatformSxbet
	}
	return ""
}

func (leg legInfo) resolve(platformByID map[uint64]string) (string, error) {
	if leg == (legInfo{}) {
		return "", nil
	}
	result := leg.Platform
	outcomePlatform := strings.ToUpper(strings.TrimSpace(platformByID[leg.OutcomeID]))
	for _, platform := range []string{leg.URLPlatform, outcomePlatform} {
		if platform == "" {
			continue
		}
		if result != "" && result != platform {
			return "", fmt.Errorf("conflicting leg platforms %s and %s", result, platform)
		}
		result = platform
	}
	if result == "" {
		return "", fmt.Errorf("no platform for market_outcome.id=%d", leg.OutcomeID)
	}
	return result, nil
}
