package kalshi

import (
	"context"
	"net/http"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws"
)

// Stream 订阅 orderbook_delta。header 是已经签好的握手头。每 10 秒发 ping 帧。
func Stream(ctx context.Context, mapKey string, header http.Header, tickers []string, on func(Update) error) error {
	raw, err := ws.Dial(ctx, ws.KalshiURL, header)
	if err != nil {
		return err
	}
	conn := ws.Wrap(raw)
	defer conn.Close()
	go ws.CloseOnDone(ctx, conn)
	if err := conn.WriteJSON(map[string]any{
		"id":  1,
		"cmd": "subscribe",
		"params": map[string]any{
			"channels":       []string{"orderbook_delta"},
			"market_tickers": tickers,
		},
	}); err != nil {
		return err
	}
	go ws.PingControl(ctx, conn)
	return conn.Read(ctx, func(msg []byte) error {
		u, ok, err := Parse(msg)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		return on(u)
	})
}
