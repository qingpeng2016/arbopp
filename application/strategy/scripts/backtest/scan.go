package backtest

import (
	"context"
	"fmt"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

// ScanResult 按秒重放后统计策略是否命中套利。
type ScanResult struct {
	Seconds     int
	HitSeconds  int
	TotalHits   int
	FirstHitSec int
	SamplePer   float64
}

// ScanMapKeyOpportunities 读 data_history_order_book，扫描二元 yesno 与 1X2 onextwo 命中（只读不落库）。
func ScanMapKeyOpportunities(
	ctx context.Context,
	db *gorm.DB,
	historyRepo persistentrepo.HistoryDataOrderBookRepo,
	outcomeRepo persistentrepo.MarketOutcomeRepo,
	marketRepo persistentrepo.MarketRepo,
	platformRepo persistentrepo.PlatformRepo,
	thresholdRepo persistentrepo.ConfigStrategyThresholdRepo,
	feeRepo persistentrepo.ConfigStrategyFeeRepo,
	mapKey string,
) (ScanResult, error) {
	var out ScanResult
	out.FirstHitSec = -1
	if historyRepo == nil || outcomeRepo == nil || marketRepo == nil || platformRepo == nil {
		return out, fmt.Errorf("repo nil")
	}
	_, snaps, err := historyRepo.List(ctx, nil, map[string]interface{}{
		"map_key = ?":      mapKey,
		"record_kind = ?":  persistententity.HistoryRecordKindSnapshot,
		"snapshot_at_ms > ?": 0,
	}, "snapshot_at_ms asc, id asc", "")
	if err != nil {
		return out, err
	}
	if len(snaps) == 0 {
		return out, fmt.Errorf("no snapshots for map_key %q", mapKey)
	}
	fixtures, err := marketmap.LoadMarketsByMapKey(ctx, marketRepo, platformRepo, mapKey)
	if err != nil {
		return out, err
	}
	outcomes, err := marketmap.OutcomesForFixtureMarkets(ctx, outcomeRepo, fixtures)
	if err != nil {
		return out, err
	}
	params := loadStrategyParams(ctx, thresholdRepo, feeRepo)
	marketsByRow := map[uint]persistententity.Market{}
	for i := range fixtures {
		fx := fixtures[i]
		marketsByRow[fx.ID] = fx
	}
	sess := newStrategyEntrySession(ctx, mapKey, outcomes, marketsByRow, 0, params, nil, nil)

	tsList := groupSnapshotsByTs(snaps)
	out.Seconds = len(tsList)
	for _, tsMs := range tsList {
		replayAt := time.UnixMilli(tsMs).UTC()
		sess.SetReplayTime(replayAt)
		sec := parseSecFromBatch(rowsAtTs(snaps, tsMs))
		batch := rowsAtTs(snaps, tsMs)
		for _, row := range batch {
			_ = applySnapshot(sess.watch, row, replayAt)
		}
		secHit := false
		for i := range outcomes {
			pick, ok := sess.runner.TryPickForOutcome(outcomes[i].ID)
			if !ok {
				continue
			}
			out.TotalHits++
			secHit = true
			if out.FirstHitSec < 0 {
				out.FirstHitSec = sec
				out.SamplePer = pick.Per
			}
		}
		if sess.onextwo != nil {
			if pick, ok := sess.onextwo.TryTriplePick(); ok {
				out.TotalHits++
				secHit = true
				if out.FirstHitSec < 0 {
					out.FirstHitSec = sec
					out.SamplePer = pick.Per
				}
			}
		}
		if secHit {
			out.HitSeconds++
		}
	}
	return out, nil
}

func parseSecFromBatch(batch []*persistententity.HistoryDataOrderBook) int {
	for _, row := range batch {
		if row == nil {
			continue
		}
		var sec int
		if _, err := fmt.Sscanf(row.ErrorText, "sec=%d", &sec); err == nil {
			return sec
		}
	}
	return -1
}
