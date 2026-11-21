package polymarket

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws"
)

// Stream 订阅 market channel。应用层每 10s 文本 PING（Poly 协议）；同时 WS Ping 帧续 ReadDeadline（冷场无盘口推送也不断线）。
func Stream(ctx context.Context, mapKey string, assets []string, on func(Message) error) error {
	raw, err := ws.Dial(ctx, ws.PolyURL, nil)
	if err != nil {
		return err
	}
	conn := ws.Wrap(raw)
	defer conn.Close()
	go ws.CloseOnDone(ctx, conn)
	if err := conn.WriteJSON(map[string]any{
		"assets_ids":             assets,
		"type":                   "market",
		"custom_feature_enabled": true,
	}); err != nil {
		return err
	}
	go ws.PingText(ctx, conn)
	go ws.PingControl(ctx, conn)
	return conn.Read(ctx, func(msg []byte) error {
		msgs, err := Parse(msg)
		if err != nil {
			return err
		}
		for _, one := range msgs {
			if err := on(one); err != nil {
				return err
			}
		}
		return nil
	})
}
