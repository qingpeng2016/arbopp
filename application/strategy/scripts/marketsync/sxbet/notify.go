package sxbet

import (
	"context"
	"strconv"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"go.uber.org/zap"
)

func notifySxbetLeagueDone(ctx context.Context, leagueID int, meta LeagueMeta, fixtures, outcomes int) {
	league := meta.Name
	if league == "" {
		league = strconv.Itoa(leagueID)
	}
	logger.InfoZ(ctx, "market_sync-sxbet-league-done",
		zap.Int("league_id", leagueID),
		zap.String("league", league),
		zap.String("league_en", meta.NameDesc),
		zap.Int("fixtures", fixtures),
		zap.Int("outcomes", outcomes))
}