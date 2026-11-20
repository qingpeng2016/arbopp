package marketmap

import (
	"context"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"go.uber.org/zap"
)

type mapMember struct {
	id       uint
	platform string
}

// MarketMap 按 category+sub_category+competition_en 归桶，在同桶内跨 Poly / Kalshi / Predict 对齐 slug，写 map_key 并置 map_verified=1。
type MarketMap struct {
	marketRepo  persistentrepo.MarketRepo
	outcomeRepo persistentrepo.MarketOutcomeRepo
}

func NewMarketMap(marketRepo persistentrepo.MarketRepo, outcomeRepo persistentrepo.MarketOutcomeRepo) *MarketMap {
	return &MarketMap{marketRepo: marketRepo, outcomeRepo: outcomeRepo}
}

// Run 发现仍有 map_verified=0 的联赛后，拉取该联赛下全部 ACTIVE 场次做跨平台对齐（含已核对 Poly/Kalshi），
// 同组内统一 map_key；未核对行另置 map_verified=1。
func (j *MarketMap) Run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			notification.SendErrorLog(ctx, "market_map-Panic",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())))
		}
	}()
	if j.marketRepo == nil {
		notification.SendErrorLog(ctx, "market_map-no-repo")
		return
	}

	_, pending, err := j.marketRepo.List(ctx, nil, map[string]interface{}{
		"status = ?":       string(cryptoPredictionEnum.MarketStatusActive),
		"map_verified = ?": 0,
	}, "competition asc, id asc", "")
	if err != nil {
		notification.SendErrorLog(ctx, "market_map-ListFailed", zap.Error(err))
		return
	}
	if len(pending) == 0 {
		logger.InfoZ(ctx, "market_map-Completed", zap.Int("scanned", 0))
		return
	}
	scopes := distinctMapMatchScopes(pending)
	rows, err := j.listActiveMarketsInScopes(ctx, scopes)
	if err != nil {
		notification.SendErrorLog(ctx, "market_map-ListLeagueFailed", zap.Error(err))
		return
	}

	leagues := groupMarketsByLeague(rows)
	var updated, skippedVerified, crossPlatformGroups int
	for _, lg := range leagues {
		groups, leagueUpdated, leagueSkipped := j.mapLeague(ctx, rows, lg)
		crossPlatformGroups += groups
		updated += leagueUpdated
		skippedVerified += leagueSkipped
		notifyMarketMapLeagueDone(ctx, lg, groups, leagueUpdated)
	}

	logger.InfoZ(ctx, "market_map-Completed",
		zap.Int("pending", len(pending)),
		zap.Int("league_rows", len(rows)),
		zap.Int("cross_platform_groups", crossPlatformGroups),
		zap.Int("updated", updated),
		zap.Int("skipped_verified", skippedVerified))

	_ = notification.GetGlobalNotificationManager().SendInfoAlert(ctx, "market_map 完成", []notification.FieldPair{
		{Key: "待核对", Value: strconv.Itoa(len(pending))},
		{Key: "联赛行数", Value: strconv.Itoa(len(rows))},
		{Key: "跨平台组", Value: strconv.Itoa(crossPlatformGroups)},
		{Key: "更新 map_key", Value: strconv.Itoa(updated)},
	})
}

type mapMatchScope struct {
	category      string
	subCategory   string
	competitionEn string
}

type leagueMarkets struct {
	category      string
	subCategory   string
	competition   string
	competitionEn string
	scopeKey      string
	idxs          []int
	keys          map[string]struct{}
	poly          int
	kalshi        int
	predict       int
}

func groupMarketsByLeague(rows []persistententity.Market) []leagueMarkets {
	byKey := make(map[string]*leagueMarkets)
	var order []string
	for i := range rows {
		comp, compEn, key, ok := platformFixtureKeyFromMarket(&rows[i])
		if !ok {
			continue
		}
		cat, sub := marketCategorySub(&rows[i])
		bucket := leagueBucketKey(cat, sub, compEn)
		if bucket == "" {
			continue
		}
		lg := byKey[bucket]
		if lg == nil {
			lg = &leagueMarkets{
				category:      cat,
				subCategory:   sub,
				competitionEn: compEn,
				scopeKey:      bucket,
				keys:          make(map[string]struct{}),
			}
			byKey[bucket] = lg
			order = append(order, bucket)
		}
		lg.idxs = append(lg.idxs, i)
		lg.keys[key] = struct{}{}
		plat := strings.ToUpper(strings.TrimSpace(rows[i].Platform))
		switch plat {
		case appcommon.PlatformPolymarket:
			lg.poly++
			lg.competition = comp
		case appcommon.PlatformPredict:
			lg.predict++
			if lg.competition == "" {
				lg.competition = comp
			}
		case appcommon.PlatformKalshi:
			lg.kalshi++
			if lg.competition == "" {
				lg.competition = comp
			}
		default:
			if lg.competition == "" {
				lg.competition = comp
			}
		}
	}
	sort.Slice(order, func(i, j int) bool {
		return byKey[order[i]].competitionEn < byKey[order[j]].competitionEn
	})
	out := make([]leagueMarkets, 0, len(order))
	for _, bucket := range order {
		out = append(out, *byKey[bucket])
	}
	return out
}

