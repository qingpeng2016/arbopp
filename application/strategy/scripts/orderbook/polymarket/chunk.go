package polymarket

import (
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
		tokens := tokenIDsForOutcome(&m)
		if len(tokens) == 0 {
			continue
		}
		if len(cur.items)+len(tokens) > batchSize && len(cur.items) > 0 {
			appendChunk()
		}
		cur.markets = append(cur.markets, m)
		for _, tokenID := range tokens {
			cur.items = append(cur.items, httpentity.BatchGetOrderBooksRequestItem{TokenID: tokenID})
		}
		if len(cur.items) >= batchSize {
			appendChunk()
		}
	}
	appendChunk()
	return chunks
}