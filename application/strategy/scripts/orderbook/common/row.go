package common

import (
	"strings"
	"time"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// RowFromOutcome 由 market_outcome + 盘口 JSON 写 market_order_book 行。
func RowFromOutcome(m *persistententity.MarketOutcome, fx *persistententity.Market, bidsJSON, asksJSON, tokenID string) *persistententity.OrderBook {
	ms := time.Now().UnixMilli()
	return RowFromOutcomeAt(m, fx, bidsJSON, asksJSON, tokenID, ms/1000, ms)
}

// RowFromOutcomeAt 同上；snapshotAtMs≤0 时用 snapshotSecond×1000。
func RowFromOutcomeAt(m *persistententity.MarketOutcome, fx *persistententity.Market, bidsJSON, asksJSON, tokenID string, snapshotSecond, snapshotAtMs int64) *persistententity.OrderBook {
	if m == nil {
		return nil
	}
	addr := ""
	if m.MarketAddress != nil {
		addr = strings.TrimSpace(*m.MarketAddress)
	}
	row := &persistententity.OrderBook{
		Platform:                strings.ToUpper(strings.TrimSpace(m.Platform)),
		MarketRowID:             m.MarketRowID,
		PlatformMarketID:        strings.TrimSpace(m.PlatformMarketID),
		MarketURL:               strings.TrimSpace(m.MarketURL),
		PlatformMarketOutcomeID: strings.TrimSpace(m.PlatformMarketOutcomeID),
		MarketAddress:           addr,
		TokenID:                 tokenID,
		OrderBookBids:           bidsJSON,
		OrderBookAsks:           asksJSON,
		SnapshotSecond:          snapshotSecond,
		SnapshotAtMs:            snapshotAtMsOrDefault(snapshotSecond, snapshotAtMs),
	}
	if m.SelectionID != nil {
		row.SelectionID = m.SelectionID
	}
	EnrichOrderBookFromOutcome(row, m, fx, tokenID)
	return row
}

func snapshotAtMsOrDefault(snapshotSecond, snapshotAtMs int64) int64 {
	if snapshotAtMs > 0 {
		return snapshotAtMs
	}
	if snapshotSecond > 0 {
		return snapshotSecond * 1000
	}
	return time.Now().UnixMilli()
}

// ApplyOrderBookFixtureMeta 写 map_key / 联赛等冗余列（来自 outcome + 父场次 market）。
func ApplyOrderBookFixtureMeta(row *persistententity.OrderBook, oc *persistententity.MarketOutcome, fx *persistententity.Market) {
	if row == nil || oc == nil {
		return
	}
	row.Category = oc.Category
	row.SubCategory = oc.SubCategory
	row.Competition = oc.Competition
	if fx == nil {
		return
	}
	if row.Category == nil {
		row.Category = fx.Category
	}
	if row.SubCategory == nil {
		row.SubCategory = fx.SubCategory
	}
	if row.Competition == nil {
		row.Competition = fx.Competition
	}
	row.MapKey = fx.MapKey
	row.CompetitionEn = fx.CompetitionEn
}