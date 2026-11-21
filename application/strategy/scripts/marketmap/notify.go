package marketmap

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"go.uber.org/zap"
)

func notifyMarketMapLeagueDone(ctx context.Context, lg leagueMarkets, stats crossPlatformGroupStats, updated int) {
	leagueEn := lg.competitionEn
	if leagueEn == "" {
		leagueEn = "-"
	}
	logger.InfoZ(ctx, "market_map-league-done",
		zap.String("league", lg.competition),
		zap.String("league_en", leagueEn),
		zap.Int("polymarket", lg.poly),
		zap.Int("kalshi", lg.kalshi),
		zap.Int("predict", lg.predict),
		zap.Int("cross_platform_groups", stats.total()),
		zap.Int("updated", updated))
	// 仅本地日志；汇总 Lark 见 market_map.Run 末尾「market_map 完成」
	// _ = notification.GetGlobalNotificationManager().SendInfoAlert(ctx, "market_map 联赛完成", ...)
}
