package pmdata

import (
	"bytes"
	"io"
	"math"
	"strings"

	"github.com/parquet-go/parquet-go"
)

type polymarketL2Row struct {
	EventType string         `parquet:"event_type"`
	Timestamp int64          `parquet:"timestamp"`
	LocalTS   int64          `parquet:"local_timestamp"`
	BidPrices parquetDoubles `parquet:"bid_prices"`
	BidSizes  parquetDoubles `parquet:"bid_sizes"`
	AskPrices parquetDoubles `parquet:"ask_prices"`
	AskSizes  parquetDoubles `parquet:"ask_sizes"`
	BestBid   float64        `parquet:"best_bid"`
	BestAsk   float64        `parquet:"best_ask"`
	PcPrice   float64        `parquet:"pc_price"`
	PcSize    float64        `parquet:"pc_size"`
	PcSide    string         `parquet:"pc_side"`
}

// ParsePolymarketL2Parquet 只落 PMData 中 book 事件自带的列表深度（不做 carry / 不拼推断深度）。
func ParsePolymarketL2Parquet(data []byte, maxRows int) ([]PredictSnapshot, error) {
	return ParsePolymarketHonestBookOnly(data, maxRows)
}

func capSnapshots(snaps []PredictSnapshot, max int) []PredictSnapshot {
	if max <= 0 || len(snaps) <= max {
		return snaps
	}
	return snaps[:max]
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

// 以下仅供分析工具使用 raw parquet 遍历。
func readAllPolyRows(data []byte) ([]polymarketL2Row, error) {
	f, err := parquet.OpenFile(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	r := parquet.NewGenericReader[polymarketL2Row](f)
	defer r.Close()
	buf := make([]polymarketL2Row, 128)
	var out []polymarketL2Row
	for {
		n, readErr := r.Read(buf)
		out = append(out, buf[:n]...)
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return out, readErr
		}
	}
	return out, nil
}