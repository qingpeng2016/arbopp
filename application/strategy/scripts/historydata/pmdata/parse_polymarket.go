package pmdata

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/parquet-go/parquet-go"
)

type polymarketL2Row struct {
	Timestamp int64          `parquet:"timestamp"`
	LocalTS   int64          `parquet:"local_timestamp"`
	BidPrices parquetDoubles `parquet:"bid_prices"`
	BidSizes  parquetDoubles `parquet:"bid_sizes"`
	AskPrices parquetDoubles `parquet:"ask_prices"`
	AskSizes  parquetDoubles `parquet:"ask_sizes"`
	BestBid   float64   `parquet:"best_bid"`
	BestAsk   float64   `parquet:"best_ask"`
	PcPrice   float64   `parquet:"pc_price"`
	PcSize    float64   `parquet:"pc_size"`
	PcSide    string    `parquet:"pc_side"`
}

type polyL1State struct {
	bidPx, bidSz, askPx, askSz float64
}

const defaultL1Size = 100

// ParsePolymarketL2Parquet 解析 PMData Polymarket 日 Parquet，按分钟降采样写入 snapshot。
func ParsePolymarketL2Parquet(data []byte, maxRows int) ([]PredictSnapshot, error) {
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
	r := parquet.NewGenericReader[polymarketL2Row](f)
	defer r.Close()
	buf := make([]polymarketL2Row, 64)
	var st polyL1State
	var raw []PredictSnapshot
	for {
		n, readErr := r.Read(buf)
		for i := 0; i < n; i++ {
			snap, ok := polymarketRowToSnapshot(buf[i], &st)
			if !ok {
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
			return nil, fmt.Errorf("read polymarket parquet: %w", readErr)
		}
	}
	return capSnapshots(downsampleByMinute(raw), maxRows), nil
}

func capSnapshots(snaps []PredictSnapshot, max int) []PredictSnapshot {
	if max <= 0 || len(snaps) <= max {
		return snaps
	}
	return snaps[:max]
}

func polymarketRowToSnapshot(row polymarketL2Row, st *polyL1State) (PredictSnapshot, bool) {
	if st == nil {
		return PredictSnapshot{}, false
	}
	ts := rowTsMs(row.Timestamp, row.LocalTS)
	bids := zipLevels(row.BidPrices.values(), row.BidSizes.values())
	asks := zipLevels(row.AskPrices.values(), row.AskSizes.values())
	if len(bids) > 0 || len(asks) > 0 {
		if len(bids) > 0 {
			st.bidPx, st.bidSz = bids[0][0], bids[0][1]
		}
		if len(asks) > 0 {
			st.askPx, st.askSz = asks[0][0], asks[0][1]
		}
		if ts <= 0 {
			return PredictSnapshot{}, false
		}
		return PredictSnapshot{TsMs: ts, Bids: bids, Asks: asks}, true
	}
	if row.BestBid > 0 {
		st.bidPx = row.BestBid
		if row.PcSize > 0 && sideTouchesBid(row) {
			st.bidSz = row.PcSize
		}
		if st.bidSz <= 0 {
			st.bidSz = defaultL1Size
		}
	}
	if row.BestAsk > 0 {
		st.askPx = row.BestAsk
		if row.PcSize > 0 && sideTouchesAsk(row) {
			st.askSz = row.PcSize
		}
		if st.askSz <= 0 {
			st.askSz = defaultL1Size
		}
	}
	if st.bidPx <= 0 && st.askPx <= 0 || ts <= 0 {
		return PredictSnapshot{}, false
	}
	outBids := make([]BookLevel, 0, 1)
	outAsks := make([]BookLevel, 0, 1)
	if st.bidPx > 0 && st.bidSz > 0 {
		outBids = append(outBids, BookLevel{st.bidPx, st.bidSz})
	}
	if st.askPx > 0 && st.askSz > 0 {
		outAsks = append(outAsks, BookLevel{st.askPx, st.askSz})
	}
	if len(outBids) == 0 && len(outAsks) == 0 {
		return PredictSnapshot{}, false
	}
	return PredictSnapshot{TsMs: ts, Bids: outBids, Asks: outAsks}, true
}

func rowTsMs(ts, local int64) int64 {
	ms := microsToMs(ts)
	if ms <= 0 {
		ms = microsToMs(local)
	}
	return ms
}

func sideTouchesBid(row polymarketL2Row) bool {
	side := strings.ToUpper(strings.TrimSpace(row.PcSide))
	if side == "BUY" {
		return true
	}
	if row.PcPrice > 0 && row.BestBid > 0 {
		return math.Abs(row.PcPrice-row.BestBid) < 0.0005
	}
	return false
}

func sideTouchesAsk(row polymarketL2Row) bool {
	side := strings.ToUpper(strings.TrimSpace(row.PcSide))
	if side == "SELL" {
		return true
	}
	if row.PcPrice > 0 && row.BestAsk > 0 {
		return math.Abs(row.PcPrice-row.BestAsk) < 0.0005
	}
	return false
}