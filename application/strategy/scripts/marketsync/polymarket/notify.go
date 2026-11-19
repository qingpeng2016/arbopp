package polymarket

import (
	"context"
	"strconv"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	"go.uber.org/zap"
)

func sportNamesByID(cfg SportsConfig, subCat cryptoPredictionEnum.SubMarketCategory) map[string]string {
	items := cfg.BySubCategory[subCat]
	out := make(map[string]string, len(items))
	for _, it := range items {
		sid := strings.ToLower(strings.TrimSpace(it.SportID))
		if sid == "" {
			continue
		}
		if n := strings.TrimSpace(it.Name); n != "" {
			out[sid] = n
		}
	}
	return out
}

func sportNameDescByID(cfg SportsConfig, subCat cryptoPredictionEnum.SubMarketCategory) map[string]string {
	items := cfg.BySubCategory[subCat]
	out := make(map[string]string, len(items))
	for _, it := range items {
		sid := strings.ToLower(strings.TrimSpace(it.SportID))
		if sid == "" {
			continue
		}
		n := strings.TrimSpace(it.NameDesc)
		if n == "" {
			n = strings.TrimSpace(it.NameEn)
		}
		if n != "" {
			out[sid] = n
		}
	}
	return out
}

func leagueDisplayName(sportID string, nameBySport, compBySport map[string]string) string {
	sportID = strings.ToLower(strings.TrimSpace(sportID))
	if n := strings.TrimSpace(nameBySport[sportID]); n != "" {
		return n
	}
	if n := strings.TrimSpace(compBySport[sportID]); n != "" {
		return n
	}
	return sportID
}

func leagueDisplayNameEn(sportID string, nameEnBySport map[string]string, metaBySport map[string]httpentity.SportsMetadata) string {
	sportID = strings.ToLower(strings.TrimSpace(sportID))
	if n := strings.TrimSpace(nameEnBySport[sportID]); n != "" {
		return n
	}
	if metaBySport != nil {
		if n := strings.TrimSpace(metaBySport[sportID].Name); n != "" {
			return n
		}
	}
	return ""
}

func formatTagIDs(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}

func notifyPolymarketLeagueDone(ctx context.Context, sportID, leagueName, leagueNameEn string, tagIDs []int, fixtures, outcomes int) {
	logger.InfoZ(ctx, "market_sync-polymarket-league-done",
		zap.String("sport_id", sportID),
		zap.String("league", leagueName),
		zap.String("league_en", leagueNameEn),
		zap.Int("fixtures", fixtures),
		zap.Int("outcomes", outcomes),
		zap.String("tag_ids", formatTagIDs(tagIDs)))
	if outcomes == 0 {
		return
	}
	leagueEn := strings.TrimSpace(leagueNameEn)
	if leagueEn == "" {
		leagueEn = "-"
	}
	// TODO: 恢复成功 Lark
	// _ = notification.GetGlobalNotificationManager().SendInfoAlert(ctx, "market_sync 联赛完成", []notification.FieldPair{ ... })
	_ = leagueEn
}
