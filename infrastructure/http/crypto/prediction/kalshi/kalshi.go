package kalshi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	repo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL                = "https://api.elections.kalshi.com"
	apiPrefix                     = "/trade-api/v2"
	maxOrderbookTickersPerRequest = 100
	maxListPages                  = 100
	listPageLimit                 = 1000

	// 空闲连接池上限：Kalshi 客户端只打自家两三个 host，全局上限与单 host 取同一值即可
	maxIdleConns        = 512
	maxIdleConnsPerHost = 512
)

// newPooledTransport 复用连接的 Transport（Kalshi 专用一份）。
// 默认 Transport 只有 2 条/host 空闲连接，批量盘口/翻页并发时会频繁建连。
func newPooledTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          maxIdleConns,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// Signer 在账号没有带上私钥时，给测试注入一个签名桩。生产下单使用账号上的 RSA 私钥。
type Signer interface {
	Sign(ctx context.Context, keyID, timestamp, method, path string) (string, error)
}

// Client is a Kalshi V2 HTTP client. Public market endpoints do not require a
// signer; portfolio endpoints require an account and a configured Signer.
type Client struct {
	baseURL    string
	httpClient *http.Client
	signer     Signer
	batchGap   time.Duration
}

var _ repo.KalshiRepo = (*Client)(nil)

// NewPooledTransport 供 boot 按 platform.timeout 构造 http.Client。
func NewPooledTransport() *http.Transport {
	return newPooledTransport()
}

// NewClientWithOptions is intended for tests and local integration tools.
func NewClientWithOptions(baseURL string, client *http.Client, signer Signer) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second, Transport: newPooledTransport()}
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), httpClient: client, signer: signer, batchGap: 200 * time.Millisecond}
}

// NewClientWithBatchGap allows tests to remove the production batch delay.
func NewClientWithBatchGap(baseURL string, client *http.Client, signer Signer, gap time.Duration) *Client {
	c := NewClientWithOptions(baseURL, client, signer)
	c.batchGap = gap
	return c
}

func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values, body any, account *entity.KalshiAccount, out any) error {
	fullURL := c.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if account != nil {
		key, ts, sig, err := c.auth(ctx, *account, method, path)
		if err != nil {
			return err
		}
		req.Header.Set("KALSHI-ACCESS-KEY", key)
		req.Header.Set("KALSHI-ACCESS-TIMESTAMP", ts)
		req.Header.Set("KALSHI-ACCESS-SIGNATURE", sig)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read Kalshi response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Kalshi %s %s failed: status=%d body=%s", method, path, resp.StatusCode, string(raw))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("decode Kalshi response: %w", err)
		}
	}
	return nil
}

