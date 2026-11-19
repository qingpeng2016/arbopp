package polymarket

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
)

// Message 一条 market channel 推送。Kind 为 book 或 price_change。多档在 Changes。
type Message struct {
	Gen     int
	Kind    string
	ID      string
	Bids    []book.Level
	Asks    []book.Level
	Tick    float64
	MinSize float64
	TS      time.Time
	Changes []book.PolyChange
}

func Parse(raw []byte) ([]Message, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || strings.EqualFold(string(raw), "PONG") || strings.EqualFold(string(raw), "PING") {
		return nil, nil
	}
	if raw[0] == '[' {
		var arr []json.RawMessage
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil, err
		}
		out := make([]Message, 0, len(arr))
		for _, el := range arr {
			msg, ok, err := parseOne(el)
			if err != nil {
				return nil, err
			}
			if ok {
				out = append(out, msg)
			}
		}
		return out, nil
	}
	msg, ok, err := parseOne(raw)
	if err != nil || !ok {
		return nil, err
	}
	return []Message{msg}, nil
}

func parseOne(raw []byte) (Message, bool, error) {
	var head struct {
		EventType string `json:"event_type"`
		Type      string `json:"type"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return Message{}, false, err
	}
	kind := strings.ToLower(strings.TrimSpace(head.EventType))
	if kind == "" {
		kind = strings.ToLower(strings.TrimSpace(head.Type))
	}
	switch kind {
	case "book":
		return parseBook(raw)
	case "price_change":
		return parsePriceChange(raw)
	default:
		return Message{}, false, nil
	}
}

func parseBook(raw []byte) (Message, bool, error) {
	var body struct {
		AssetID      string          `json:"asset_id"`
		Bids         json.RawMessage `json:"bids"`
		Asks         json.RawMessage `json:"asks"`
		Timestamp    string          `json:"timestamp"`
		TickSize     string          `json:"tick_size"`
		MinOrderSize string          `json:"min_order_size"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return Message{}, false, err
	}
	if strings.TrimSpace(body.AssetID) == "" {
		return Message{}, false, nil
	}
	return Message{
		Kind:    "book",
		ID:      strings.TrimSpace(body.AssetID),
		Bids:    parseLevels(body.Bids),
		Asks:    parseLevels(body.Asks),
		TS:      parseTS(body.Timestamp),
		Tick:    book.ParseNum(body.TickSize),
		MinSize: book.ParseNum(body.MinOrderSize),
	}, true, nil
}

func parsePriceChange(raw []byte) (Message, bool, error) {
	var body struct {
		Timestamp    string `json:"timestamp"`
		AssetID      string `json:"asset_id"`
		Price        string `json:"price"`
		Size         string `json:"size"`
		Side         string `json:"side"`
		PriceChanges []struct {
			AssetID string `json:"asset_id"`
			Price   string `json:"price"`
			Size    string `json:"size"`
			Side    string `json:"side"`
		} `json:"price_changes"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return Message{}, false, err
	}
	msg := Message{Kind: "price_change", TS: parseTS(body.Timestamp)}
	if len(body.PriceChanges) == 0 && strings.TrimSpace(body.AssetID) != "" {
		msg.Changes = []book.PolyChange{{
			ID:    strings.TrimSpace(body.AssetID),
			Side:  body.Side,
			Price: book.ParsePrice(body.Price),
			Size:  book.ParseNum(body.Size),
		}}
		return msg, true, nil
	}
	msg.Changes = make([]book.PolyChange, 0, len(body.PriceChanges))
	for _, ch := range body.PriceChanges {
		id := strings.TrimSpace(ch.AssetID)
		if id == "" {
			continue
		}
		msg.Changes = append(msg.Changes, book.PolyChange{
			ID:    id,
			Side:  ch.Side,
			Price: book.ParsePrice(ch.Price),
			Size:  book.ParseNum(ch.Size),
		})
	}
	if len(msg.Changes) == 0 {
		return Message{}, false, nil
	}
	return msg, true, nil
}

func parseLevels(raw json.RawMessage) []book.Level {
	if len(raw) == 0 {
		return nil
	}
	var objs []struct {
		Price string `json:"price"`
		Size  string `json:"size"`
	}
	if err := json.Unmarshal(raw, &objs); err == nil && len(objs) > 0 && objs[0].Price != "" {
		out := make([]book.Level, 0, len(objs))
		for _, o := range objs {
			out = append(out, book.Level{Price: book.ParsePrice(o.Price), Size: book.ParseNum(o.Size)})
		}
		return out
	}
	var rows [][]string
	if err := json.Unmarshal(raw, &rows); err == nil {
		out := make([]book.Level, 0, len(rows))
		for _, row := range rows {
			if len(row) < 2 {
				continue
			}
			out = append(out, book.Level{Price: book.ParsePrice(row[0]), Size: book.ParseNum(row[1])})
		}
		return out
	}
	return nil
}

func parseTS(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	n := book.ParseNum(s)
	if n <= 0 {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t
		}
		return time.Time{}
	}
	if n < 1e12 {
		return time.Unix(int64(n), 0)
	}
	ms := int64(n)
	return time.UnixMilli(ms)
}
