package strategyentry

import (
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
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

// maybeRelayBook 本机官方 WS 更新内存簿后，按当前 Books 全量扇出（非仅 *_all_book 原始包）。
func (w *watch) maybeRelayBook(e event) {
	if w == nil || w.entry == nil || e.Mirror {
		return
	}
	hub := w.entry.Relay()
	prof := w.entry.NodeProfile()
	if hub == nil || prof == nil || !prof.IsLocalPlatform(e.Platform) {
		return
	}
	for _, p := range w.buildRelayPayloadsFromMemory(e) {
		hub.PublishBook(p)
	}
}

func relayKindForPlatform(platform string) string {
	switch strings.ToUpper(strings.TrimSpace(platform)) {
	case appcommon.PlatformPolymarket:
		return "poly_all_book"
	case appcommon.PlatformPredict:
		return "predict_all_book"
	case appcommon.PlatformSxbet:
		return "sxbet_all_book"
	case appcommon.PlatformKalshi:
		return "kalshi_all_book"
	default:
		return ""
	}
}

// relayBookIDs 本条 WS 影响到的 book id（token / hash / ticker 等）。
func (w *watch) relayBookIDs(e event) []string {
	var ids []string
	seen := map[string]struct{}{}
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, dup := seen[id]; dup {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	switch e.Kind {
	case "poly_price_change":
		for _, ch := range e.Changes {
			add(ch.ID)
			if no := w.polyNoTokenForYes(ch.ID); no != "" {
				add(no)
			}
		}
	case "poly_all_book":
		add(e.ID)
		if no := w.polyNoTokenForYes(e.ID); no != "" {
			add(no)
		}
	case "sxbet_all_book":
		add(e.ID)
	case "kalshi_all_book", "kalshi_price_change":
		add(e.ID)
	case "predict_all_book":
		add(e.ID)
	default:
		add(e.ID)
	}
	return ids
}

// relayPushAllMemoryBooks 定时主动扇出本机官方平台当前内存全量簿。
func (w *watch) relayPushAllMemoryBooks(localPlatform string) {
	if w == nil || w.entry == nil {
		return
	}
	hub := w.entry.Relay()
	prof := w.entry.NodeProfile()
	if hub == nil || prof == nil || !prof.IsLocalPlatform(localPlatform) {
		return
	}
	for _, p := range w.buildRelayPayloadsAllMemory(localPlatform) {
		hub.PublishBook(p)
	}
}

func (w *watch) buildRelayPayloadsAllMemory(platform string) []noderelay.BookPayload {
	platform = strings.ToUpper(strings.TrimSpace(platform))
	kind := relayKindForPlatform(platform)
	if kind == "" {
		return nil
	}
	var out []noderelay.BookPayload
	snap := w.Books.Snapshot()
	prefix := platform + "|"
	for k, b := range snap.books {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		id := strings.TrimPrefix(k, prefix)
		if id == "" || strings.Contains(id, ":not") {
			continue
		}
		if !b.Live && len(b.Bids) == 0 && len(b.Asks) == 0 && len(b.NoBids) == 0 {
			continue
		}
		tsMs := time.Now().UnixMilli()
		if !b.ExchangeTS.IsZero() {
			tsMs = b.ExchangeTS.UnixMilli()
		}
		p := noderelay.BookPayload{
			MapKey:   w.MapKey,
			Platform: platform,
			Kind:     kind,
			ID:       id,
			Bids:     bookToRelayLevels(b.Bids),
			Asks:     bookToRelayLevels(b.Asks),
			NoBids:   bookToRelayLevels(b.NoBids),
			Tick:     b.Tick,
			MinSize:  b.MinSize,
			TSUnixMs: tsMs,
		}
		if platform == appcommon.PlatformSxbet {
			if bNo, ok := snap.Get(platform, id+":not"); ok && len(bNo.Asks) > 0 {
				p.NotAsks = bookToRelayLevels(bNo.Asks)
			}
		}
		out = append(out, p)
	}
	return out
}

func (w *watch) buildRelayPayloadsFromMemory(e event) []noderelay.BookPayload {
	platform := strings.ToUpper(strings.TrimSpace(e.Platform))
	kind := relayKindForPlatform(platform)
	if kind == "" {
		return nil
	}
	var out []noderelay.BookPayload
	tsMs := e.TS.UnixMilli()

	for _, id := range w.relayBookIDs(e) {
		b, ok := w.Books.Get(platform, id)
		if !ok {
			continue
		}
		if !b.Live && len(b.Bids) == 0 && len(b.Asks) == 0 && len(b.NoBids) == 0 {
			continue
		}
		if !b.ExchangeTS.IsZero() {
			tsMs = b.ExchangeTS.UnixMilli()
		}
		p := noderelay.BookPayload{
			MapKey:   w.MapKey,
			Platform: platform,
			Kind:     kind,
			ID:       id,
			Bids:     bookToRelayLevels(b.Bids),
			Asks:     bookToRelayLevels(b.Asks),
			NoBids:   bookToRelayLevels(b.NoBids),
			Tick:     b.Tick,
			MinSize:  b.MinSize,
			TSUnixMs: tsMs,
		}
		if platform == appcommon.PlatformSxbet && !strings.Contains(id, ":not") {
			if bNo, ok2 := w.Books.Get(platform, id+":not"); ok2 && len(bNo.Asks) > 0 {
				p.NotAsks = bookToRelayLevels(bNo.Asks)
			} else if len(e.NotAsks) > 0 && strings.TrimSpace(e.ID) == id {
				p.NotAsks = bookToRelayLevels(e.NotAsks)
			}
		}
		out = append(out, p)
	}
	return out
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