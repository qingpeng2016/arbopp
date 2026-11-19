package predict

import (
	"strings"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

type fetchChunk struct {
	markets []persistententity.MarketOutcome
	ids     []string
}

func chunkOutcomes(markets []persistententity.MarketOutcome, batchSize int) []fetchChunk {
	if batchSize <= 0 {
		batchSize = 100
	}
	var chunks []fetchChunk
	var cur fetchChunk
	appendChunk := func() {
		if len(cur.ids) == 0 {
			return
		}
		chunks = append(chunks, cur)
		cur = fetchChunk{}
	}
	for i := range markets {
		m := markets[i]
		id := strings.TrimSpace(m.PlatformMarketOutcomeID)
		if id == "" {
			continue
		}
		cur.markets = append(cur.markets, m)
		cur.ids = append(cur.ids, id)
		if len(cur.ids) >= batchSize {
			appendChunk()
		}
	}
	appendChunk()
	return chunks
}
