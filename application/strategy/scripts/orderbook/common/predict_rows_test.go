package common

import (
	"encoding/json"
	"strconv"
	"testing"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestPredictBookRowsYesAndNo(t *testing.T) {
	m := &persistententity.MarketOutcome{
		Platform: "PREDICT", YesTokenID: "y", NoTokenID: "n",
		PlatformMarketOutcomeID: "4211675",
	}
	yesBids := LevelsFromPairs([][]float64{{0.01, 600}})
	yesAsks := LevelsFromPairs([][]float64{{0.97, 200}})
	rows := PredictBookRows(m, yesBids, yesAsks)
	if len(rows) != 2 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[0].TokenID != "y" || rows[1].TokenID != "n" {
		t.Fatalf("tokens=%s %s", rows[0].TokenID, rows[1].TokenID)
	}
	askPx, _ := firstLevelPrice(rows[1].OrderBookAsks)
	bidPx, _ := firstLevelPrice(rows[1].OrderBookBids)
	if askPx < 0.989 || askPx > 0.991 {
		t.Fatalf("no ask px=%v", askPx)
	}
	if bidPx < 0.029 || bidPx > 0.031 {
		t.Fatalf("no bid px=%v", bidPx)
	}
}

func firstLevelPrice(bookJSON string) (float64, float64) {
	var levels [][]string
	if err := json.Unmarshal([]byte(bookJSON), &levels); err != nil || len(levels) == 0 || len(levels[0]) < 2 {
		return 0, 0
	}
	px, _ := strconv.ParseFloat(levels[0][0], 64)
	sz, _ := strconv.ParseFloat(levels[0][1], 64)
	return px, sz
}