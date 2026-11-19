package kalshi

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
)

// Update 是 orderbook_snapshot 或 orderbook_delta。
type Update struct {
	Ticker   string
	Snapshot bool
	Seq      int64
	Yes      []book.Level
	No       []book.Level
	Side     string
	Price    float64
	Delta    float64
	TS       time.Time
}

func Parse(raw []byte) (Update, bool, error) {
	var frame struct {
		Type string          `json:"type"`
		Seq  int64           `json:"seq"`
		Msg  json.RawMessage `json:"msg"`
	}
	if err := json.Unmarshal(raw, &frame); err != nil {
		return Update{}, false, err
	}
	kind := strings.ToLower(strings.TrimSpace(frame.Type))
	switch kind {
	case "orderbook_snapshot":
		u, ok := parseSnapshot(frame.Msg, frame.Seq)
		return u, ok, nil
	case "orderbook_delta":
		u, ok := parseDelta(frame.Msg, frame.Seq)
		return u, ok, nil
	case "error":
		return Update{}, false, errFrame(frame.Msg)
	default:
		return Update{}, false, nil
	}
}

func parseSnapshot(raw json.RawMessage, seq int64) (Update, bool) {
	var body struct {
		MarketTicker string     `json:"market_ticker"`
		Yes          [][]string `json:"yes_dollars"`
		No           [][]string `json:"no_dollars"`
		YesFP        [][]string `json:"yes_dollars_fp"`
		NoFP         [][]string `json:"no_dollars_fp"`
		YesCents     [][]any    `json:"yes"`
		NoCents      [][]any    `json:"no"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return Update{}, false
	}
	ticker := strings.TrimSpace(body.MarketTicker)
	if ticker == "" {
		return Update{}, false
	}
	yes := firstLevels(body.YesFP, body.Yes, body.YesCents)
	no := firstLevels(body.NoFP, body.No, body.NoCents)
	return Update{Ticker: ticker, Snapshot: true, Seq: seq, Yes: yes, No: no}, true
}

func parseDelta(raw json.RawMessage, seq int64) (Update, bool) {
	var body struct {
		MarketTicker string  `json:"market_ticker"`
		PriceDollars string  `json:"price_dollars"`
		Price        float64 `json:"price"`
		DeltaFP      string  `json:"delta_fp"`
		Delta        float64 `json:"delta"`
		Side         string  `json:"side"`
		Ts           int64   `json:"ts"`
		TsMs         int64   `json:"ts_ms"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return Update{}, false
	}
	ticker := strings.TrimSpace(body.MarketTicker)
	if ticker == "" {
		return Update{}, false
	}
	px := book.ParsePrice(body.PriceDollars)
	if px == 0 && body.Price > 0 {
		if body.Price > 1 {
			px = body.Price / 100
		} else {
			px = body.Price
		}
	}
	delta := body.Delta
	if strings.TrimSpace(body.DeltaFP) != "" {
		delta = book.ParseNum(body.DeltaFP)
	}
	return Update{
		Ticker: ticker,
		Seq:    seq,
		Side:   body.Side,
		Price:  px,
		Delta:  delta,
		TS:     timestamp(body.TsMs, body.Ts),
	}, true
}

func firstLevels(fp, dollars [][]string, cents [][]any) []book.Level {
	if len(fp) > 0 {
		return levelsFromPairs(fp)
	}
	if len(dollars) > 0 {
		return levelsFromPairs(dollars)
	}
	return levelsFromAny(cents)
}

func levelsFromPairs(rows [][]string) []book.Level {
	out := make([]book.Level, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		out = append(out, book.Level{Price: book.ParsePrice(row[0]), Size: book.ParseNum(row[1])})
	}
	return out
}

func levelsFromAny(rows [][]any) []book.Level {
	out := make([]book.Level, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		px, ok1 := numAny(row[0])
		sz, ok2 := numAny(row[1])
		if !ok1 || !ok2 {
			continue
		}
		if px > 1 {
			px = px / 100
		}
		out = append(out, book.Level{Price: px, Size: sz})
	}
	return out
}

func numAny(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case string:
		return book.ParseNum(x), true
	default:
		return 0, false
	}
}

func timestamp(ms, sec int64) time.Time {
	if ms > 0 {
		return time.UnixMilli(ms)
	}
	if sec > 1_000_000_000_000 {
		return time.UnixMilli(sec)
	}
	if sec > 0 {
		return time.Unix(sec, 0)
	}
	return time.Time{}
}

func errFrame(msg json.RawMessage) error {
	if len(msg) == 0 {
		return errString("kalshi websocket error")
	}
	return errString("kalshi websocket error: " + string(msg))
}

type errString string

func (e errString) Error() string { return string(e) }
