package noderelay

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

const maxFrame = 4 << 20

type Envelope struct {
	Type string `json:"type"`
}

// BookPayload mirror book push (official WS → TCP peers).
type BookPayload struct {
	Type     string  `json:"type"`
	MapKey   string  `json:"map_key"`
	Platform string  `json:"platform"`
	Kind     string  `json:"kind"`
	ID       string  `json:"id"`
	Bids     []Level `json:"bids,omitempty"`
	Asks     []Level `json:"asks,omitempty"`
	NotAsks  []Level `json:"not_asks,omitempty"`
	NoBids   []Level `json:"no_bids,omitempty"`
	Tick     float64 `json:"tick,omitempty"`
	MinSize  float64 `json:"min_size,omitempty"`
	TSUnixMs int64   `json:"ts_unix_ms,omitempty"`
}

type Level struct {
	Price float64 `json:"p"`
	Size  float64 `json:"s"`
}

// HedgeCmd cross-node leg (peer executes local broker).
type HedgeCmd struct {
	Type      string  `json:"type"`
	ArbID     string  `json:"arb_id,omitempty"`
	MapKey    string  `json:"map_key"`
	Platform  string  `json:"platform"`
	OutcomeID uint    `json:"outcome_id"`
	BuyYes    bool    `json:"buy_yes"`
	Px        float64 `json:"px"`
	Qty       float64 `json:"qty"`
}

func writeFrame(w io.Writer, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(raw) > maxFrame {
		return io.ErrShortBuffer
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(raw)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err = w.Write(raw)
	return err
}

func readFrame(r io.Reader) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 0 || n > maxFrame {
		return nil, io.ErrUnexpectedEOF
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}