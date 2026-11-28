package noderelay

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/noderoute"
)

// BookHandler applies mirror book on this node.
type BookHandler func(BookPayload)

// HedgeHandler executes hedge leg locally (broker).
type HedgeHandler func(HedgeCmd) error

// Hub TCP relay: listen + dial peers (1→N), dial retry if peer starts late.
type Hub struct {
	ctx    context.Context
	cancel context.CancelFunc
	prof   *noderoute.Profile
	onBook BookHandler
	onHedge HedgeHandler

	mu    sync.Mutex
	conns map[string]*peerConn // publish target addr → conn

	seenHedge sync.Map // arb_id → struct{}
}

type peerConn struct {
	addr string
	mu   sync.Mutex
	conn net.Conn
}

func NewHub(parent context.Context, prof *noderoute.Profile, onBook BookHandler, onHedge HedgeHandler) *Hub {
	if prof == nil {
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	h := &Hub{
		ctx:     ctx,
		cancel:  cancel,
		prof:    prof,
		onBook:  onBook,
		onHedge: onHedge,
		conns:   map[string]*peerConn{},
	}
	for _, e := range prof.Edges {
		addr := strings.TrimSpace(e.PublishTo)
		if addr == "" {
			continue
		}
		h.conns[addr] = &peerConn{addr: addr}
		go h.dialLoop(addr)
	}
	go h.listen()
	go h.pingLoop()
	return h
}

func (h *Hub) Stop() {
	if h == nil || h.cancel == nil {
		return
	}
	h.cancel()
}

func (h *Hub) listen() {
	addr := strings.TrimSpace(h.prof.RelayListen)
	if addr == "" {
		addr = ":9000"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}
	go func() {
		<-h.ctx.Done()
		_ = ln.Close()
	}()
	for {
		c, err := ln.Accept()
		if err != nil {
			select {
			case <-h.ctx.Done():
				return
			default:
				continue
			}
		}
		go h.serveConn(c)
	}
}

func (h *Hub) serveConn(c net.Conn) {
	defer c.Close()
	_ = setTCPNoDelay(c)
	for {
		select {
		case <-h.ctx.Done():
			return
		default:
		}
		raw, err := readFrame(c)
		if err != nil {
			return
		}
		var kind Envelope
		if err := json.Unmarshal(raw, &kind); err != nil {
			continue
		}
		switch strings.ToUpper(kind.Type) {
		case "PING":
			continue
		case "BOOK":
			var p BookPayload
			if err := json.Unmarshal(raw, &p); err != nil || h.onBook == nil {
				continue
			}
			h.onBook(p)
		case "HEDGE":
			var cmd HedgeCmd
			if err := json.Unmarshal(raw, &cmd); err != nil || h.onHedge == nil {
				continue
			}
			if id := strings.TrimSpace(cmd.ArbID); id != "" {
				if _, dup := h.seenHedge.LoadOrStore(id, struct{}{}); dup {
					continue
				}
			}
			_ = h.onHedge(cmd)
		}
	}
}

func (h *Hub) dialLoop(addr string) {
	backoff := time.Second
	for {
		select {
		case <-h.ctx.Done():
			return
		default:
		}
		d := net.Dialer{Timeout: 5 * time.Second}
		c, err := d.DialContext(h.ctx, "tcp", addr)
		if err != nil {
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		_ = setTCPNoDelay(c)
		h.setConn(addr, c)
		h.serveConn(c)
		h.setConn(addr, nil)
	}
}

func (h *Hub) setConn(addr string, c net.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	pc, ok := h.conns[addr]
	if !ok {
		pc = &peerConn{addr: addr}
		h.conns[addr] = pc
	}
	pc.mu.Lock()
	if pc.conn != nil {
		_ = pc.conn.Close()
	}
	pc.conn = c
	pc.mu.Unlock()
}

func (h *Hub) pingLoop() {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-h.ctx.Done():
			return
		case <-t.C:
			h.broadcast(map[string]any{"type": "PING", "ts_ms": time.Now().UnixMilli()})
		}
	}
}

// PublishBook fan-out to all edges (1→N).
func (h *Hub) PublishBook(p BookPayload) {
	if h == nil {
		return
	}
	p.Type = "BOOK"
	h.broadcast(p)
}

// SendHedge sends leg to platform's peer edge.
func (h *Hub) SendHedge(platform string, cmd HedgeCmd) error {
	if h == nil {
		return errors.New("relay hub nil")
	}
	edge := h.prof.EdgeForPlatform(platform)
	if edge == nil {
		return errors.New("no relay edge for platform")
	}
	addr := strings.TrimSpace(edge.HedgeTo)
	if addr == "" {
		addr = strings.TrimSpace(edge.PublishTo)
	}
	cmd.Type = "HEDGE"
	return h.writeAddr(addr, cmd)
}

func (h *Hub) broadcast(v any) {
	h.mu.Lock()
	addrs := make([]string, 0, len(h.conns))
	for a := range h.conns {
		addrs = append(addrs, a)
	}
	h.mu.Unlock()
	for _, a := range addrs {
		_ = h.writeAddr(a, v)
	}
}

func (h *Hub) writeAddr(addr string, v any) error {
	h.mu.Lock()
	pc, ok := h.conns[addr]
	h.mu.Unlock()
	if !ok || pc == nil {
		return errors.New("peer not configured")
	}
	pc.mu.Lock()
	c := pc.conn
	pc.mu.Unlock()
	if c == nil {
		return errors.New("peer not connected")
	}
	return writeFrame(c, v)
}

func setTCPNoDelay(c net.Conn) error {
	if tc, ok := c.(*net.TCPConn); ok {
		return tc.SetNoDelay(true)
	}
	return nil
}