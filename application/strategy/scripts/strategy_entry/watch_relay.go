package strategyentry

import (
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/noderelay"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
)

func (w *watch) ingestMirrorBook(p noderelay.BookPayload) {
	if w == nil {
		return
	}
	if mk := strings.TrimSpace(p.MapKey); mk != "" && mk != w.MapKey {
		return
	}
	e := event{
		Mirror:   true,
		Platform: strings.ToUpper(strings.TrimSpace(p.Platform)),
		Kind:     strings.TrimSpace(p.Kind),
		ID:       strings.TrimSpace(p.ID),
		Bids:     relayToBookLevels(p.Bids),
		Asks:     relayToBookLevels(p.Asks),
		NotAsks:  relayToBookLevels(p.NotAsks),
		NoBids:   relayToBookLevels(p.NoBids),
		Tick:     p.Tick,
		MinSize:  p.MinSize,
		RecvAt:   time.Now(),
	}
	if p.TSUnixMs > 0 {
		e.TS = time.UnixMilli(p.TSUnixMs)
	}
	w.write(e)
}

func (w *watch) maybeRelayBook(e event) {
	if w == nil || w.entry == nil || e.Mirror {
		return
	}
	hub := w.entry.Relay()
	prof := w.entry.NodeProfile()
	if hub == nil || prof == nil || !prof.IsLocalPlatform(e.Platform) {
		return
	}
	switch e.Kind {
	case "poly_all_book", "predict_all_book", "sxbet_all_book", "kalshi_all_book":
	default:
		return
	}
	hub.PublishBook(noderelay.BookPayload{
		MapKey:   w.MapKey,
		Platform: e.Platform,
		Kind:     e.Kind,
		ID:       e.ID,
		Bids:     bookToRelayLevels(e.Bids),
		Asks:     bookToRelayLevels(e.Asks),
		NotAsks:  bookToRelayLevels(e.NotAsks),
		NoBids:   bookToRelayLevels(e.NoBids),
		Tick:     e.Tick,
		MinSize:  e.MinSize,
		TSUnixMs: e.TS.UnixMilli(),
	})
}

func bookToRelayLevels(in []book.Level) []noderelay.Level {
	if len(in) == 0 {
		return nil
	}
	out := make([]noderelay.Level, len(in))
	for i := range in {
		out[i] = noderelay.Level{Price: in[i].Price, Size: in[i].Size}
	}
	return out
}

func relayToBookLevels(in []noderelay.Level) []book.Level {
	if len(in) == 0 {
		return nil
	}
	out := make([]book.Level, len(in))
	for i := range in {
		out[i] = book.Level{Price: in[i].Price, Size: in[i].Size}
	}
	return out
}