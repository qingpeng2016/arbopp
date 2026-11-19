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
	MarketID      int64      `parquet:"market_id"`
	Timestamp     int64      `parquet:"timestamp"` // parquet may store as logical timestamp; fallback read
	BidPrices     []float64  `parquet:"bid_prices"`
	BidSizes      []float64  `parquet:"bid_sizes"`
	AskPrices     []float64  `parquet:"ask_prices"`
	AskSizes      []float64  `parquet:"ask_sizes"`
	LocalTS       int64      `parquet:"local_timestamp"`
	UpdateMS      int64      `parquet:"updateTimestampMs"`
	UpdateTS      int64      `parquet:"update_timestamp_ms"`
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
	var out []PredictSnapshot
	for {
		n, readErr := r.Read(buf)
		for i := 0; i < n; i++ {
			snap := rowToSnapshot(buf[i])
			if snap.TsMs <= 0 && len(snap.Bids) == 0 && len(snap.Asks) == 0 {
				continue
			}
			out = append(out, snap)
			if len(out) >= maxRows {
				return downsampleByMinute(out), nil
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			if len(out) > 0 {
				return downsampleByMinute(out), nil
			}
			return nil, fmt.Errorf("read parquet: %w", readErr)
		}
	}
	return downsampleByMinute(out), nil
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
		Bids: zipLevels(row.BidPrices, row.BidSizes),
		Asks: zipLevels(row.AskPrices, row.AskSizes),
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

func downsampleByMinute(snaps []PredictSnapshot) []PredictSnapshot {
	if len(snaps) <= 1 {
		return snaps
	}
	type bucket struct {
		ts int64
		s  PredictSnapshot
	}
	seen := make(map[int64]PredictSnapshot)
	var order []int64
	for _, s := range snaps {
		if s.TsMs <= 0 {
			continue
		}
		b := s.TsMs / (60 * 1000)
		if _, ok := seen[b]; !ok {
			order = append(order, b)
		}
		seen[b] = s
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