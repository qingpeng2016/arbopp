package predict

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws"
)

// Stream 订阅 predictOrderbook/{marketId}，收到全量盘口替换本地副本。
func Stream(ctx context.Context, apiKey, mapKey string, marketIDs []string, on func(Message) error) error {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("predict ws: missing api key")
	}
	hdr := http.Header{}
	hdr.Set("x-api-key", apiKey)
	raw, err := ws.Dial(ctx, PredictWSURL, hdr)
	if err != nil {
		return err
	}
	conn := ws.Wrap(raw)
	defer conn.Close()
	go ws.CloseOnDone(ctx, conn)
	params := make([]string, 0, len(marketIDs))
	for _, id := range marketIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		params = append(params, "predictOrderbook/"+id)
	}
	if len(params) == 0 {
		return fmt.Errorf("predict ws: no market ids")
	}
	// 服务端只读 params[0]，每个盘口必须单独 subscribe，否则只会订上第一个 market。
	for i, topic := range params {
		if err := conn.WriteJSON(map[string]any{
			"method":    "subscribe",
			"requestId": i + 1,
			"params":    []string{topic},
		}); err != nil {
			return err
		}
	}
	return conn.Read(ctx, func(msg []byte) error {
		kind, bookMsg, hb, err := Parse(msg)
		if err != nil {
			return err
		}
		if kind == "heartbeat" {
			if hb != 0 {
				if err := conn.WriteJSON(map[string]any{"method": "heartbeat", "data": hb}); err != nil {
					return err
				}
			}
			return on(Message{Kind: "heartbeat"})
		}
		if kind == "book" && bookMsg != nil {
			return on(*bookMsg)
		}
		return nil
	})
}
