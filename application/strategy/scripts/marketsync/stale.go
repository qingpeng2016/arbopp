package marketsync

import (
	"context"
	"fmt"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"go.uber.org/zap"
)

// closeStaleAfterSync 关本轮未见的场次。盘口没有自己的状态，跟着 market.status。
func (j *MarketSync) closeStaleAfterSync(
	ctx context.Context,
	platform, category, subCategory string,
	seenFixtures, _ map[string]struct{},
) (fixturesClosed, outcomesClosed int, err error) {
	n, err := j.closeStaleActiveMarkets(ctx, platform, category, subCategory, seenFixtures)
	if err != nil {
		return 0, 0, err
	}
	return n, 0, nil
}

// closeStaleActiveMarkets 本轮同步仍 ACTIVE 的 platform_market_id 在 seen 中；其余同平台/子类标为 CLOSE。
func (j *MarketSync) closeStaleActiveMarkets(
	ctx context.Context,
	platform, category, subCategory string,
	seen map[string]struct{},
) (int, error) {
	if j.marketRepo == nil {
		return 0, nil
	}
	_, list, err := j.marketRepo.List(ctx, nil, map[string]interface{}{
		"platform = ?":     platform,
		"category = ?":     category,
		"sub_category = ?": subCategory,
		"status = ?":       string(cryptoPredictionEnum.MarketStatusActive),
	}, "id asc", "")
	if err != nil {
		return 0, fmt.Errorf("list active markets: %w", err)
	}
	closeStatus := string(cryptoPredictionEnum.MarketStatusClose)
	closed := 0
	for i := range list {
		m := list[i]
		if _, ok := seen[m.PlatformMarketID]; ok {
			continue
		}
		if err := j.marketRepo.Update(ctx, nil, map[string]interface{}{
			"id = ?": m.ID,
		}, map[string]interface{}{
			"status": closeStatus,
		}); err != nil {
			return closed, fmt.Errorf("close market id=%d: %w", m.ID, err)
		}
		closed++
	}
	logger.InfoZ(ctx, "market_sync-closed-stale",
		zap.String("platform", platform),
		zap.String("sub_category", subCategory),
		zap.Int("closed", closed),
		zap.Int("still_open_in_sync", len(seen)),
		zap.Int("active_before", len(list)))
	return closed, nil
}
