package pmdata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client PMData V1 Parquet 下载。
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.pmdata.dev/v1"
	}
	return &Client{
		BaseURL: baseURL,
		APIKey:  strings.TrimSpace(apiKey),
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// DownloadL2 下载指定交易所单日 L2 Parquet；poly outcome 为 yes 时 query 为空。
func (c *Client) DownloadL2(ctx context.Context, exchange, dataType, symbol string, utcDay time.Time, polyOutcome string) ([]byte, int, error) {
	if c == nil {
		return nil, 0, fmt.Errorf("pmdata client nil")
	}
	exchange = strings.TrimSpace(exchange)
	symbol = strings.TrimSpace(symbol)
	if exchange == "" || symbol == "" {
		return nil, 0, fmt.Errorf("exchange/symbol required")
	}
	if dataType == "" {
		dataType = "l2"
	}
	y, m, d := utcDay.UTC().Date()
	path := fmt.Sprintf("/%s/%s/%04d/%02d/%02d/%s.parquet", exchange, dataType, y, int(m), d, symbol)
	reqURL := c.BaseURL + path
	q := url.Values{}
	if strings.EqualFold(exchange, "polymarket") && strings.EqualFold(strings.TrimSpace(polyOutcome), "no") {
		q.Set("outcome", "No")
	}
	if enc := q.Encode(); enc != "" {
		reqURL += "?" + enc
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, err
	}
	if c.APIKey != "" {
		req.Header.Set("api_key", c.APIKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, resp.StatusCode, fmt.Errorf("not found")
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, resp.StatusCode, fmt.Errorf("http %d: %s", resp.StatusCode, msg)
	}
	return body, resp.StatusCode, nil
}

func WriteLocalFile(storageDir, exchange, dataType string, utcDay time.Time, symbol string, data []byte) (path string, sha string, err error) {
	y, m, d := utcDay.UTC().Date()
	dir := filepath.Join(storageDir, exchange, dataType, fmt.Sprintf("%04d-%02d-%02d", y, m, d))
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	safe := strings.ReplaceAll(symbol, "/", "_")
	fname := safe + ".parquet"
	full := filepath.Join(dir, fname)
	if err = os.WriteFile(full, data, 0o644); err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(data)
	return full, hex.EncodeToString(sum[:]), nil
}