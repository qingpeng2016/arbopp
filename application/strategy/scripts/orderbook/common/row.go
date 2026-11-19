package common

import (
	"strings"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// RowFromOutcome 由 market_outcome + 盘口 JSON 写 market_order_book 行。
func RowFromOutcome(m *persistententity.MarketOutcome, bidsJSON, asksJSON, tokenID string) *persistententity.OrderBook {
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
		MarketName:              m.Name,
		MarketURL:               strings.TrimSpace(m.MarketURL),
		PlatformMarketOutcomeID: strings.TrimSpace(m.PlatformMarketOutcomeID),
		MarketAddress:           addr,
		TokenID:                 tokenID,
		OrderBookBids:           bidsJSON,
		OrderBookAsks:           asksJSON,
	}
	if m.SelectionID != nil {
		row.SelectionID = m.SelectionID
	}
	return row
}