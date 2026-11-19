package polymarket

import (
	"context"
	"fmt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	apphttp "github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http"
)

func serveDataAPI(t *testing.T, handler http.HandlerFunc) *Polymarket {
	t.Helper()
	srv := httptest.NewServer(handler)
	old := dataAPIBaseURL
	dataAPIBaseURL = srv.URL
	t.Cleanup(func() { dataAPIBaseURL = old; srv.Close() })
	return &Polymarket{client: apphttp.NewHttpClient()}
}

func TestGetPositionsV2PaginationAndMapping(t *testing.T) {
	var seen int
	p := serveDataAPI(t, func(w http.ResponseWriter, r *http.Request) {
		seen++
		if r.URL.Path != "/v2/positions" || r.URL.Query().Get("user") != "0xwallet" || r.URL.Query().Get("condition") != "0xcondition" {
			t.Errorf("unexpected request: %s", r.URL.String())
		}
		if seen == 1 {
			if r.URL.Query().Get("cursor") != "" {
				t.Errorf("first request has cursor")
			}
			fmt.Fprint(w, `{"data":[{"proxy_wallet":"0xwallet","token_id":"token-1","condition_id":"0xcondition","current_size":3.5,"avg_price":0.4,"entry_cost_usdc":1.4,"current_value":1.75,"unrealized_pnl":0.35,"percent_pnl":25,"total_size":4,"realized_pnl":0.1,"percent_realized_pnl":7,"current_price":0.5,"title":"Question","outcome":"Yes","outcome_index":0,"opposite_token_id":"token-2","end_date":"2030-01-01","negative_risk":true}],"pagination":{"has_more":true,"next_cursor":"next"}}`)
			return
		}
		if r.URL.Query().Get("cursor") != "next" {
			t.Errorf("second request cursor=%q", r.URL.Query().Get("cursor"))
		}
		fmt.Fprint(w, `{"data":[],"pagination":{"has_more":false,"next_cursor":""}}`)
	})
	resp, err := p.GetPositionsV2(context.Background(), "0xwallet", "0xcondition")
	if err != nil {
		t.Fatal(err)
	}
	if seen != 2 || !resp.Success || resp.Total != 1 || len(resp.Positions) != 1 {
		t.Fatalf("unexpected response: %#v, requests=%d", resp, seen)
	}
	pos := resp.Positions[0]
	if pos.TokenID != "token-1" || pos.ConditionID != "0xcondition" || pos.Balance != "3.500000" || pos.InitialValue != 1.4 || pos.OppositeAsset != "token-2" || !pos.NegativeRisk {
		t.Fatalf("mapping mismatch: %#v", pos)
	}
}

func TestGetPositionsV2NullData(t *testing.T) {
	p := serveDataAPI(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":null,"pagination":{"has_more":false}}`)
	})
	resp, err := p.GetPositionsV2(context.Background(), "0xwallet", "")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.Total != 0 || len(resp.Positions) != 0 {
		t.Fatalf("unexpected empty result: %#v", resp)
	}
}

