package sxbet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	repo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
)

const (
	defaultBaseURL = "https://api.sx.bet"
	apiKeyHeader   = "x-sx-api-key"
)

func newPooledTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          64,
		MaxIdleConnsPerHost:   64,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// Client SX Bet HTTP 客户端。
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

var _ repo.SxbetRepo = (*Client)(nil)

// NewPooledTransport 供 boot 构造 http.Client。
func NewPooledTransport() *http.Transport {
	return newPooledTransport()
}

// NewClientWithOptions 生产与测试入口。apiKey 可选，仅公开市场数据可不传。
func NewClientWithOptions(baseURL, apiKey string, client *http.Client) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second, Transport: newPooledTransport()}
	}
	return &Client{baseURL: baseURL, apiKey: strings.TrimSpace(apiKey), httpClient: client}
}

func (c *Client) doGET(ctx context.Context, path string, query url.Values, out any) error {
	if c == nil {
		return fmt.Errorf("sxbet client is nil")
	}
	fullURL := c.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set(apiKeyHeader, c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sxbet GET %s: http %d: %s", path, resp.StatusCode, truncateErrBody(body))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("sxbet decode %s: %w (body %s)", path, err, truncateErrBody(body))
	}
	return nil
}

func truncateErrBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 512 {
		return s[:512] + "..."
	}
	return s
}

func (c *Client) GetActiveMarkets(ctx context.Context, params repo.SxbetListActiveMarketsParams) (*entity.SxbetActiveMarketsResponse, error) {
	q := url.Values{}
	if params.LeagueID > 0 {
		q.Set("leagueId", strconv.Itoa(params.LeagueID))
	}
	if e := strings.TrimSpace(params.EventID); e != "" {
		q.Set("eventId", e)
	}
	if len(params.SportIDs) > 0 {
		parts := make([]string, len(params.SportIDs))
		for i, id := range params.SportIDs {
			parts[i] = strconv.Itoa(id)
		}
		q.Set("sportIds", strings.Join(parts, ","))
	}
	if len(params.Types) > 0 {
		parts := make([]string, len(params.Types))
		for i, id := range params.Types {
			parts[i] = strconv.Itoa(id)
		}
		q.Set("type", strings.Join(parts, ","))
	}
	if params.OnlyMainLine {
		q.Set("onlyMainLine", "true")
	}
	if params.LiveOnly {
		q.Set("liveOnly", "true")
	}
	if params.GameTimeAfter > 0 {
		q.Set("gameTime", strconv.FormatInt(params.GameTimeAfter, 10))
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}
	if pageSize > 100 {
		pageSize = 100
	}
	q.Set("pageSize", strconv.Itoa(pageSize))
	if k := strings.TrimSpace(params.PaginationKey); k != "" {
		q.Set("paginationKey", k)
	}
	var resp entity.SxbetActiveMarketsResponse
	if err := c.doGET(ctx, "/markets/active", q, &resp); err != nil {
		return nil, err
	}
	if strings.EqualFold(resp.Status, "failure") {
		return nil, fmt.Errorf("sxbet markets/active: status failure")
	}
	return &resp, nil
}

func (c *Client) GetActiveLeagues(ctx context.Context) ([]entity.SxbetLeague, error) {
	var resp entity.SxbetLeaguesResponse
	if err := c.doGET(ctx, "/leagues/active", nil, &resp); err != nil {
		return nil, err
	}
	if strings.EqualFold(resp.Status, "failure") {
		return nil, fmt.Errorf("sxbet leagues/active: status failure")
	}
	return resp.Data, nil
}

func (c *Client) GetSports(ctx context.Context) ([]entity.SxbetSport, error) {
	var resp entity.SxbetSportsResponse
	if err := c.doGET(ctx, "/sports", nil, &resp); err != nil {
		return nil, err
	}
	if strings.EqualFold(resp.Status, "failure") {
		return nil, fmt.Errorf("sxbet sports: status failure")
	}
	return resp.Data, nil
}

// APIKey 配置在 platform.extra 的 API Key（realtime / 盘口快照）。
func (c *Client) APIKey() string {
	if c == nil {
		return ""
	}
	return c.apiKey
}

// GetRealtimeToken GET /user/realtime-token-v3/api-key（Centrifugo 连接用）。
func (c *Client) GetRealtimeToken(ctx context.Context) (string, error) {
	if c == nil {
		return "", fmt.Errorf("sxbet client is nil")
	}
	if strings.TrimSpace(c.apiKey) == "" {
		return "", fmt.Errorf("sxbet: missing api key for realtime")
	}
	var resp entity.SxbetRealtimeTokenResponse
	if err := c.doGET(ctx, "/user/realtime-token-v3/api-key", nil, &resp); err != nil {
		return "", err
	}
	tok := strings.TrimSpace(resp.Token)
	if tok == "" {
		return "", fmt.Errorf("sxbet realtime token empty (status=%s)", resp.Status)
	}
	return tok, nil
}

// GetOrderbookSnapshot GET /orderbook-v3/snapshot/market。
func (c *Client) GetOrderbookSnapshot(ctx context.Context, marketHash string) (*entity.SxbetOrderbookV3, error) {
	marketHash = strings.TrimSpace(marketHash)
	if marketHash == "" {
		return nil, fmt.Errorf("sxbet snapshot: empty marketHash")
	}
	q := url.Values{}
	q.Set("marketHash", marketHash)
	var resp entity.SxbetOrderbookSnapshotResponse
	if err := c.doGET(ctx, "/orderbook-v3/snapshot/market", q, &resp); err != nil {
		return nil, err
	}
	if strings.EqualFold(resp.Status, "failure") {
		return nil, fmt.Errorf("sxbet orderbook snapshot: status failure")
	}
	out := resp.Data
	if strings.TrimSpace(out.MarketHash) == "" {
		out.MarketHash = marketHash
	}
	return &out, nil
}