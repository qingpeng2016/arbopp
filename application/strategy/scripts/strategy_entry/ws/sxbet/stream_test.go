//go:build integration

package sxbet

import (
	"context"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sxbethttp "github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http/crypto/prediction/sxbet"
)

// 需环境变量 SX_API_KEY；可选 SX_MARKET_HASH（默认示例 hash）。
//
//	SX_API_KEY=xxx go test -tags=integration -run TestStreamLive -v ./application/strategy/scripts/strategy_entry/ws/sxbet/ -count=1
func TestStreamLive(t *testing.T) {
	key := strings.TrimSpace(os.Getenv("SX_API_KEY"))
	if key == "" {
		t.Skip("SX_API_KEY not set")
	}
	hash := strings.TrimSpace(os.Getenv("SX_MARKET_HASH"))
	if hash == "" {
		hash = "0xbf06c6379c922d8118612d1d7493b20f6df6929437fbf6022904327f9516a2eb"
	}
	api := sxbethttp.NewClientWithOptions("", key, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	var n int64
	err := Stream(ctx, api, []string{hash}, 1, func(m Message) {
		switch m.Kind {
		case "book", "heartbeat":
			atomic.AddInt64(&n, 1)
		}
	})
	if err != nil && ctx.Err() == nil {
		t.Fatalf("stream: %v", err)
	}
	if n == 0 {
		t.Fatal("no book/heartbeat received")
	}
}