func (j *MarketMap) mapLeague(ctx context.Context, rows []persistententity.Market, lg leagueMarkets) (groups, updated, skippedVerified int) {
	names := make(map[string]string, len(lg.idxs))
	starts := make(map[string]string, len(lg.idxs))
	for _, idx := range lg.idxs {
		m := &rows[idx]
		_, _, key, ok := platformFixtureKeyFromMarket(m)
		if !ok {
			continue
		}
		if m.Name != nil {
			names[key] = strings.TrimSpace(*m.Name)
		}
		if m.EventStartTime != nil {
			if s := strings.TrimSpace(*m.EventStartTime); s != "" {
				starts[key] = s
			}
		}
	}
	linked := linkPlatformFixtures(lg.keys, names, starts)
	grouped := make(map[fixtureGroupKey][]mapMember)
	for _, idx := range lg.idxs {
		m := &rows[idx]
		_, _, pfKey, ok := platformFixtureKeyFromMarket(m)
		if !ok {
			continue
		}
		cf := linked[pfKey]
		if cf == "" {
			cf = canonicalFixtureID(pfKey)
		}
		comp := lg.competition
		if comp == "" && m.Competition != nil {
			comp = strings.TrimSpace(*m.Competition)
		}
		gk := fixtureGroupKey{competition: comp, competitionEn: lg.competitionEn, fixtureID: cf}
		grouped[gk] = append(grouped[gk], mapMember{
			id:       m.ID,
			platform: strings.ToUpper(strings.TrimSpace(m.Platform)),
		})
	}
	for gk, members := range grouped {
		members, dropped := pickOneMarketPerPlatform(rows, members, gk.fixtureID)
		for _, d := range dropped {
			if err := j.clearMapAssignment(ctx, d.id); err != nil {
				notification.SendErrorLog(ctx, "market_map-ClearDupPlatformFailed",
					zap.Uint("id", d.id), zap.String("platform", d.platform), zap.Error(err))
			}
		}
		if len(distinctPlatforms(members)) < 2 {
			continue
		}
		mapKey := buildFixtureMapKey(gk.competition, gk.fixtureID)
		if mapKey == "" {
			continue
		}
		groups++
		for _, mem := range members {
			row := findMarket(rows, mem.id)
			if row == nil {
				continue
			}
			needsKey, needsVerify := marketRowNeedsUpdate(row, mapKey)
			if !needsKey && !needsVerify {
				if row.MapVerified != 0 {
					skippedVerified++
				}
				continue
			}
			if err := j.applyMapKeyVerified(ctx, mem.id, row, mapKey); err != nil {
				notification.SendErrorLog(ctx, "market_map-UpdateFailed",
					zap.Uint("id", mem.id), zap.String("map_key", mapKey), zap.Error(err))
				continue
			}
			if needsKey || needsVerify {
				updated++
			}
		}
	}
	return groups, updated, skippedVerified
}

// marketRowNeedsUpdate 返回 (需改 map_key, 需置 map_verified)。
func marketRowNeedsUpdate(row *persistententity.Market, mapKey string) (needsKey, needsVerify bool) {
	if row == nil {
		return false, false
	}
	keyOK := row.MapKey != nil && strings.TrimSpace(*row.MapKey) == mapKey
	if !keyOK {
		needsKey = true
	}
	if row.MapVerified == 0 {
		needsVerify = true
	}
	return needsKey, needsVerify
}

func (j *MarketMap) applyMapKeyVerified(ctx context.Context, id uint, row *persistententity.Market, mapKey string) error {
	if row == nil {
		return nil
	}
	needsKey, needsVerify := marketRowNeedsUpdate(row, mapKey)
	if !needsKey && !needsVerify {
		return nil
	}
	upd := map[string]interface{}{}
	if needsKey {
		upd["map_key"] = mapKey
	}
	if needsVerify {
		upd["map_verified"] = 1
	}
	if err := j.marketRepo.Update(ctx, nil, map[string]interface{}{"id = ?": id}, upd); err != nil {
		return err
	}
	if strings.TrimSpace(mapKey) != "" {
		return j.propagateOutcomeMapKey(ctx, id, mapKey)
	}
	return nil
}

