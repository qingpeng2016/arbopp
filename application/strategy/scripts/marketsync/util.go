package marketsync

import (
	"context"
	"fmt"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/batch"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/common"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func (j *MarketSync) batchUpsertMarkets(ctx context.Context, list []*persistententity.Market) error {
	return j.marketRepo.BatchUpsert(ctx, nil, list)
}

func (j *MarketSync) batchUpsertOutcomes(ctx context.Context, list []*persistententity.MarketOutcome) error {
	if j.outcomeRepo == nil {
		return nil
	}
	return j.outcomeRepo.BatchUpsert(ctx, nil, list)
}

func (j *MarketSync) batchUpsertFixtureOutcomes(ctx context.Context, fo *batch.FixtureOutcome) error {
	if fo == nil {
		return nil
	}
	fixtures := dedupeFixtures(fo.Fixtures)
	if len(fixtures) > 0 {
		if err := j.marketRepo.BatchUpsert(ctx, nil, fixtures); err != nil {
			return err
		}
	}
	if len(fo.Outcomes) > 0 && j.outcomeRepo != nil {
		fillOutcomeMarketNames(fo.Fixtures, fo.Outcomes)
		if err := j.fillOutcomeMarketRowIDs(ctx, fo.Outcomes); err != nil {
			return err
		}
		return j.outcomeRepo.BatchUpsert(ctx, nil, fo.Outcomes)
	}
	return nil
}

// fillOutcomeMarketRowIDs 写入 market.id，便于按场次主键关联查询。
func (j *MarketSync) fillOutcomeMarketRowIDs(ctx context.Context, outcomes []*persistententity.MarketOutcome) error {
	if j.marketRepo == nil {
		return nil
	}
	cache := make(map[string]uint)
	for _, o := range outcomes {
		if o == nil || o.PlatformMarketID == "" {
			continue
		}
		ck := outcomeParentMarketCacheKey(o)
		if id, ok := cache[ck]; ok {
			o.MarketRowID = id
			continue
		}
		where := map[string]interface{}{
			"platform = ?":      o.Platform,
			"platform_market_id = ?": o.PlatformMarketID,
			"category = ?":      derefStr(o.Category),
			"sub_category = ?":  derefStr(o.SubCategory),
		}
		m, err := j.marketRepo.FindOne(ctx, nil, where, "id asc")
		if err != nil {
			return fmt.Errorf("resolve market_row_id platform=%s platform_market_id=%s: %w",
				o.Platform, o.PlatformMarketID, err)
		}
		cache[ck] = m.ID
		o.MarketRowID = m.ID
	}
	return nil
}

func outcomeParentMarketCacheKey(o *persistententity.MarketOutcome) string {
	return o.Platform + "\x00" + o.PlatformMarketID + "\x00" + derefStr(o.Category) + "\x00" + derefStr(o.SubCategory)
}

// fillOutcomeMarketNames 用本批 fixture 的 name 写 market_name（与 market 表 upsert 内容一致）。
func fillOutcomeMarketNames(fixtures []*persistententity.Market, outcomes []*persistententity.MarketOutcome) {
	if len(fixtures) == 0 || len(outcomes) == 0 {
		return
	}
	byKey := make(map[string]*string, len(fixtures))
	for _, fx := range fixtures {
		if fx == nil || fx.Name == nil {
			continue
		}
		k := fixtureUpsertKey(fx)
		if k == "" {
			continue
		}
		byKey[k] = fx.Name
	}
	for _, o := range outcomes {
		if o == nil {
			continue
		}
		name := byKey[outcomeParentMarketCacheKey(o)]
		if name != nil {
			o.MarketName = name
		}
	}
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func dedupeFixtures(list []*persistententity.Market) []*persistententity.Market {
	seen := make(map[string]*persistententity.Market, len(list))
	for _, m := range list {
		if m == nil {
			continue
		}
		k := fixtureUpsertKey(m)
		if k == "" {
			continue
		}
		seen[k] = m
	}
	out := make([]*persistententity.Market, 0, len(seen))
	for _, m := range seen {
		out = append(out, m)
	}
	return out
}

func fixtureUpsertKey(m *persistententity.Market) string {
	if m == nil {
		return ""
	}
	cat, sub := "", ""
	if m.Category != nil {
		cat = *m.Category
	}
	if m.SubCategory != nil {
		sub = *m.SubCategory
	}
	return m.Platform + "\x00" + m.PlatformMarketID + "\x00" + cat + "\x00" + sub
}

// 兼容子包内仍引用 marketsync 包级 helper 的测试/过渡（新代码请用 common 包）。
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func strPtr(s string) *string {
	s = truncate(s, 2000)
	return &s
}

func strPtr255(s string) *string  { s = truncate(s, 255); return &s }
func strPtr200(s string) *string  { s = truncate(s, 200); return &s }
func strPtr128(s string) *string  { s = truncate(s, 128); return &s }
func floatPtr(v float64) *float64  { return &v }

func skipEventEndedUTC(eventStartTime, eventEndTime string) bool {
	return common.SkipEventEndedUTC(eventStartTime, eventEndTime)
}
