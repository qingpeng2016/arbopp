package pmdata

import (
	"bytes"
	"io"

	"github.com/parquet-go/parquet-go"
)

// PolyRawStats 未加工 Parquet 行统计。
type PolyRawStats struct {
	BookRows int
	PcRows   int
	PcSnaps  []PCBBOSnap
}

// PCBBOSnap 来自 Polymarket price_change 的源 BBO（非推断）。
type PCBBOSnap struct {
	Ts      int64
	BestBid float64
	BestAsk float64
}

// ParsePolymarketBookAndPCRaw 只统计 book / price_change，price_change 只取源里 BBO。
func ParsePolymarketBookAndPCRaw(data []byte) (PolyRawStats, error) {
	var out PolyRawStats
	if len(data) == 0 {
		return out, nil
	}
	f, err := parquet.OpenFile(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return out, err
	}
	r := parquet.NewGenericReader[polymarketL2Row](f)
	defer r.Close()
	buf := make([]polymarketL2Row, 128)
	for {
		n, readErr := r.Read(buf)
		for i := 0; i < n; i++ {
			row := buf[i]
			bids := zipLevels(row.BidPrices.values(), row.BidSizes.values())
			asks := zipLevels(row.AskPrices.values(), row.AskSizes.values())
			if len(bids) > 0 || len(asks) > 0 {
				out.BookRows++
				continue
			}
			if row.BestBid > 0 || row.BestAsk > 0 {
				out.PcRows++
				ts := rowTsMs(row.Timestamp, row.LocalTS)
				out.PcSnaps = append(out.PcSnaps, PCBBOSnap{Ts: ts, BestBid: row.BestBid, BestAsk: row.BestAsk})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			break
		}
	}
	return out, nil
}

// ParsePolymarketHonestBookOnly 仅 book 事件原文落库，不做 carry / 不拼 price_change。
func ParsePolymarketHonestBookOnly(data []byte, maxRows int) ([]PredictSnapshot, error) {
	if len(data) == 0 {
		return nil, nil
	}
	f, err := parquet.OpenFile(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	r := parquet.NewGenericReader[polymarketL2Row](f)
	defer r.Close()
	buf := make([]polymarketL2Row, 64)
	var raw []PredictSnapshot
	for {
		n, readErr := r.Read(buf)
		for i := 0; i < n; i++ {
			row := buf[i]
			bids := zipLevels(row.BidPrices.values(), row.BidSizes.values())
			asks := zipLevels(row.AskPrices.values(), row.AskSizes.values())
			if len(bids) == 0 && len(asks) == 0 {
				continue
			}
			ts := rowTsMs(row.Timestamp, row.LocalTS)
			if ts <= 0 {
				continue
			}
			raw = append(raw, PredictSnapshot{TsMs: ts, Bids: bids, Asks: asks})
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			if len(raw) > 0 {
				return capSnapshots(downsampleByMinute(raw), maxRows), nil
			}
			return nil, readErr
		}
	}
	return capSnapshots(downsampleByMinute(raw), maxRows), nil
}