package sxbet

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

// Message watch 层盘口事件。
type Message struct {
	Gen      int
	Kind     string // book | pause | heartbeat
	ID       string // marketHash
	Version  string
	Bids     []book.Level
	Asks     []book.Level
	TS       time.Time
}

// MessageFromSnapshot REST 种子转 Message。
func MessageFromSnapshot(gen int, ob *httpentity.SxbetOrderbookV3) (Message, bool) {
	return messageFromBook(gen, ob)
}

func messageFromBook(gen int, ob *httpentity.SxbetOrderbookV3) (Message, bool) {
	if ob == nil {
		return Message{}, false
	}
	hash := strings.TrimSpace(ob.MarketHash)
	if hash == "" {
		return Message{}, false
	}
	bids, asks := LevelsToYesBook(ob.OutcomeOne, ob.OutcomeTwo)
	return Message{
		Gen: gen, Kind: "book", ID: hash, Version: strings.TrimSpace(ob.Version),
		Bids: bids, Asks: asks, TS: time.Now().UTC(),
	}, true
}

func parseOrderbookV3(data []byte) (*httpentity.SxbetOrderbookV3, error) {
	var ob httpentity.SxbetOrderbookV3
	if err := json.Unmarshal(data, &ob); err != nil {
		return nil, err
	}
	if strings.TrimSpace(ob.MarketHash) == "" {
		var wrap struct {
			Data httpentity.SxbetOrderbookV3 `json:"data"`
		}
		if err := json.Unmarshal(data, &wrap); err == nil && strings.TrimSpace(wrap.Data.MarketHash) != "" {
			return &wrap.Data, nil
		}
	}
	return &ob, nil
}