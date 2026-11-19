package kalshi

import (
	"strings"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

type fetchChunk struct {
	markets []persistententity.MarketOutcome
	tickers []string
}

func chunkOutcomes(markets []persistententity.MarketOutcome, batchSize int) []fetchChunk {
	if batchSize <= 0 {
		batchSize = 100
	}
	var chunks []fetchChunk
	var cur fetchChunk
	appendChunk := func() {
		if len(cur.tickers) == 0 {
			return
		}
		chunks = append(chunks, cur)
		cur = fetchChunk{}
	}
	for i := range markets {
		m := markets[i]
		ticker := strings.TrimSpace(m.PlatformMarketOutcomeID)
		if ticker == "" {
			continue
		}
		cur.markets = append(cur.markets, m)
		cur.tickers = append(cur.tickers, ticker)
		if len(cur.tickers) >= batchSize {
			appendChunk()
		}
	}
	appendChunk()
	return chunks
}