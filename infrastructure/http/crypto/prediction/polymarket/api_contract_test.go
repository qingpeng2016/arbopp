package polymarket

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	apphttp "github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http"
)

// TestPolyPublicAndAccountHTTPContracts exercises the public HTTP surface against local servers only.
// Private CLOB requests sign with a throwaway private key; no exchange account is contacted.
func TestPolyPublicAndAccountHTTPContracts(t *testing.T) {
	seen := map[string]int{}
	clob := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.Method+" "+r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/orders":
			var payload []map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || len(payload) != 1 {
				t.Errorf("invalid order body: %v %#v", err, payload)
			}
			if payload[0]["order"] == nil {
				t.Error("order payload missing signed order")
			}
			_, _ = io.WriteString(w, `[{"orderID":"order-1","status":"live","message":"accepted"}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/data/order/order-1":
			_, _ = io.WriteString(w, `{"order":{"id":"order-1","status":"LIVE","market":"condition","original_size":"2","size_matched":"0","asset_id":"token"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/data/trades":
			if r.URL.Query().Get("id") == "source-trade-id" {
				_, _ = io.WriteString(w, `{"data":[{"id":"source-trade-id","market":"condition","asset_id":"token","side":"BUY","size":"2","price":"0.5","fee_rate_bps":"0","match_time":"1700000000"}],"count":1}`)
				return
			}
			if r.URL.Query().Get("asset_id") != "token" || r.URL.Query().Get("limit") != "10" {
				t.Errorf("trade query mismatch: %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"data":[{"id":"trade-1","market":"condition","asset_id":"token","side":"BUY","size":"2","price":"0.5","fee_rate_bps":"0","match_time":"2026-01-01T00:00:00Z"}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/data/orders":
			_, _ = io.WriteString(w, `{"orders":[{"id":"order-1","market":"condition","asset_id":"token","status":"LIVE","original_size":"2","size_matched":"0","price":"0.5","side":"BUY"}],"total":1}`)
		case r.Method == http.MethodGet && r.URL.Path == "/fee-rate":
			if r.URL.Query().Get("token_id") != "token" {
				t.Errorf("fee query mismatch: %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"base_fee":0}`)
		case r.Method == http.MethodGet && r.URL.Path == "/book":
			_, _ = io.WriteString(w, `{"market":"condition","asset_id":"token","bids":[{"price":"0.4","size":"2"}],"asks":[{"price":"0.5","size":"3"}],"tick_size":"0.01"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/books":
			var payload []map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || len(payload) != 2 {
				t.Errorf("invalid books body: %v %#v", err, payload)
			}
			_, _ = io.WriteString(w, `[{"market":"condition","asset_id":"token","bids":[],"asks":[]},{"market":"condition","asset_id":"token-2","bids":[],"asks":[]}]`)
		case r.Method == http.MethodDelete && r.URL.Path == "/orders":
			var ids []string
			if err := json.NewDecoder(r.Body).Decode(&ids); err != nil || len(ids) != 1 || ids[0] != "order-1" {
				t.Errorf("cancel body mismatch: %v %v", ids, err)
			}
			_, _ = io.WriteString(w, `{"canceled":["order-1"],"not_canceled":{}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/auth/api-key":
			_, _ = io.WriteString(w, `{"apiKey":"local-api-key","secret":"local-secret","passphrase":"local-pass"}`)
		default:
			t.Errorf("unexpected CLOB endpoint: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer clob.Close()
	oldClob := clobBaseURL
	clobBaseURL = clob.URL
	t.Cleanup(func() { clobBaseURL = oldClob })

	gamma := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.Method+" "+r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/markets/market-id":
			_, _ = io.WriteString(w, `{"id":"market-id","question":"local question","conditionId":"condition","closed":false}`)
		case r.Method == http.MethodGet && r.URL.Path == "/markets" && r.URL.Query().Get("condition_ids") != "":
			_, _ = io.WriteString(w, `[{"id":"market-id","question":"local question","conditionId":"condition"}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/markets":
			_, _ = io.WriteString(w, `[{"id":"market-id","question":"local question","conditionId":"condition"}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/sports":
			_, _ = io.WriteString(w, `[]`)
		case r.Method == http.MethodGet && r.URL.Path == "/events/pagination":
			_, _ = io.WriteString(w, `{"data":[],"pagination":{"hasMore":false,"totalResults":0}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/events/keyset":
			_, _ = io.WriteString(w, `{"events":[],"next_cursor":""}`)
		case r.Method == http.MethodGet && r.URL.Path == "/events/slug/local-event":
			_, _ = io.WriteString(w, `{"id":"event-id","title":"local event","slug":"local-event","markets":[]}`)
		default:
			t.Errorf("unexpected Gamma endpoint: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer gamma.Close()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	p := NewPolymarket(apphttp.NewHttpClient()).(*Polymarket)
	p.baseURL, p.orderBookURL = gamma.URL, clob.URL
	ctx := context.Background()
	account := entity.PolymarketAccount{
		APIKey: "test-ak", APISecret: "test-sk", APIPassphrase: "test-pass",
		EOAAddress: testFunder, ProxyAddress: testFunder, SignatureType: 1,
		PrivateKey: hex.EncodeToString(crypto.FromECDSA(key)),
	}

	placed, err := p.PlaceOrder(ctx, entity.PlaceOrderReq{TokenID: testTokenID, LimitPrice: "0.5", LimitSize: "2", Side: "BUY", OrderType: "GTC"}, account)
	if err != nil || placed.OrderID != "order-1" {
		t.Fatalf("PlaceOrder: %#v err=%v", placed, err)
	}
	if _, err := p.GetOrderById(ctx, entity.GetOrderReq{OrderID: "order-1"}, account); err != nil {
		t.Fatal(err)
	}
	if _, err := p.GetTrades(ctx, entity.GetTradesReq{TokenID: "token", Limit: 10}, account); err != nil {
		t.Fatal(err)
	}
	byID, err := p.GetTrades(ctx, entity.GetTradesReq{TradeID: "source-trade-id"}, account)
	if err != nil || byID.Total != 1 || byID.Trades[0].ID != "source-trade-id" {
		t.Fatalf("GetTrades by id: %#v err=%v", byID, err)
	}
	if _, err := p.GetAllOrders(ctx, account, nil); err != nil {
		t.Fatal(err)
	}
	fee, err := p.GetPolymarketFeeRate(ctx, "token", account)
	if err != nil || fee != "0" {
		t.Fatalf("fee rate=%q err=%v", fee, err)
	}
	book, err := p.GetOrderBook(ctx, entity.GetOrderBookRequest{TokenID: "token"})
	if err != nil || len(book.Asks) != 1 {
		t.Fatalf("orderbook=%#v err=%v", book, err)
	}
	books, err := p.BatchGetOrderBooks(ctx, entity.BatchGetOrderBooksRequest{Items: []entity.BatchGetOrderBooksRequestItem{{TokenID: "token"}, {TokenID: "token-2"}}})
	if err != nil || len(books.OrderBooks) != 2 {
		t.Fatalf("batch orderbooks=%#v err=%v", books, err)
	}
	canceled, err := p.BatchCancelOrder(ctx, entity.BatchCancelOrderRequest{OrderIDs: []string{"order-1"}}, account)
	if err != nil || canceled.Successful != 1 {
		t.Fatalf("cancel=%#v err=%v", canceled, err)
	}
	if _, err := p.GetMarketByMarketId(ctx, entity.GetMarketRequest{MarketID: "market-id"}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.GetMarketsByMarketAddressList(ctx, entity.GetMarketsByConditionIdsRequest{ConditionIDs: []string{"condition"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ListMarkets(ctx, entity.ListMarketsRequest{Limit: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ListEventsPagination(ctx, entity.ListEventsPaginationRequest{Limit: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ListEventsKeyset(ctx, entity.ListEventsKeysetRequest{Limit: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.GetSportsMetadata(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := p.GetEventBySlug(ctx, entity.GetEventBySlugRequest{SlugName: "local-event"}); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := p.CreateAPIKey(ctx, strings.Repeat("1", 64), testFunder); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"POST /orders", "GET /data/order/order-1", "GET /data/trades", "GET /data/orders", "GET /fee-rate", "GET /book", "POST /books", "DELETE /orders", "POST /auth/api-key"} {
		if seen[key] == 0 {
			t.Errorf("missing CLOB request %q", key)
		}
	}
	for _, key := range []string{"GET /markets/market-id", "GET /markets", "GET /sports", "GET /events/pagination", "GET /events/keyset", "GET /events/slug/local-event"} {
		if seen[key] == 0 {
			t.Errorf("missing Gamma request %q", key)
		}
	}
}

func TestGetAccountInfoUsesConfiguredLocalRPC(t *testing.T) {
	rpc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.Method != "eth_call" {
			t.Errorf("rpc method=%s", req.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":"0x00000000000000000000000000000000000000000000000000000000000f4240"}`)
	}))
	defer rpc.Close()
	p := &Polymarket{client: apphttp.NewHttpClient(), orderBookURL: "local-clob", rpcURL: rpc.URL}
	info, err := p.GetAccountInfo(context.Background(), entity.PolymarketAccount{ProxyAddress: testFunder, EOAAddress: testFunder, SignatureType: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !info.Success || info.USDCBalance == "" || info.ChainID != 137 {
		t.Fatalf("unexpected account info: %#v", info)
	}
}
