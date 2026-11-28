package noderelay

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"sync/atomic"
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
	health PeerHealth

	mu    sync.Mutex
	conns map[string]*peerConn // publish target addr → conn

	pingSeq     atomic.Int64
	pingPending sync.Map // addr -> time.Time

	seenHedge sync.Map // arb_id → struct{}
}

type peerConn struct {
	addr string
	mu   sync.Mutex
	conn net.Conn
}

func NewHub(parent context.Context, prof *noderoute.Profile, onBook BookHandler, onHedge HedgeHandler, health PeerHealth) *Hub {
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
		health:  health,
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
	remote := peerRemoteKey(c)
	if h.health != nil {
		h.health.OnPeerConnUp(remote)
		defer h.health.OnPeerConnDown(remote)
	}
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
			var ping struct {
				Type string `json:"type"`
				Seq  int64  `json:"seq"`
				TSMs int64  `json:"ts_ms"`
			}
			if err := json.Unmarshal(raw, &ping); err != nil {
				continue
			}
			_ = writeFrame(c, map[string]any{
				"type":   "PONG",
				"seq":    ping.Seq,
				"ts_ms":  ping.TSMs,
				"echo_ms": time.Now().UnixMilli(),
			})
		case "PONG":
			var pong struct {
				Type string `json:"type"`
				Seq  int64  `json:"seq"`
			}
			if err := json.Unmarshal(raw, &pong); err != nil {
				continue
			}
			if sent, pendingKey, ok := h.pingPendingLookup(remote); ok {
				rtt := time.Since(sent)
				h.pingPending.Delete(pendingKey)
				if h.health != nil {
					h.health.OnPeerRTT(remote, rtt)
				}
			}
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

func (h *Hub) pingPendingLookup(remote string) (sent time.Time, pendingKey string, ok bool) {
	if v, hit := h.pingPending.Load(remote); hit {
		if t, ok2 := v.(time.Time); ok2 {
			return t, remote, true
		}
	}
	if h.prof == nil {
		return time.Time{}, "", false
	}
	remote = strings.TrimSpace(remote)
	for _, e := range h.prof.Edges {
		for _, cfg := range []string{e.PublishTo, e.HedgeTo} {
			cfg = strings.TrimSpace(cfg)
			if cfg == "" {
				continue
			}
			if !noderoute.PeerAddrMatch(cfg, remote) {
				continue
			}
			if v, hit := h.pingPending.Load(cfg); hit {
				if t, ok2 := v.(time.Time); ok2 {
					return t, cfg, true
				}
			}
		}
	}
	return time.Time{}, "", false
}

func peerRemoteKey(c net.Conn) string {
	if c == nil || c.RemoteAddr() == nil {
		return ""
	}
	return strings.TrimSpace(c.RemoteAddr().String())
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
		case at := <-t.C:
			seq := h.pingSeq.Add(1)
			msg := map[string]any{"type": "PING", "seq": seq, "ts_ms": at.UnixMilli()}
			h.mu.Lock()
			addrs := make([]string, 0, len(h.conns))
			for a := range h.conns {
				addrs = append(addrs, a)
			}
			h.mu.Unlock()
			for _, a := range addrs {
				if err := h.writeAddr(a, msg); err == nil {
					h.pingPending.Store(a, at)
				}
			}
			// inbound conns不含入 conns map；RTT 主要从来向 PONG 与 dial 方向测量
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

// Connected 出站 publish 地址是否已连接。
func (h *Hub) Connected(addr string) bool {
	if h == nil {
		return false
	}
	addr = strings.TrimSpace(addr)
	h.mu.Lock()
	pc, ok := h.conns[addr]
	h.mu.Unlock()
	if !ok || pc == nil {
		return false
	}
	pc.mu.Lock()
	ok = pc.conn != nil
	pc.mu.Unlock()
	return ok
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