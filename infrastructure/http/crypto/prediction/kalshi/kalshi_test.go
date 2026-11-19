package kalshi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	repo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func mockClient(fn func(*http.Request) (int, string)) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		status, body := fn(r)
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
}

type stubSigner struct {
	calls               int
	method, path, keyID string
}

func (s *stubSigner) Sign(_ context.Context, keyID, _, method, path string) (string, error) {
	s.calls++
	s.keyID, s.method, s.path = keyID, method, path
	return "mock-signature", nil
}

func TestPublicEndpoints(t *testing.T) {
	client := mockClient(func(r *http.Request) (int, string) {
		switch r.URL.Path {
		case apiPrefix + "/markets/MKT":
			return 200, `{"market":{"ticker":"MKT","yes_bid_dollars":"0.42"}}`
		case apiPrefix + "/markets":
			if r.URL.Query().Get("event_ticker") != "EV" || r.URL.Query().Get("status") != "open" {
				t.Errorf("unexpected query: %s", r.URL.RawQuery)
			}
			return 200, `{"markets":[{"ticker":"MKT"}],"cursor":"next"}`
		case apiPrefix + "/events":
			if r.URL.Query().Get("series_ticker") != "SER" || r.URL.Query().Get("with_nested_markets") != "true" {
				t.Errorf("unexpected events query: %s", r.URL.RawQuery)
			}
			return 200, `{"events":[{"category":"Sports","event_ticker":"EV","markets":[{"ticker":"MKT"}]}],"cursor":""}`
		case apiPrefix + "/events/EV":
			if r.URL.Query().Get("with_nested_markets") != "true" {
				t.Error("missing nested markets query")
			}
			return 200, `{"event":{"markets":[{"ticker":"MKT"}]}}`
		case apiPrefix + "/markets/MKT/orderbook":
			if r.URL.Query().Get("depth") != "10" {
				t.Error("bad depth")
			}
			return 200, `{"orderbook_fp":{"yes_dollars":[["0.42","10.00"]]}}`
		case apiPrefix + "/markets/orderbooks":
			if len(r.URL.Query()["tickers"]) != 2 {
				t.Error("tickers must be repeated query values")
			}
			return 200, `{"orderbooks":[{"ticker":"MKT","orderbook_fp":{"yes_dollars":[]}}]}`
		default:
			return 404, "not found"
		}
	})
	c := NewClientWithBatchGap("http://kalshi.test", client, nil, 0)
	ctx := context.Background()
	m, err := c.GetMarket(ctx, "MKT")
	if err != nil || m.YesBidDollars != "0.42" {
		t.Fatalf("GetMarket: %#v %v", m, err)
	}
	ml, err := c.GetMarkets(ctx, repo.KalshiListParams{Ticker: "EV", Status: "open", Limit: 2})
	if err != nil || ml.Cursor != "next" || len(ml.Markets) != 1 {
		t.Fatalf("GetMarkets: %#v %v", ml, err)
	}
	evs, err := c.GetEvents(ctx, repo.KalshiListParams{
		SeriesTicker: "SER", Status: "open", Category: "Sports", WithNestedMarkets: true, Limit: 5,
	})
	if err != nil || len(evs.Events) != 1 || len(evs.Events[0].Markets) != 1 {
		t.Fatalf("GetEvents: %#v %v", evs, err)
	}
	ev, err := c.GetEvent(ctx, "EV")
	if err != nil || len(ev.Event.Markets) != 1 {
		t.Fatalf("GetEvent: %#v %v", ev, err)
	}
	b, err := c.GetOrderbook(ctx, "MKT", 10)
	if err != nil || len(b.Yes) != 1 {
		t.Fatalf("GetOrderbook: %#v %v", b, err)
	}
	books, err := c.GetOrderbooks(ctx, []string{"MKT", "OTHER"})
	if err != nil || len(books) != 1 {
		t.Fatalf("GetOrderbooks: %#v %v", books, err)
	}
}

