package polymarket

import (
	"testing"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestBooksToRowsYesAndNo(t *testing.T) {
	m := persistententity.MarketOutcome{
		Platform: "POLYMARKET", PlatformMarketOutcomeID: "6998357",
		YesTokenID: "yes-tok", NoTokenID: "no-tok",
	}
	books := []httpentity.OrderBookResponse{
		{AssetID: "yes-tok", Bids: []httpentity.OrderBookEntry{{Price: "0.1", Size: "10"}}},
		{AssetID: "no-tok", Asks: []httpentity.OrderBookEntry{{Price: "0.9", Size: "5"}}},
	}
	rows, missed := booksToRows([]persistententity.MarketOutcome{m}, books, nil, 1000)
	if missed != 0 || len(rows) != 2 {
		t.Fatalf("rows=%d missed=%d", len(rows), missed)
	}
	if rows[0].TokenID != "yes-tok" || rows[1].TokenID != "no-tok" {
		t.Fatalf("tokens %s %s", rows[0].TokenID, rows[1].TokenID)
	}
}

func TestChunkOutcomesIncludesNoToken(t *testing.T) {
	m := persistententity.MarketOutcome{YesTokenID: "y", NoTokenID: "n"}
	chunks := chunkOutcomes([]persistententity.MarketOutcome{m}, 500)
	if len(chunks) != 1 || len(chunks[0].items) != 2 {
		t.Fatalf("items=%d", len(chunks[0].items))
	}
}