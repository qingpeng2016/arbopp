package backtest

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func parseLevelPairs(raw string) ([]book.Level, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var pairs [][2]float64
	if err := json.Unmarshal([]byte(raw), &pairs); err != nil {
		return nil, err
	}
	out := make([]book.Level, 0, len(pairs))
	for _, p := range pairs {
		if p[1] <= 0 {
			continue
		}
		out = append(out, book.Level{Price: p[0], Size: p[1]})
	}
	return out, nil
}

func outcomeByID(outcomes []persistententity.MarketOutcome, id uint) *persistententity.MarketOutcome {
	for i := range outcomes {
		if outcomes[i].ID == id {
			return &outcomes[i]
		}
	}
	return nil
}

// applySnapshot 把一行历史 L2（各平台 YES 视角）写入内存；NO 由内存互补，与实盘 strategy_entry 一致。
func applySnapshot(w *watchgroup.Watch, row *persistententity.HistoryDataOrderBook, ts time.Time) error {
	if w == nil || row == nil || w.Books == nil {
		return fmt.Errorf("watch nil")
	}
	oc := outcomeByID(w.Outcomes, uint(row.MarketOutcomeID))
	if oc == nil {
		return fmt.Errorf("unknown market_outcome_id %d", row.MarketOutcomeID)
	}
	bids, err := parseLevelPairs(row.OrderBookBids)
	if err != nil {
		return err
	}
	asks, err := parseLevelPairs(row.OrderBookAsks)
	if err != nil {
		return err
	}
	plat := strings.ToUpper(strings.TrimSpace(oc.Platform))
	switch plat {
	case appcommon.PlatformPolymarket:
		yesTok := strings.TrimSpace(oc.YesTokenID)
		noTok := strings.TrimSpace(oc.NoTokenID)
		if yesTok == "" {
			return fmt.Errorf("polymarket missing yes token outcome=%d", oc.ID)
		}
		w.Books.ApplyPolyBook(yesTok, bids, asks, 0.01, 0, ts)
		if noTok != "" {
			noBids, noAsks := book.NoBookFromYesBook(bids, asks)
			w.Books.ApplyPolyBook(noTok, noBids, noAsks, 0.01, 0, ts)
		}
	case appcommon.PlatformPredict:
		id := strings.TrimSpace(oc.PlatformMarketOutcomeID)
		if id == "" {
			return fmt.Errorf("predict missing market id outcome=%d", oc.ID)
		}
		w.Books.ApplyPredictBook(id, bids, asks, ts)
	case appcommon.PlatformKalshi:
		id := strings.TrimSpace(oc.PlatformMarketOutcomeID)
		if id == "" {
			return fmt.Errorf("kalshi missing ticker outcome=%d", oc.ID)
		}
		w.Books.ApplyKalshiYesBook(id, bids, asks, ts)
	case appcommon.PlatformSxbet:
		id := strings.TrimSpace(oc.PlatformMarketOutcomeID)
		if id == "" {
			return fmt.Errorf("sxbet missing market hash outcome=%d", oc.ID)
		}
		w.Books.ApplySxbetBook(id, bids, asks, ts)
	default:
		return fmt.Errorf("unsupported platform %q", plat)
	}
	return nil
}

func groupSnapshotsByTs(rows []persistententity.HistoryDataOrderBook) []int64 {
	if len(rows) == 0 {
		return nil
	}
	var tsList []int64
	var last int64 = -1
	for i := range rows {
		t := rows[i].SnapshotAtMs
		if t != last {
			tsList = append(tsList, t)
			last = t
		}
	}
	return tsList
}

func rowsAtTs(rows []persistententity.HistoryDataOrderBook, tsMs int64) []*persistententity.HistoryDataOrderBook {
	var out []*persistententity.HistoryDataOrderBook
	for i := range rows {
		if rows[i].SnapshotAtMs == tsMs {
			out = append(out, &rows[i])
		}
	}
	return out
}