package kalshi

import (
	"context"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"go.uber.org/zap"
)

func notifyKalshiLeagueDone(ctx context.Context, series string, meta kalshiLeagueMeta, fixtures, outcomes int) {
	league := strings.TrimSpace(meta.Name)
	if league == "" {
		league = series
	}
	leagueEn := strings.TrimSpace(meta.NameDesc)
	if leagueEn == "" {
		leagueEn = "-"
	}
	sportID := strings.TrimSpace(meta.PairPoly)
	if sportID == "" {
		sportID = "-"
	}
	logger.InfoZ(ctx, "market_sync-kalshi-league-done",
		zap.String("series", series),
		zap.String("league", league),
		zap.String("league_en", leagueEn),
		zap.String("sport_id", sportID),
		zap.Int("fixtures", fixtures),
		zap.Int("outcomes", outcomes))
	// TODO: 恢复成功 Lark
	// _ = notification.GetGlobalNotificationManager().SendInfoAlert(ctx, "market_sync 联赛完成", []notification.FieldPair{ ... })
}
