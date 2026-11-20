package common

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// PredictBookRows 由 YES 视角盘口写 Predict 的 YES/NO 两行（token_id 对应该侧 outcome）。
func PredictBookRows(m *persistententity.MarketOutcome, yesBids, yesAsks []book.Level) []*persistententity.OrderBook {
	ms := time.Now().UnixMilli()
	return predictBookRows(m, nil, yesBids, yesAsks, ms/1000, ms)
}

// PredictBookRowsWithFixture orderbook_sync 写库时带上父场次冗余字段。
func PredictBookRowsWithFixture(m *persistententity.MarketOutcome, fx *persistententity.Market, yesBids, yesAsks []book.Level, snapshotSecond, snapshotAtMs int64) []*persistententity.OrderBook {
	return predictBookRows(m, fx, yesBids, yesAsks, snapshotSecond, snapshotAtMs)
}

func predictBookRows(m *persistententity.MarketOutcome, fx *persistententity.Market, yesBids, yesAsks []book.Level, snapshotSecond, snapshotAtMs int64) []*persistententity.OrderBook {
	if m == nil {
		return nil
	}
	yesTok := strings.TrimSpace(m.YesTokenID)
	noTok := strings.TrimSpace(m.NoTokenID)
	snap := SnapshotSecondOrNow(snapshotSecond)
	ms := snapshotAtMs
	if ms <= 0 {
		ms = snap * 1000
	}
	var rows []*persistententity.OrderBook
	if yesTok != "" {
		rows = append(rows, RowFromOutcomeAt(m, fx, levelsJSON(yesBids), levelsJSON(yesAsks), yesTok, snap, ms))
	}
	if noTok != "" && noTok != yesTok {
		noBids, noAsks := book.NoBookFromYesBook(yesBids, yesAsks)
		rows = append(rows, RowFromOutcomeAt(m, fx, levelsJSON(noBids), levelsJSON(noAsks), noTok, snap, ms))
	}
	return rows
}

// LevelsFromPairs REST/WS 的 [price,size] 转成内存档位。
func LevelsFromPairs(rows [][]float64) []book.Level {
	out := make([]book.Level, 0, len(rows))
	for _, r := range rows {
		if len(r) < 2 {
			continue
		}
		out = append(out, book.Level{Price: r[0], Size: r[1]})
	}
	return out
}

func levelsJSON(ls []book.Level) string {
	rows := make([][]string, 0, len(ls))
	for _, lv := range ls {
		rows = append(rows, []string{
			strconv.FormatFloat(lv.Price, 'f', -1, 64),
			strconv.FormatFloat(lv.Size, 'f', -1, 64),
		})
	}
	b, err := json.Marshal(rows)
	if err != nil || len(b) == 0 {
		return "[]"
	}
	return string(b)
}