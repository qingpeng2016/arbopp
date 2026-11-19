package predict

import (
	"strconv"
	"testing"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestBooksToRowsAllOutcomes(t *testing.T) {
	markets := []persistententity.MarketOutcome{
		{Platform: "PREDICT", PlatformMarketID: "clf-tur-jab-2026-11-17", PlatformMarketOutcomeID: "4211675", YesTokenID: "yes1", NoTokenID: "no1"},
		{Platform: "PREDICT", PlatformMarketID: "clf-tur-jab-2026-11-17", PlatformMarketOutcomeID: "4211677", YesTokenID: "yes2", NoTokenID: "no2"},
		{Platform: "PREDICT", PlatformMarketID: "clf-tur-jab-2026-11-17", PlatformMarketOutcomeID: "4212047", YesTokenID: "yes3", NoTokenID: "no3"},
	}
	books := []httpentity.PredictOrderbook{
		{MarketID: 4211675, Asks: [][]float64{{0.97, 200}}, Bids: [][]float64{{0.01, 600}}},
		{MarketID: 4211677, Asks: [][]float64{{0.5, 10}}, Bids: [][]float64{{0.49, 8}}},
	}
	rows, missed := booksToRows(markets, books)
	if len(rows) != 4 || missed != 1 {
		t.Fatalf("rows=%d missed=%d", len(rows), missed)
	}
	if rows[0].TokenID != "yes1" || rows[1].TokenID != "no1" || rows[2].TokenID != "yes2" || rows[3].TokenID != "no2" {
		t.Fatalf("tokens=%s %s %s %s", rows[0].TokenID, rows[1].TokenID, rows[2].TokenID, rows[3].TokenID)
	}
}

func TestChunkOutcomes(t *testing.T) {
	markets := make([]persistententity.MarketOutcome, 3)
	for i := range markets {
		markets[i].PlatformMarketOutcomeID = strconv.Itoa(i + 1)
	}
	chunks := chunkOutcomes(markets, 2)
	if len(chunks) != 2 || len(chunks[0].ids) != 2 || len(chunks[1].ids) != 1 {
		t.Fatalf("chunks=%d sizes=%d %d", len(chunks), len(chunks[0].ids), len(chunks[1].ids))
	}
}