func (c *Client) GetMarket(ctx context.Context, ticker string) (*entity.KalshiMarket, error) {
	if ticker == "" {
		return nil, fmt.Errorf("ticker is required")
	}
	var out entity.KalshiGetMarketResponse
	if err := c.doRequest(ctx, http.MethodGet, apiPrefix+"/markets/"+url.PathEscape(ticker), nil, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out.Market, nil
}

func (c *Client) GetMarkets(ctx context.Context, params repo.KalshiListParams) (*entity.KalshiGetMarketsResponse, error) {
	q := url.Values{}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Cursor != "" {
		q.Set("cursor", params.Cursor)
	}
	if params.Ticker != "" {
		q.Set("event_ticker", params.Ticker)
	}
	if params.Status != "" {
		q.Set("status", params.Status)
	}
	if params.MveFilter != "" {
		q.Set("mve_filter", params.MveFilter)
	}
	if params.SeriesTicker != "" {
		q.Set("series_ticker", params.SeriesTicker)
	}
	var out entity.KalshiGetMarketsResponse
	if err := c.doRequest(ctx, http.MethodGet, apiPrefix+"/markets", q, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetEvents(ctx context.Context, params repo.KalshiListParams) (*entity.KalshiGetEventsResponse, error) {
	q := url.Values{}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Cursor != "" {
		q.Set("cursor", params.Cursor)
	}
	if params.Status != "" {
		q.Set("status", params.Status)
	}
	if params.Category != "" {
		q.Set("category", params.Category)
	}
	if params.SeriesTicker != "" {
		q.Set("series_ticker", params.SeriesTicker)
	}
	if params.WithNestedMarkets {
		q.Set("with_nested_markets", "true")
	}
	var out entity.KalshiGetEventsResponse
	if err := c.doRequest(ctx, http.MethodGet, apiPrefix+"/events", q, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetEvent(ctx context.Context, eventTicker string) (*entity.KalshiGetEventResponse, error) {
	if eventTicker == "" {
		return nil, fmt.Errorf("event_ticker is required")
	}
	q := url.Values{"with_nested_markets": []string{"true"}}
	var out entity.KalshiGetEventResponse
	if err := c.doRequest(ctx, http.MethodGet, apiPrefix+"/events/"+url.PathEscape(eventTicker), q, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetOrderbook(ctx context.Context, ticker string, depth int) (*entity.KalshiOrderbook, error) {
	if ticker == "" {
		return nil, fmt.Errorf("ticker is required")
	}
	q := url.Values{}
	if depth > 0 {
		q.Set("depth", strconv.Itoa(depth))
	}
	var out entity.KalshiGetOrderbookResponse
	if err := c.doRequest(ctx, http.MethodGet, apiPrefix+"/markets/"+url.PathEscape(ticker)+"/orderbook", q, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out.OrderbookFp, nil
}

func (c *Client) GetOrderbooks(ctx context.Context, tickers []string) (map[string]*entity.KalshiOrderbook, error) {
	out := make(map[string]*entity.KalshiOrderbook, len(tickers))
	for start := 0; start < len(tickers); start += maxOrderbookTickersPerRequest {
		end := min(start+maxOrderbookTickersPerRequest, len(tickers))
		q := url.Values{}
		for _, ticker := range tickers[start:end] {
			if ticker != "" {
				q.Add("tickers", ticker)
			}
		}
		if len(q) > 0 {
			var batch entity.KalshiGetOrderbooksResponse
			if err := c.doRequest(ctx, http.MethodGet, apiPrefix+"/markets/orderbooks", q, nil, nil, &batch); err != nil {
				return nil, err
			}
			for i := range batch.Orderbooks {
				item := batch.Orderbooks[i]
				if item.Ticker != "" {
					out[item.Ticker] = &item.OrderbookFp
				}
			}
		}
		if end < len(tickers) && c.batchGap > 0 {
			t := time.NewTimer(c.batchGap)
			select {
			case <-ctx.Done():
				t.Stop()
				return nil, ctx.Err()
			case <-t.C:
			}
		}
	}
	return out, nil
}

// SignHeaders 为 WebSocket 握手或自定义路径签名。path 必须与交易所验签路径一致。
func (c *Client) SignHeaders(ctx context.Context, account entity.KalshiAccount, method, path string) (string, string, string, error) {
	return c.auth(ctx, account, method, path)
}

// auth 优先用账号上的私钥本地签名。没有私钥时才走注入的 Signer，现有单测仍走这条。
func (c *Client) auth(ctx context.Context, account entity.KalshiAccount, method, path string) (string, string, string, error) {
	keyID := strings.TrimSpace(account.KeyID)
	if keyID == "" {
		return "", "", "", fmt.Errorf("kalshi api_key is required")
	}
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	if strings.TrimSpace(account.PrivateKey) != "" {
		sig, err := signKalshiPSS(account.PrivateKey, account.Passphrase, ts+method+path)
		if err != nil {
			return "", "", "", err
		}
		return keyID, ts, sig, nil
	}
	if c.signer == nil {
		return "", "", "", fmt.Errorf("kalshi api_secret is required")
	}
	sig, err := c.signer.Sign(ctx, keyID, ts, method, path)
	if err != nil {
		return "", "", "", fmt.Errorf("sign Kalshi request: %w", err)
	}
	return keyID, ts, sig, nil
}

func (c *Client) PlaceOrder(ctx context.Context, req entity.KalshiPlaceOrderReq, account entity.KalshiAccount) (*entity.KalshiOrder, error) {
	var out entity.KalshiCreateOrderResponse
	if err := c.doRequest(ctx, http.MethodPost, apiPrefix+"/portfolio/events/orders", nil, req, &account, &out); err != nil {
		return nil, err
	}
	return &entity.KalshiOrder{OrderID: out.OrderID, ClientOrderID: out.ClientOrderID, Ticker: req.Ticker, BookSide: req.Side, Type: "limit", InitialCountFp: req.Count, FillCountFp: out.FillCount, RemainingCntFp: out.RemainingCount}, nil
}

func (c *Client) CancelOrder(ctx context.Context, orderID string, account entity.KalshiAccount) (*entity.KalshiOrder, error) {
	if orderID == "" {
		return nil, fmt.Errorf("order_id is required")
	}
	var out entity.KalshiCancelOrderResponse
	if err := c.doRequest(ctx, http.MethodDelete, apiPrefix+"/portfolio/events/orders/"+url.PathEscape(orderID), nil, nil, &account, &out); err != nil {
		return nil, err
	}
	id := out.OrderID
	if id == "" {
		id = orderID
	}
	return &entity.KalshiOrder{OrderID: id, ClientOrderID: out.ClientOrderID, Status: "canceled", RemainingCntFp: "0"}, nil
}

func (c *Client) AmendOrder(ctx context.Context, orderID string, req entity.KalshiAmendOrderReq, account entity.KalshiAccount) (*entity.KalshiOrder, error) {
	if orderID == "" {
		return nil, fmt.Errorf("order_id is required")
	}
	var out entity.KalshiCreateOrderResponse
	if err := c.doRequest(ctx, http.MethodPost, apiPrefix+"/portfolio/events/orders/"+url.PathEscape(orderID)+"/amend", nil, req, &account, &out); err != nil {
		return nil, err
	}
	id := out.OrderID
	if id == "" {
		id = orderID
	}
	return &entity.KalshiOrder{OrderID: id, ClientOrderID: out.ClientOrderID, Ticker: req.Ticker, BookSide: req.Side, Type: "limit", InitialCountFp: req.Count, FillCountFp: out.FillCount, RemainingCntFp: out.RemainingCount}, nil
}

func (c *Client) GetOrder(ctx context.Context, orderID string, account entity.KalshiAccount) (*entity.KalshiOrder, error) {
	if orderID == "" {
		return nil, fmt.Errorf("order_id is required")
	}
	var out entity.KalshiOrderResponse
	if err := c.doRequest(ctx, http.MethodGet, apiPrefix+"/portfolio/orders/"+url.PathEscape(orderID), nil, nil, &account, &out); err != nil {
		return nil, err
	}
	return &out.Order, nil
}

func (c *Client) GetOrders(ctx context.Context, params repo.KalshiListParams, account entity.KalshiAccount) (*entity.KalshiGetOrdersResponse, error) {
	var out entity.KalshiGetOrdersResponse
	if err := c.doRequest(ctx, http.MethodGet, apiPrefix+"/portfolio/orders", buildListQuery(params), nil, &account, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPositions(ctx context.Context, params repo.KalshiListParams, account entity.KalshiAccount) (*entity.KalshiGetPositionsResponse, error) {
	if params.Limit > 0 {
		var out entity.KalshiGetPositionsResponse
		if err := c.doRequest(ctx, http.MethodGet, apiPrefix+"/portfolio/positions", buildListQuery(params), nil, &account, &out); err != nil {
			return nil, err
		}
		return &out, nil
	}
	page := params
	page.Limit = listPageLimit
	var all []entity.KalshiMarketPosition
	seen := make(map[string]bool)
	for i := 0; i < maxListPages; i++ {
		var out entity.KalshiGetPositionsResponse
		if err := c.doRequest(ctx, http.MethodGet, apiPrefix+"/portfolio/positions", buildListQuery(page), nil, &account, &out); err != nil {
			return nil, err
		}
		all = append(all, out.MarketPositions...)
		if out.Cursor == "" || seen[out.Cursor] {
			break
		}
		seen[out.Cursor] = true
		page.Cursor = out.Cursor
	}
	return &entity.KalshiGetPositionsResponse{MarketPositions: all}, nil
}

func (c *Client) GetFills(ctx context.Context, params repo.KalshiListParams, account entity.KalshiAccount) (*entity.KalshiGetFillsResponse, error) {
	if params.Limit > 0 {
		var out entity.KalshiGetFillsResponse
		if err := c.doRequest(ctx, http.MethodGet, apiPrefix+"/portfolio/fills", fillsQuery(params), nil, &account, &out); err != nil {
			return nil, err
		}
		return &out, nil
	}
	page := params
	page.Limit = listPageLimit
	var all []entity.KalshiFill
	seen := make(map[string]bool)
	for i := 0; i < maxListPages; i++ {
		var out entity.KalshiGetFillsResponse
		if err := c.doRequest(ctx, http.MethodGet, apiPrefix+"/portfolio/fills", fillsQuery(page), nil, &account, &out); err != nil {
			return nil, err
		}
		all = append(all, out.Fills...)
		if out.Cursor == "" || seen[out.Cursor] {
			break
		}
		seen[out.Cursor] = true
		page.Cursor = out.Cursor
	}
	return &entity.KalshiGetFillsResponse{Fills: all}, nil
}

func (c *Client) GetBalance(ctx context.Context, account entity.KalshiAccount) (*entity.KalshiBalance, error) {
	var out entity.KalshiBalance
	if err := c.doRequest(ctx, http.MethodGet, apiPrefix+"/portfolio/balance", nil, nil, &account, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func buildListQuery(p repo.KalshiListParams) url.Values {
	q := url.Values{}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Cursor != "" {
		q.Set("cursor", p.Cursor)
	}
	if p.Ticker != "" {
		q.Set("ticker", p.Ticker)
	}
	if p.Status != "" {
		q.Set("status", p.Status)
	}
	return q
}
func fillsQuery(p repo.KalshiListParams) url.Values {
	q := buildListQuery(p)
	if p.OrderID != "" {
		q.Set("order_id", p.OrderID)
	}
	return q
}
