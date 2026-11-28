package strategyentry

import (
	"context"
	"time"
)

func (e *Entry) startRelayBookPushLoop(ctx context.Context) {
	if e == nil || e.NodeProfile() == nil || e.Relay() == nil {
		return
	}
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
			iv := 5 * time.Second
			if e.relayRisk != nil {
				iv = e.relayRisk.Config().BookPushInterval
			}
			if iv <= 0 {
				iv = 5 * time.Second
			}
			t := time.NewTimer(iv)
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
				e.relayPushAllMemoryBooks()
			}
		}
	}()
}

func (e *Entry) relayPushAllMemoryBooks() {
	if e == nil {
		return
	}
	prof := e.NodeProfile()
	if prof == nil || e.relay == nil {
		return
	}
	local := prof.LocalPlatform
	e.mu.Lock()
	list := make([]*watch, 0, len(e.watches))
	for _, r := range e.watches {
		if r != nil && r.ws != nil {
			list = append(list, r.ws)
		}
	}
	e.mu.Unlock()
	for _, w := range list {
		w.relayPushAllMemoryBooks(local)
	}
}