func TestGetTradeHistoryV3V2PaginationAndQuery(t *testing.T) {
	var seen int
	p := serveDataAPI(t, func(w http.ResponseWriter, r *http.Request) {
		seen++
		q := r.URL.Query()
		if r.URL.Path != "/v2/activity" || q.Get("user") != "0xproxy" || q.Get("condition") != "0xcond" || q.Get("type") != "TRADE" || q.Get("start") != "1700000000" || q.Get("end") != "1700001000" || q.Get("limit") != "1" {
			t.Errorf("unexpected query: %v", q)
		}
		if q.Get("offset") != "" || q.Get("excludeDepositsWithdrawals") != "false" || q.Get("sortBy") != "TIMESTAMP" || q.Get("sortDirection") != "DESC" {
			t.Errorf("missing v2 compatibility params: %v", q)
		}
		if seen == 1 {
			fmt.Fprint(w, `{"data":[{"proxy_wallet":"0xproxy","timestamp":1700000001,"condition_id":"0xcond","type":"TRADE","size":2.5,"usdc_size":1.125,"transaction_hash":"tx1","price":0.45,"token_id":"token","side":"BUY","outcome_index":1,"outcome":"No"}],"pagination":{"has_more":true,"next_cursor":"c1"}}`)
			return
		}
		if q.Get("cursor") != "c1" {
			t.Errorf("cursor=%q", q.Get("cursor"))
		}
		fmt.Fprint(w, `{"data":[{"proxy_wallet":"0xproxy","timestamp":1700000000,"condition_id":"0xcond","type":"TRADE","size":1,"usdc_size":0.4,"transaction_hash":"tx2","price":0.4,"token_id":"token2","side":"SELL","outcome_index":0,"outcome":"Yes"}],"pagination":{"has_more":false}}`)
	})
	after, before, limit, typ, cond := int64(1700000000), int64(1700001000), 1, "TRADE", "0xcond"
	resp, err := p.GetTradeHistoryV3V2(context.Background(), entity.GetTradeHistoryRequest{After: &after, Before: &before, Limit: &limit, Type: &typ, Market: &cond}, entity.PolymarketAccount{ProxyAddress: "0xproxy"})
	if err != nil {
		t.Fatal(err)
	}
	if seen != 2 || !resp.Success || resp.Total != 2 {
		t.Fatalf("unexpected result: %#v requests=%d", resp, seen)
	}
	if resp.Trades[0].TransactionHash != "tx1" || resp.Trades[0].AssetId != "token" || resp.Trades[0].UsdcSize != "1.125000" || resp.Trades[0].MatchTime != "1700000001" {
		t.Fatalf("mapping mismatch: %#v", resp.Trades[0])
	}
}

func TestGetTradeHistoryV3V2SplitsConditions(t *testing.T) {
	var calls int
	p := serveDataAPI(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		conditions := strings.Split(r.URL.Query().Get("condition"), ",")
		want := 20
		if calls == 2 {
			want = 1
		}
		if len(conditions) != want {
			t.Errorf("call %d has %d conditions, want %d", calls, len(conditions), want)
		}
		fmt.Fprint(w, `{"data":[],"pagination":{"has_more":false}}`)
	})
	ids := make([]string, 21)
	for i := range ids {
		ids[i] = fmt.Sprintf("condition-%d", i)
	}
	resp, err := p.GetTradeHistoryV3V2(context.Background(), entity.GetTradeHistoryRequest{Markets: ids}, entity.PolymarketAccount{ProxyAddress: "0xproxy"})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || !resp.Success || resp.Total != 0 {
		t.Fatalf("unexpected result: %#v calls=%d", resp, calls)
	}
}

func TestGetTradeHistoryV3V2RejectsInvalidAccountAndLimit(t *testing.T) {
	p := &Polymarket{client: apphttp.NewHttpClient()}
	if _, err := p.GetTradeHistoryV3V2(context.Background(), entity.GetTradeHistoryRequest{}, entity.PolymarketAccount{}); err == nil {
		t.Fatal("expected account validation error")
	}
	invalid := 1001
	if _, err := p.GetTradeHistoryV3V2(context.Background(), entity.GetTradeHistoryRequest{Limit: &invalid}, entity.PolymarketAccount{ProxyAddress: "0xproxy"}); err == nil {
		t.Fatal("expected limit validation error")
	}
}

func TestGetTradeHistoryV3V2RepeatedCursorErrors(t *testing.T) {
	p := serveDataAPI(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[],"pagination":{"has_more":true,"next_cursor":"same"}}`)
	})
	_, err := p.GetTradeHistoryV3V2(context.Background(), entity.GetTradeHistoryRequest{}, entity.PolymarketAccount{ProxyAddress: "0xproxy"})
	if err == nil || !strings.Contains(err.Error(), "cursor repeated") {
		t.Fatalf("expected repeated cursor error, got %v", err)
	}
}

func TestDataAPIConditionQueryEscaped(t *testing.T) {
	got := url.Values{"condition": {"0xone,0xtwo"}}.Encode()
	if got != "condition=0xone%2C0xtwo" {
		t.Fatalf("unexpected encoding %q", got)
	}
}
