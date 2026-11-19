package ws

import (
	"context"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	PolyURL   = "wss://ws-subscriptions-clob.polymarket.com/ws/market"
	KalshiURL = "wss://api.elections.kalshi.com/trade-api/ws/v2"
	// KalshiPath 参与 RSA-PSS 签名，必须和握手路径一致。
	KalshiPath = "/trade-api/ws/v2"
)

// StaggerConnect 各场次首次建连前随机等待 [0,2s]，错开同时握手。
func StaggerConnect(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	delay := time.Duration(rand.IntN(2001)) * time.Millisecond
	t := time.NewTimer(delay)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func Dial(ctx context.Context, rawURL string, header http.Header) (*websocket.Conn, error) {
	d := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := d.DialContext(ctx, rawURL, header)
	return conn, err
}

// Conn 保证同一条连接只有一个写者。
type Conn struct {
	c  *websocket.Conn
	mu sync.Mutex
}

func Wrap(c *websocket.Conn) *Conn {
	return &Conn{c: c}
}

func (c *Conn) WriteJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.c.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.c.WriteJSON(v)
}

func (c *Conn) WriteText(b []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.c.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.c.WriteMessage(websocket.TextMessage, b)
}

func (c *Conn) Ping() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.c.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
}

func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.c.Close()
}

func (c *Conn) Read(ctx context.Context, on func([]byte) error) error {
	c.c.SetReadLimit(8 << 20)
	_ = c.c.SetReadDeadline(time.Now().Add(45 * time.Second))
	c.c.SetPongHandler(func(string) error {
		return c.c.SetReadDeadline(time.Now().Add(45 * time.Second))
	})
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, msg, err := c.c.ReadMessage()
		if err != nil {
			return err
		}
		_ = c.c.SetReadDeadline(time.Now().Add(45 * time.Second))
		if len(msg) == 0 {
			continue
		}
		if err := on(msg); err != nil {
			return err
		}
	}
}

// Sleep 在 ctx 取消时返回 false。
func Sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func CloseOnDone(ctx context.Context, conn *Conn) {
	<-ctx.Done()
	_ = conn.Close()
}

func PingText(ctx context.Context, conn *Conn) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := conn.WriteText([]byte("PING")); err != nil {
				return
			}
		}
	}
}

func PingControl(ctx context.Context, conn *Conn) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := conn.Ping(); err != nil {
				return
			}
		}
	}
}
