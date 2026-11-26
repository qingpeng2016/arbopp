package predict

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	repo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
)

const (
	defaultBaseURL          = "https://api.predict.fun/v1"
	apiKeyHeader            = "x-api-key"
	maxOrderbooksPerRequest = 100
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

// Client Predict.fun REST。
type Client struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	jwtMu       sync.Mutex
	jwtByKey    map[string]jwtSlot
	marketCache sync.Map // marketID -> *entity.PredictMarket
}

var _ repo.PredictRepo = (*Client)(nil)

// NewPooledTransport 供 boot 构造 http.Client。
func NewPooledTransport() *http.Transport {
	return newPooledTransport()
}

// NewClientWithOptions 生产与测试入口。
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
	return c.doGETWithKey(ctx, c.apiKey, path, query, out)
}

func (c *Client) doGETWithKey(ctx context.Context, apiKey, path string, query url.Values, out any) error {
	return c.doGETAuth(ctx, apiKey, "", path, query, out)
}

func (c *Client) doGETAuth(ctx context.Context, apiKey, bearer, path string, query url.Values, out any) error {
	if c == nil {
		return fmt.Errorf("predict client is nil")
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(c.apiKey)
	}
	if apiKey == "" {
		return fmt.Errorf("predict api_key is required")
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
	req.Header.Set(apiKeyHeader, apiKey)
	if b := strings.TrimSpace(bearer); b != "" {
		req.Header.Set("Authorization", "Bearer "+b)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("predict GET %s: http %d: %s", path, resp.StatusCode, truncateErrBody(body))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("predict decode %s: %w", path, err)
	}
	return nil
}

func (c *Client) doPOSTWithKey(ctx context.Context, apiKey, path string, body any, bearer string, out any) error {
	if c == nil {
		return fmt.Errorf("predict client is nil")
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(c.apiKey)
	}
	if apiKey == "" {
		return fmt.Errorf("predict api_key is required")
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(apiKeyHeader, apiKey)
	if bearer := strings.TrimSpace(bearer); bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("predict POST %s: http %d: %s", path, resp.StatusCode, truncateErrBody(respBody))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("predict decode POST %s: %w", path, err)
	}
	return nil
}

func truncateErrBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 400 {
		return s[:400] + "..."
	}
	return s
}

// GetCategories 分页拉 category（含 markets）。
func (c *Client) GetCategories(ctx context.Context, params repo.PredictListCategoriesParams) (*entity.PredictGetCategoriesResponse, error) {
	q := url.Values{}
	if params.First > 0 {
		q.Set("first", strconv.Itoa(params.First))
	}
	if after := strings.TrimSpace(params.After); after != "" {
		q.Set("after", after)
	}
	if st := strings.TrimSpace(params.Status); st != "" {
		q.Set("status", st)
	}
	if ids := joinTagIDs(params.TagIDs); ids != "" {
		q.Set("tagIds", ids)
	}
	if v := joinMarketVariants(params.MarketVariants); v != "" {
		q.Set("marketVariants", v)
	}
	var out entity.PredictGetCategoriesResponse
	if err := c.doGET(ctx, "/categories", q, &out); err != nil {
		return nil, err
	}
	if !out.Success {
		msg := strings.TrimSpace(out.Message)
		if msg == "" {
			msg = strings.TrimSpace(out.Error)
		}
		if msg == "" {
			msg = "success=false"
		}
		return nil, fmt.Errorf("predict categories: %s", msg)
	}
	return &out, nil
}

// GetOrderbooks GET /markets/orderbooks?ids=… 最多 100 个 id。
func (c *Client) GetOrderbooks(ctx context.Context, marketIDs []string) ([]entity.PredictOrderbook, error) {
	ids := uniqueNonEmpty(marketIDs)
	if len(ids) == 0 {
		return nil, nil
	}
	var out []entity.PredictOrderbook
	for start := 0; start < len(ids); start += maxOrderbooksPerRequest {
		end := start + maxOrderbooksPerRequest
		if end > len(ids) {
			end = len(ids)
		}
		q := url.Values{}
		for _, id := range ids[start:end] {
			q.Add("ids", id)
		}
		var resp entity.PredictGetOrderbooksResponse
		if err := c.doGET(ctx, "/markets/orderbooks", q, &resp); err != nil {
			return nil, err
		}
		if !resp.Success {
			msg := strings.TrimSpace(resp.Message)
			if msg == "" {
				msg = strings.TrimSpace(resp.Error)
			}
			if msg == "" {
				msg = "success=false"
			}
			return nil, fmt.Errorf("predict orderbooks: %s", msg)
		}
		out = append(out, resp.Data...)
	}
	return out, nil
}

func uniqueNonEmpty(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func joinTagIDs(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	var parts []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			parts = append(parts, id)
		}
	}
	return strings.Join(parts, ",")
}

func joinMarketVariants(vs []string) string {
	if len(vs) == 0 {
		return ""
	}
	var parts []string
	for _, v := range vs {
		v = strings.TrimSpace(v)
		if v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, ",")
}
