package main

import "testing"

func TestLegPlatformFromInfo(t *testing.T) {
	platforms := map[uint64]string{7: "kalshi", 8: "POLYMARKET", 9: "SXBET", 9007199254740993: "PREDICT"}
	for _, tt := range []struct {
		name, info, want string
		wantErr          bool
	}{
		{"old YES without URL", `{"yes_market_id":"8","yes_outcome_id":7}`, "KALSHI", false},
		{"NO with URL and ID", `{"no_outcome_id":8,"market_url":"https://polymarket.com/event/a"}`, "POLYMARKET", false},
		{"draw with quoted ID", `{"draw_outcome_id":"9"}`, "SXBET", false},
		{"large exact ID", `{"no_outcome_id":9007199254740993}`, "PREDICT", false},
		{"URL fallback after outcome deletion", `{"yes_outcome_id":123,"market_url":"https://WWW.Predict.Fun/market/a"}`, "PREDICT", false},
		{"URL only", `{"market_url":"https://sx.bet/sports/a"}`, "SXBET", false},
		{"explicit platform", `{"platform":" kalshi "}`, "KALSHI", false},
		{"empty slot", `{}`, "", false},
		{"null slot", `null`, "", false},
		{"blank slot", " \n ", "", false},
		{"empty role struct", `{"draw_market_id":"","market_url":"","draw_outcome_id":0,"draw_token_name":""}`, "", false},
		{"nonempty unmapped leg", `{"yes_market_id":"7"}`, "", true},
		{"unresolved outcome", `{"no_outcome_id":77}`, "", true},
		{"invalid JSON", `{`, "", true},
		{"not an object", `[{"platform":"KALSHI"}]`, "", true},
		{"invalid ID", `{"draw_outcome_id":2.5}`, "", true},
		{"two different ids", `{"yes_outcome_id":7,"no_outcome_id":8}`, "", true},
		{"explicit conflicts with url", `{"platform":"SXBET","market_url":"https://kalshi.com/events/a"}`, "", true},
		{"id conflicts with url", `{"yes_outcome_id":8,"market_url":"https://kalshi.com/markets/a"}`, "", true},
		{"unrecognized platform", `{"platform":"other"}`, "", true},
		{"lookalike host", `{"market_url":"https://kalshi.com.example.org/sx.bet"}`, "", true},
		{"userinfo confusion", `{"market_url":"https://predict.fun@outside.com"}`, "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			leg, err := parseLegInfo(tt.info)
			var got string
			if err == nil {
				got, err = leg.resolve(platforms)
			}
			if got != tt.want || (err != nil) != tt.wantErr {
				t.Errorf("platform = %q, error = %v; want %q, want error %t", got, err, tt.want, tt.wantErr)
			}
		})
	}
}
