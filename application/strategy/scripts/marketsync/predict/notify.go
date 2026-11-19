package predict

import (
	"context"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"go.uber.org/zap"
)

func notifyPredictLeagueDone(ctx context.Context, item predictSportItem, fixtures, outcomes int) {
	league := strings.TrimSpace(item.Name)
	leagueEn := strings.TrimSpace(item.NameDesc)
	if leagueEn == "" {
		leagueEn = strings.TrimSpace(item.NameEn)
	}
	hasTags := predictItemHasTagIDs(item)
	if league == "" {
		if hasTags {
			league = "-"
		} else {
			league = "未配置 tag_ids"
		}
	}
	if leagueEn == "" {
		leagueEn = "-"
	}
	sportID := strings.TrimSpace(item.SportID)
	if sportID == "" && len(item.TagIDs) > 0 {
		sportID = item.TagIDs[0]
	}
	if sportID == "" {
		sportID = "-"
	}
	tagIDs := strings.Join(item.TagIDs, ",")
	if tagIDs == "" {
		tagIDs = "-"
	}
	logger.InfoZ(ctx, "market_sync-predict-league-done",
		zap.String("league", league),
		zap.String("tag_ids", tagIDs),
		zap.Int("fixtures", fixtures),
		zap.Int("outcomes", outcomes))
	// TODO: 恢复成功 Lark
	// if err := notification.GetGlobalNotificationManager().SendInfoAlert(ctx, "market_sync 联赛完成", []notification.FieldPair{
	// 	{Key: "平台", Value: appcommon.PlatformPredict},
	// 	{Key: "联赛", Value: league},
	// 	{Key: "联赛(英文)", Value: leagueEn},
	// 	{Key: "sport_id", Value: sportID},
	// 	{Key: "场次", Value: strconv.Itoa(fixtures)},
	// 	{Key: "盘口", Value: strconv.Itoa(outcomes)},
	// 	{Key: "tag_ids", Value: tagIDs},
	// }); err != nil {
	// 	logger.ErrorZ(ctx, "market_sync-predict-league-notify-failed", zap.Error(err))
	// }
	_, _, _ = leagueEn, sportID, tagIDs
}

// predictLeagueDoneStats 按配置联赛中文名聚合（sync_all_soccer 一次拉 Soccer 根 tag 时用）。
type predictLeagueDoneStats struct {
	fixtures map[string]struct{}
	outcomes int
}

func (s *predictLeagueDoneStats) addFixture(fid string, outcomeN int) {
	if s.fixtures == nil {
		s.fixtures = make(map[string]struct{})
	}
	if fid != "" {
		s.fixtures[fid] = struct{}{}
	}
	s.outcomes += outcomeN
}

func notifyPredictLeaguesAfterSyncRoot(ctx context.Context, subItems []predictSportItem, byCompetition map[string]*predictLeagueDoneStats) {
	if byCompetition == nil {
		byCompetition = map[string]*predictLeagueDoneStats{}
	}
	notified := make(map[string]bool, len(subItems))
	for _, it := range subItems {
		cn := strings.TrimSpace(it.Name)
		if cn == "" {
			continue
		}
		st := byCompetition[cn]
		fx, oc := 0, 0
		if st != nil {
			fx = len(st.fixtures)
			oc = st.outcomes
		}
		notifyPredictLeagueDone(ctx, it, fx, oc)
		notified[cn] = true
	}
	for cn, st := range byCompetition {
		if notified[cn] || st == nil {
			continue
		}
		notifyPredictLeagueDone(ctx, predictSportItem{Name: cn, NameEn: cn}, len(st.fixtures), st.outcomes)
	}
}