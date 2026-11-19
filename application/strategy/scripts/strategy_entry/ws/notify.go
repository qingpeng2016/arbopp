package ws

import (
	"context"
	"strconv"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
)

// NotifyReconnect 断线重连后新一轮订阅开始（gen>1）时发 Lark 信息通知。
func NotifyReconnect(ctx context.Context, platform, mapKey string, gen int) {
	if gen <= 1 || strings.TrimSpace(mapKey) == "" {
		return
	}
	_ = notification.GetGlobalNotificationManager().SendInfoAlert(ctx, "strategy_entry WS 重连", []notification.FieldPair{
		{Key: "平台", Value: platform},
		{Key: "map_key", Value: mapKey},
		{Key: "gen", Value: strconv.Itoa(gen)},
	})
}