// propagateOutcomeMapKey 将该场次 map_key 写到其下全部 market_outcome（与 market 场次键一致）。
func (j *MarketMap) propagateOutcomeMapKey(ctx context.Context, marketRowID uint, mapKey string) error {
	if j == nil || j.outcomeRepo == nil || marketRowID == 0 {
		return nil
	}
	mapKey = strings.TrimSpace(mapKey)
	if mapKey == "" {
		return nil
	}
	return j.outcomeRepo.Update(ctx, nil, map[string]interface{}{
		"market_row_id = ?": marketRowID,
	}, map[string]interface{}{
		"map_key": mapKey,
	})
}

func marketCategorySub(m *persistententity.Market) (category, subCategory string) {
	if m == nil {
		return "", ""
	}
	if m.Category != nil {
		category = strings.TrimSpace(*m.Category)
	}
	if m.SubCategory != nil {
		subCategory = strings.TrimSpace(*m.SubCategory)
	}
	return category, subCategory
}

func distinctMapMatchScopes(rows []persistententity.Market) []mapMatchScope {
	seen := make(map[string]struct{})
	var out []mapMatchScope
	for i := range rows {
		cat, sub := marketCategorySub(&rows[i])
		if rows[i].CompetitionEn == nil {
			continue
		}
		en := strings.TrimSpace(*rows[i].CompetitionEn)
		key := leagueBucketKey(cat, sub, en)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, mapMatchScope{category: cat, subCategory: sub, competitionEn: en})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.category != b.category {
			return a.category < b.category
		}
		if a.subCategory != b.subCategory {
			return a.subCategory < b.subCategory
		}
		return a.competitionEn < b.competitionEn
	})
	return out
}

func (j *MarketMap) listActiveMarketsInScopes(ctx context.Context, scopes []mapMatchScope) ([]persistententity.Market, error) {
	if len(scopes) == 0 {
		return nil, nil
	}
	active := string(cryptoPredictionEnum.MarketStatusActive)
	var rows []persistententity.Market
	for _, sc := range scopes {
		_, part, err := j.marketRepo.List(ctx, nil, map[string]interface{}{
			"status = ?":         active,
			"category = ?":       sc.category,
			"sub_category = ?":   sc.subCategory,
			"competition_en = ?": sc.competitionEn,
		}, "id asc", "")
		if err != nil {
			return nil, err
		}
		rows = append(rows, part...)
	}
	return rows, nil
}

func findMarket(rows []persistententity.Market, id uint) *persistententity.Market {
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i]
		}
	}
	return nil
}

func distinctPlatforms(members []mapMember) map[string]struct{} {
	out := make(map[string]struct{})
	for _, m := range members {
		if m.platform == "" {
			continue
		}
		out[m.platform] = struct{}{}
	}
	return out
}

// pickOneMarketPerPlatform 同组内每平台只保留 slug 最贴 canonical 的一场；其余从本组剔除并清空 map_key。
func pickOneMarketPerPlatform(rows []persistententity.Market, members []mapMember, canonicalFixture string) (keep, dropped []mapMember) {
	if len(members) == 0 {
		return nil, nil
	}
	canonicalFixture = strings.ToLower(strings.TrimSpace(canonicalFixture))
	byPlat := make(map[string][]mapMember)
	for _, m := range members {
		byPlat[m.platform] = append(byPlat[m.platform], m)
	}
	keep = make([]mapMember, 0, len(byPlat))
	for _, list := range byPlat {
		if len(list) == 1 {
			keep = append(keep, list[0])
			continue
		}
		best := list[0]
		bestScore := fixtureSlugMatchScore(rows, best.id, canonicalFixture)
		for _, m := range list[1:] {
			if sc := fixtureSlugMatchScore(rows, m.id, canonicalFixture); sc > bestScore {
				bestScore = sc
				best = m
			}
		}
		keep = append(keep, best)
		for _, m := range list {
			if m.id != best.id {
				dropped = append(dropped, m)
			}
		}
	}
	return keep, dropped
}

func fixtureSlugMatchScore(rows []persistententity.Market, id uint, canonical string) int {
	row := findMarket(rows, id)
	if row == nil || row.Slug == nil {
		return 0
	}
	slug := strings.ToLower(strings.TrimSpace(*row.Slug))
	base := polySlugFixtureBase(slug)
	if base == canonical {
		return 3
	}
	if sportSlugPrefix(slug) == sportSlugPrefix(canonical) {
		return 1
	}
	return 0
}

func (j *MarketMap) clearMapAssignment(ctx context.Context, id uint) error {
	if j == nil || j.marketRepo == nil || id == 0 {
		return nil
	}
	if err := j.marketRepo.Update(ctx, nil, map[string]interface{}{"id = ?": id}, map[string]interface{}{
		"map_key":      nil,
		"map_verified": 0,
	}); err != nil {
		return err
	}
	if j.outcomeRepo != nil {
		_ = j.outcomeRepo.Update(ctx, nil, map[string]interface{}{
			"market_row_id = ?": id,
		}, map[string]interface{}{
			"map_key": nil,
		})
	}
	return nil
}
