package main

import (
	"context"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	strategyentry "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func buildScanWatch(mapKey string, outcomes []outcomeRec, params strategyentry.Params) (*watchgroup.Watch, []persistententity.MarketOutcome) {
	entities := make([]persistententity.MarketOutcome, 0, len(outcomes))
	for _, o := range outcomes {
		mo := outcomeToEntity(o)
		mo.ID = o.ID
		if (mo.MapKey == nil || strings.TrimSpace(*mo.MapKey) == "") && strings.TrimSpace(mapKey) != "" {
			mk := strings.TrimSpace(mapKey)
			mo.MapKey = &mk
		}
		entities = append(entities, *mo)
	}
	w := &watchgroup.Watch{
		Ctx:    context.Background(),
		MapKey: mapKey,
		Books:  book.NewStore(),
		Outcomes: entities,
		ByToken:   map[string]int{},
		ByTicker:  map[string]int{},
		ByPredict: map[string]int{},
		BySxbet:   map[string]int{},
	}
	for i := range w.Outcomes {
		oc := &w.Outcomes[i]
		plat := strings.ToUpper(strings.TrimSpace(oc.Platform))
		switch plat {
		case appcommon.PlatformPolymarket:
			if tok := strings.TrimSpace(oc.YesTokenID); tok != "" {
				w.ByToken[tok] = i
			}
			if tok := strings.TrimSpace(oc.NoTokenID); tok != "" && tok != strings.TrimSpace(oc.YesTokenID) {
				w.ByToken[tok] = i
			}
		case appcommon.PlatformKalshi:
			id := strings.TrimSpace(oc.PlatformMarketOutcomeID)
			if id != "" {
				w.ByTicker[id] = i
			}
		case appcommon.PlatformPredict:
			if id := strings.TrimSpace(oc.PlatformMarketOutcomeID); id != "" {
				w.ByPredict[id] = i
			}
		case appcommon.PlatformSxbet:
			if id := strings.TrimSpace(oc.PlatformMarketOutcomeID); id != "" {
				w.BySxbet[id] = i
			}
		}
	}
	w.Ops = watchgroup.Ops{
		StrategyParams: func() (float64, float64, float64) {
			return params.MaxCost, params.MinAmount, params.MaxAmount
		},
		OneX2Limits: func() (float64, float64) {
			return params.OneX2YesMax, params.OneX2NoMax
		},
		TradeFee: func(platform, category, sub string) float64 {
			return params.Fee(platform, category, sub)
		},
	}
	return w, entities
}

func levelsToBook(ls []level) []book.Level {
	out := make([]book.Level, 0, len(ls))
	for _, lv := range ls {
		out = append(out, book.Level{Price: lv.price, Size: lv.size})
	}
	return out
}

func applyBookRow(store *book.Store, row wsRow, mo outcomeRec) {
	if store == nil {
		return
	}
	if !useWSRowForTopBook(row.Platform, row.TokenID, mo) {
		return
	}
	asks := levelsToBook(parseLevels(row.Asks))
	bids := levelsToBook(parseLevels(row.Bids))
	ts := time.UnixMilli(row.SnapshotAtMs)
	if row.SnapshotAtMs <= 0 {
		ts = time.Now().UTC()
	}
	plat := strings.ToUpper(strings.TrimSpace(row.Platform))
	switch plat {
	case appcommon.PlatformPolymarket:
		tok := strings.TrimSpace(row.TokenID)
		if tok == "" {
			return
		}
		store.ApplyPolyBook(tok, bids, asks, 0.01, 1, ts)
	case appcommon.PlatformPredict:
		id := strings.TrimSpace(row.PlatformMarketOutcomeID)
		store.ApplyPredictBook(id, bids, asks, ts)
	case appcommon.PlatformSxbet:
		id := strings.TrimSpace(row.PlatformMarketOutcomeID)
		store.ApplySxbetBook(id, bids, asks, ts)
	case appcommon.PlatformKalshi:
		id := strings.TrimSpace(row.PlatformMarketOutcomeID)
		store.ApplyKalshiYesBook(id, bids, asks, ts)
	}
}

func filledPickToHit(mode, gk string, fp yesno.FilledPick, snapMs, triggerID int64) yesnoHit {
	return yesnoHit{
		Mode:        mode,
		GroupKey:    gk,
		YesPlatform: fp.Yes.Platform,
		NoPlatform:  fp.No.Platform,
		Per:         round4(fp.Per),
		Qty:         round4(fp.Qty),
		SnapshotMs:  snapMs,
		TriggerWSID: triggerID,
	}
}