package pmdata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/parquet-go/parquet-go"
)

// BookLevel [price, size]
type BookLevel [2]float64

// PredictSnapshot 从 Predict L2 Parquet 解析的一条全簿。
type PredictSnapshot struct {
	TsMs int64
	Bids []BookLevel
	Asks []BookLevel
}

type predictL2Row struct {
	MarketID  int64          `parquet:"market_id"`
	Timestamp int64          `parquet:"timestamp"`
	LocalTS   int64          `parquet:"local_timestamp"`
	BidPrices parquetDoubles `parquet:"bid_prices"`
	BidSizes  parquetDoubles `parquet:"bid_sizes"`
	AskPrices parquetDoubles `parquet:"ask_prices"`
	AskSizes  parquetDoubles `parquet:"ask_sizes"`
	UpdateMS  int64          `parquet:"updateTimestampMs"`
	UpdateTS  int64          `parquet:"update_timestamp_ms"`
}

// ParsePredictL2Parquet 按 PMData schema 读快照（限 maxRows）。
func ParsePredictL2Parquet(data []byte, maxRows int) ([]PredictSnapshot, error) {
	if len(data) == 0 {
		return nil, nil
	}
	if maxRows <= 0 {
		maxRows = 500
	}
	f, err := parquet.OpenFile(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	r := parquet.NewGenericReader[predictL2Row](f)
	defer r.Close()
	buf := make([]predictL2Row, 64)
	var raw []PredictSnapshot
	for {
		n, readErr := r.Read(buf)
		for i := 0; i < n; i++ {
			snap := rowToSnapshot(buf[i])
			if snap.TsMs <= 0 && len(snap.Bids) == 0 && len(snap.Asks) == 0 {
				continue
			}
			if len(snap.Bids) == 0 && len(snap.Asks) == 0 {
				continue
			}
			raw = append(raw, snap)
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			if len(raw) > 0 {
				return capSnapshots(downsampleByMinute(raw), maxRows), nil
			}
			return nil, fmt.Errorf("read parquet: %w", readErr)
		}
	}
	return capSnapshots(downsampleByMinute(raw), maxRows), nil
}

func rowToSnapshot(row predictL2Row) PredictSnapshot {
	ts := row.UpdateMS
	if ts <= 0 {
		ts = row.UpdateTS
	}
	if ts <= 0 {
		ts = microsToMs(row.Timestamp)
	}
	if ts <= 0 {
		ts = microsToMs(row.LocalTS)
	}
	return PredictSnapshot{
		TsMs: ts,
		Bids: zipLevels(row.BidPrices.values(), row.BidSizes.values()),
		Asks: zipLevels(row.AskPrices.values(), row.AskSizes.values()),
	}
}

func microsToMs(v int64) int64 {
	if v <= 0 {
		return 0
	}
	// Heuristic: epoch ms ~1e12, us ~1e15
	if v > 1_000_000_000_000_000 {
		return v / 1000
	}
	if v > 1_000_000_000_000 {
		return v
	}
	return v
}

func zipLevels(prices, sizes []float64) []BookLevel {
	n := len(prices)
	if len(sizes) < n {
		n = len(sizes)
	}
	out := make([]BookLevel, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, BookLevel{prices[i], sizes[i]})
	}
	return out
}

func snapshotDepth(s PredictSnapshot) int {
	return len(s.Bids) + len(s.Asks)
}

// MaxBookDepth 返回序列中 bid/ask 最大档位数（写库后用于对日志）。
func MaxBookDepth(snaps []PredictSnapshot, bids bool) int {
	max := 0
	for _, s := range snaps {
		n := len(s.Asks)
		if bids {
			n = len(s.Bids)
		}
		if n > max {
			max = n
		}
	}
	return max
}

// downsampleByMinute 每分钟一桶；同分钟内优先保留档位更深的快照（避免 Poly price_change 盖住 book 全簿）。
func downsampleByMinute(snaps []PredictSnapshot) []PredictSnapshot {
	if len(snaps) <= 1 {
		return snaps
	}
	seen := make(map[int64]PredictSnapshot)
	var order []int64
	for _, s := range snaps {
		if s.TsMs <= 0 {
			continue
		}
		b := s.TsMs / (60 * 1000)
		cur, ok := seen[b]
		if !ok {
			order = append(order, b)
			seen[b] = s
			continue
		}
		if snapshotDepth(s) > snapshotDepth(cur) {
			seen[b] = s
		} else if snapshotDepth(s) == snapshotDepth(cur) && s.TsMs >= cur.TsMs {
			seen[b] = s
		}
	}
	out := make([]PredictSnapshot, 0, len(order))
	for _, b := range order {
		out = append(out, seen[b])
	}
	return out
}

// LevelsJSON [[p,s],...]
func LevelsJSON(levels []BookLevel) string {
	if len(levels) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(levels)
	return string(b)
}

func ParseTimeUTCDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func VendorExchangePredict() string { return "predict_fun" }

func VendorExchangePolymarket() string { return "polymarket" }

func FileRecordSymbolHint(exchange, symbol string) string {
	return strings.TrimSpace(exchange) + ":" + strings.TrimSpace(symbol)
}