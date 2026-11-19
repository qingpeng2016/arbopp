package polymarket

import (
	"strings"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

type fetchChunk struct {
	markets []persistententity.MarketOutcome
	items   []httpentity.BatchGetOrderBooksRequestItem
}

func chunkOutcomes(markets []persistententity.MarketOutcome, batchSize int) []fetchChunk {
	if batchSize <= 0 {
		batchSize = 500
	}
	var chunks []fetchChunk
	var cur fetchChunk
	appendChunk := func() {
		if len(cur.items) == 0 {
			return
		}
		chunks = append(chunks, cur)
		cur = fetchChunk{}
	}
	for i := range markets {
		m := markets[i]
		token := batchItemFromOutcome(&m)
		if token.TokenID == "" {
			continue
		}
		cur.markets = append(cur.markets, m)
		cur.items = append(cur.items, token)
		if len(cur.items) >= batchSize {
			appendChunk()
		}
	}
	appendChunk()
	return chunks
}

func batchItemFromOutcome(m *persistententity.MarketOutcome) httpentity.BatchGetOrderBooksRequestItem {
	if m == nil {
		return httpentity.BatchGetOrderBooksRequestItem{}
	}
	return httpentity.BatchGetOrderBooksRequestItem{TokenID: strings.TrimSpace(m.YesTokenID)}
}