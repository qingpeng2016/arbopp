package predict

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
)

const PredictWSURL = "wss://ws.predict.fun/ws"

// Message 一条 Predict WS 推送（归一后）。
type Message struct {
	Gen  int
	Kind string // pause | book | heartbeat
	ID   string // market id
	Bids []book.Level
	Asks []book.Level
	TS   time.Time
}

type wireEnvelope struct {
	Type      string          `json:"type"`
	Topic     string          `json:"topic"`
	RequestID int             `json:"requestId"`
	Success   *bool           `json:"success"`
	Data      json.RawMessage `json:"data"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type wireBook struct {
	MarketID          int64       `json:"marketId"`
	UpdateTimestampMs int64       `json:"updateTimestampMs"`
	Asks              [][]float64 `json:"asks"`
	Bids              [][]float64 `json:"bids"`
}

func Parse(msg []byte) (kind string, bookMsg *Message, heartbeatData int64, err error) {
	var env wireEnvelope
	if err := json.Unmarshal(msg, &env); err != nil {
		return "", nil, 0, err
	}
	switch strings.ToLower(strings.TrimSpace(env.Type)) {
	case "m":
		topic := strings.TrimSpace(env.Topic)
		if topic == "heartbeat" {
			var hb int64
			if err := json.Unmarshal(env.Data, &hb); err == nil {
				return "heartbeat", nil, hb, nil
			}
			return "heartbeat", nil, 0, nil
		}
		if strings.HasPrefix(topic, "predictOrderbook/") {
			var wb wireBook
			if err := json.Unmarshal(env.Data, &wb); err != nil {
				return "", nil, 0, err
			}
			id := strconv.FormatInt(wb.MarketID, 10)
			if id == "0" {
				id = strings.TrimPrefix(topic, "predictOrderbook/")
			}
			ts := time.Now().UTC()
			if wb.UpdateTimestampMs > 0 {
				ts = time.UnixMilli(wb.UpdateTimestampMs).UTC()
			}
			return "book", &Message{
				Kind: "book",
				ID:   id,
				Bids: LevelsFrom(wb.Bids),
				Asks: LevelsFrom(wb.Asks),
				TS:   ts,
			}, 0, nil
		}
		return "", nil, 0, nil
	case "r":
		if env.Success != nil && !*env.Success && env.Error != nil {
			return "error", nil, 0, errString(env.Error.Message)
		}
		return "", nil, 0, nil
	default:
		return "", nil, 0, nil
	}
}

// LevelsFrom 把 REST/WS 的 [price, size] 转成内存档位。
func LevelsFrom(rows [][]float64) []book.Level {
	out := make([]book.Level, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		out = append(out, book.Level{Price: row[0], Size: row[1]})
	}
	return out
}

type errString string

func (e errString) Error() string { return string(e) }