func TestAuthenticatedPortfolioEndpoints(t *testing.T) {
	signer := &stubSigner{}
	client := mockClient(func(r *http.Request) (int, string) {
		if r.Header.Get("KALSHI-ACCESS-KEY") != "kid" || r.Header.Get("KALSHI-ACCESS-SIGNATURE") != "mock-signature" || r.Header.Get("KALSHI-ACCESS-TIMESTAMP") == "" {
			t.Errorf("missing auth headers: %v", r.Header)
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == apiPrefix+"/portfolio/events/orders":
			var body entity.KalshiPlaceOrderReq
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Price != "0.5" || body.Side != "bid" {
				t.Errorf("bad place body: %#v", body)
			}
			return 201, `{"order_id":"OID","fill_count":"0.00","remaining_count":"1.00"}`
		case r.Method == http.MethodDelete:
			return 200, `{"order_id":"OID","reduced_by":"1.00"}`
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/amend"):
			return 200, `{"order_id":"OID2","remaining_count":"2.00"}`
		case r.URL.Path == apiPrefix+"/portfolio/orders/OID":
			return 200, `{"order":{"order_id":"OID","status":"resting"}}`
		case r.URL.Path == apiPrefix+"/portfolio/orders":
			return 200, `{"orders":[{"order_id":"OID"}]}`
		case r.URL.Path == apiPrefix+"/portfolio/positions":
			return 200, `{"market_positions":[{"ticker":"MKT"}]}`
		case r.URL.Path == apiPrefix+"/portfolio/fills":
			if r.URL.Query().Get("order_id") != "OID" {
				t.Error("missing order_id query")
			}
			return 200, `{"fills":[{"fill_id":"F"}]}`
		case r.URL.Path == apiPrefix+"/portfolio/balance":
			return 200, `{"balance":123,"balance_dollars":"1.23"}`
		default:
			return 404, "not found"
		}
	})
	c := NewClientWithOptions("http://kalshi.test", client, signer)
	ctx, account := context.Background(), entity.KalshiAccount{KeyID: "kid"}
	order, err := c.PlaceOrder(ctx, entity.KalshiPlaceOrderReq{Ticker: "MKT", Side: "bid", Count: "1.00", Price: "0.5"}, account)
	if err != nil || order.OrderID != "OID" {
		t.Fatalf("PlaceOrder: %#v %v", order, err)
	}
	if signer.method != http.MethodPost || signer.path != apiPrefix+"/portfolio/events/orders" {
		t.Fatalf("wrong signature path: %#v", signer)
	}
	order, err = c.GetOrder(ctx, "OID", account)
	if err != nil || order.Status != "resting" {
		t.Fatalf("GetOrder: %#v %v", order, err)
	}
	order, err = c.AmendOrder(ctx, "OID", entity.KalshiAmendOrderReq{Ticker: "MKT", Side: "bid", Count: "2.00"}, account)
	if err != nil || order.OrderID != "OID2" {
		t.Fatalf("AmendOrder: %#v %v", order, err)
	}
	order, err = c.CancelOrder(ctx, "OID", account)
	if err != nil || order.Status != "canceled" {
		t.Fatalf("CancelOrder: %#v %v", order, err)
	}
	orders, err := c.GetOrders(ctx, repo.KalshiListParams{Status: "resting"}, account)
	if err != nil || len(orders.Orders) != 1 {
		t.Fatalf("GetOrders: %#v %v", orders, err)
	}
	positions, err := c.GetPositions(ctx, repo.KalshiListParams{Limit: 1}, account)
	if err != nil || len(positions.MarketPositions) != 1 {
		t.Fatalf("GetPositions: %#v %v", positions, err)
	}
	fills, err := c.GetFills(ctx, repo.KalshiListParams{Limit: 1, OrderID: "OID"}, account)
	if err != nil || len(fills.Fills) != 1 {
		t.Fatalf("GetFills: %#v %v", fills, err)
	}
	balance, err := c.GetBalance(ctx, account)
	if err != nil || balance.BalanceDollars != "1.23" {
		t.Fatalf("GetBalance: %#v %v", balance, err)
	}
	if signer.calls != 8 {
		t.Fatalf("signer calls=%d, want 8", signer.calls)
	}
}

func TestPaginationAndOrderbookBatching(t *testing.T) {
	var posCalls, fillCalls, bookCalls int
	client := mockClient(func(r *http.Request) (int, string) {
		switch r.URL.Path {
		case apiPrefix + "/portfolio/positions":
			posCalls++
			if posCalls == 1 {
				return 200, `{"market_positions":[{"ticker":"A"}],"cursor":"p2"}`
			}
			if r.URL.Query().Get("cursor") != "p2" {
				t.Error("position cursor not propagated")
			}
			return 200, `{"market_positions":[{"ticker":"B"}]}`
		case apiPrefix + "/portfolio/fills":
			fillCalls++
			if fillCalls == 1 {
				return 200, `{"fills":[{"fill_id":"1"}],"cursor":"f2"}`
			}
			if r.URL.Query().Get("cursor") != "f2" {
				t.Error("fill cursor not propagated")
			}
			return 200, `{"fills":[{"fill_id":"2"}]}`
		case apiPrefix + "/markets/orderbooks":
			bookCalls++
			if len(r.URL.Query()["tickers"]) > 100 {
				t.Error("batch over 100")
			}
			return 200, `{"orderbooks":[]}`
		default:
			return 404, "not found"
		}
	})
	c := NewClientWithBatchGap("http://kalshi.test", client, &stubSigner{}, 0)
	account := entity.KalshiAccount{KeyID: "kid"}
	p, err := c.GetPositions(context.Background(), repo.KalshiListParams{}, account)
	if err != nil || len(p.MarketPositions) != 2 || posCalls != 2 {
		t.Fatalf("positions %#v calls=%d err=%v", p, posCalls, err)
	}
	f, err := c.GetFills(context.Background(), repo.KalshiListParams{}, account)
	if err != nil || len(f.Fills) != 2 || fillCalls != 2 {
		t.Fatalf("fills %#v calls=%d err=%v", f, fillCalls, err)
	}
	tickers := make([]string, 205)
	for i := range tickers {
		tickers[i] = fmt.Sprintf("T%d", i)
	}
	if _, err := c.GetOrderbooks(context.Background(), tickers); err != nil || bookCalls != 3 {
		t.Fatalf("book calls=%d err=%v", bookCalls, err)
	}
}

func TestErrorsAndInputValidation(t *testing.T) {
	c := NewClientWithOptions("http://kalshi.test", nil, nil)
	if _, err := c.GetMarket(context.Background(), ""); err == nil {
		t.Fatal("expected missing ticker error")
	}
	if err := c.doRequest(context.Background(), http.MethodGet, "/private", nil, nil, &entity.KalshiAccount{KeyID: "kid"}, nil); err == nil {
		t.Fatal("expected missing signer")
	}
	client := mockClient(func(_ *http.Request) (int, string) { return 401, "unauthorized" })
	c = NewClientWithOptions("http://kalshi.test", client, &stubSigner{})
	err := c.doRequest(context.Background(), http.MethodGet, "/private", url.Values{"cursor": []string{"a b"}}, nil, &entity.KalshiAccount{KeyID: "kid"}, nil)
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("unexpected HTTP error: %v", err)
	}
}
