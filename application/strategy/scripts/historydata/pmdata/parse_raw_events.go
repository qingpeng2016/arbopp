package pmdata

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/parquet-go/parquet-go"
)

// RawMarketEvent Parquet 内一行，与 PMData 字段一致。
type RawMarketEvent struct {
	RowIndex       int
	EventKind      string
	EventType      string
	EventAtMs      int64
	LocalEventAtMs int64
	PayloadJSON    string
}

type polyRawPayload struct {
	EventType      string    `json:"event_type"`
	Timestamp      int64     `json:"timestamp"`
	LocalTimestamp int64     `json:"local_timestamp"`
	BidPrices      []float64 `json:"bid_prices"`
	BidSizes       []float64 `json:"bid_sizes"`
	AskPrices      []float64 `json:"ask_prices"`
	AskSizes       []float64 `json:"ask_sizes"`
	BestBid        float64   `json:"best_bid"`
	BestAsk        float64   `json:"best_ask"`
	PcPrice        float64   `json:"pc_price"`
	PcSize         float64   `json:"pc_size"`
	PcSide         string    `json:"pc_side"`
}

type predictRawPayload struct {
	MarketID           int64     `json:"market_id"`
	Timestamp          int64     `json:"timestamp"`
	LocalTimestamp     int64     `json:"local_timestamp"`
	BidPrices          []float64 `json:"bid_prices"`
	BidSizes           []float64 `json:"bid_sizes"`
	AskPrices          []float64 `json:"ask_prices"`
	AskSizes           []float64 `json:"ask_sizes"`
	UpdateTimestampMs  int64     `json:"update_timestamp_ms"`
	UpdateTimestampMs2 int64     `json:"updateTimestampMs"`
}

// ParsePolymarketRawEvents 全部行：book + price_change，不降采样、不截断。
func ParsePolymarketRawEvents(data []byte) ([]RawMarketEvent, error) {
	f, err := parquet.OpenFile(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	r := parquet.NewGenericReader[polymarketL2Row](f)
	defer r.Close()
	buf := make([]polymarketL2Row, 128)
	var out []RawMarketEvent
	rowIndex := 0
	for {
		n, readErr := r.Read(buf)
		for i := 0; i < n; i++ {
			ev, ok := polymarketRowToRaw(buf[i], rowIndex)
			rowIndex++
			if !ok {
				continue
			}
			out = append(out, ev)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return out, readErr
		}
	}
	return out, nil
}

func polymarketRowToRaw(row polymarketL2Row, rowIndex int) (RawMarketEvent, bool) {
	bids := row.BidPrices.values()
	bidSizes := row.BidSizes.values()
	asks := row.AskPrices.values()
	askSizes := row.AskSizes.values()
	payload := polyRawPayload{
		EventType:      strings.TrimSpace(row.EventType),
		Timestamp:      row.Timestamp,
		LocalTimestamp: row.LocalTS,
		BidPrices:      bids,
		BidSizes:       bidSizes,
		AskPrices:      asks,
		AskSizes:       askSizes,
		BestBid:        row.BestBid,
		BestAsk:        row.BestAsk,
		PcPrice:        row.PcPrice,
		PcSize:         row.PcSize,
		PcSide:         strings.TrimSpace(row.PcSide),
	}
	eventAt := rowTsMs(row.Timestamp, row.LocalTS)
	localAt := microsToMs(row.LocalTS)
	kind := ""
	if len(bids) > 0 || len(asks) > 0 {
		kind = "book"
	} else if row.BestBid > 0 || row.BestAsk > 0 || row.PcPrice > 0 {
		kind = "price_change"
	} else {
		return RawMarketEvent{}, false
	}
	if eventAt <= 0 {
		return RawMarketEvent{}, false
	}
	raw, _ := json.Marshal(payload)
	return RawMarketEvent{
		RowIndex:       rowIndex,
		EventKind:      kind,
		EventType:      payload.EventType,
		EventAtMs:      eventAt,
		LocalEventAtMs: localAt,
		PayloadJSON:    string(raw),
	}, true
}

// ParsePredictRawEvents Predict 全 orderbook 行。
func ParsePredictRawEvents(data []byte) ([]RawMarketEvent, error) {
	f, err := parquet.OpenFile(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	r := parquet.NewGenericReader[predictL2Row](f)
	defer r.Close()
	buf := make([]predictL2Row, 64)
	var out []RawMarketEvent
	rowIndex := 0
	for {
		n, readErr := r.Read(buf)
		for i := 0; i < n; i++ {
			ev, ok := predictRowToRaw(buf[i], rowIndex)
			rowIndex++
			if !ok {
				continue
			}
			out = append(out, ev)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return out, readErr
		}
	}
	return out, nil
}

func predictRowToRaw(row predictL2Row, rowIndex int) (RawMarketEvent, bool) {
	bids := row.BidPrices.values()
	bidSizes := row.BidSizes.values()
	asks := row.AskPrices.values()
	askSizes := row.AskSizes.values()
	if len(bids) == 0 && len(asks) == 0 {
		return RawMarketEvent{}, false
	}
	eventAt := row.UpdateMS
	if eventAt <= 0 {
		eventAt = row.UpdateTS
	}
	if eventAt <= 0 {
		eventAt = microsToMs(row.Timestamp)
	}
	if eventAt <= 0 {
		eventAt = microsToMs(row.LocalTS)
	}
	if eventAt <= 0 {
		return RawMarketEvent{}, false
	}
	payload := predictRawPayload{
		MarketID:           row.MarketID,
		Timestamp:          row.Timestamp,
		LocalTimestamp:     row.LocalTS,
		BidPrices:          bids,
		BidSizes:           bidSizes,
		AskPrices:          asks,
		AskSizes:           askSizes,
		UpdateTimestampMs:  row.UpdateMS,
		UpdateTimestampMs2: row.UpdateTS,
	}
	raw, _ := json.Marshal(payload)
	return RawMarketEvent{
		RowIndex:       rowIndex,
		EventKind:      "orderbook",
		EventType:      "orderbook",
		EventAtMs:      eventAt,
		LocalEventAtMs: microsToMs(row.LocalTS),
		PayloadJSON:    string(raw),
	}, true
}