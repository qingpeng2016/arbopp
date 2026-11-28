package strategyentry

import (
	"context"
	"fmt"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/noderisk"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/noderelay"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/noderoute"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

func (e *Entry) NodeProfile() *noderoute.Profile {
	if e == nil {
		return nil
	}
	v := e.nodeProf.Load()
	if v == nil {
		return nil
	}
	p, _ := v.(*noderoute.Profile)
	return p
}

func (e *Entry) Relay() *noderelay.Hub {
	if e == nil {
		return nil
	}
	return e.relay
}

func (e *Entry) initNodeMode(ctx context.Context) error {
	if e == nil {
		return fmt.Errorf("strategy_entry: entry nil")
	}
	if conf.IsNodeAllMode() {
		return nil
	}
	node := conf.GetRunNode()
	if e.nodeCfgRepo == nil || e.nodeRouteRepo == nil {
		return fmt.Errorf("node %s: node_config/node_route repo not wired", node)
	}
	prof, err := noderoute.Load(ctx, e.nodeCfgRepo, e.nodeRouteRepo)
	if err != nil {
		return fmt.Errorf("node %s: load route: %w", node, err)
	}
	if prof == nil {
		return fmt.Errorf("node %s: empty profile", node)
	}
	if len(prof.Edges) == 0 {
		return fmt.Errorf("node %s: no enabled outbound routes", node)
	}
	e.nodeProf.Store(prof)
	e.relayRisk = noderisk.NewGuard(ctx, prof, e.thresholdRepo, node)
	if e.relayRisk != nil {
		e.relayRisk.Start()
	}
	e.startRelayBookPushLoop(ctx)
	e.relay = noderelay.NewHub(ctx, prof, e.onRelayBook, e.onRelayHedge, e.relayRisk)
	if e.relay == nil {
		e.nodeProf.Store(nil)
		return fmt.Errorf("node %s: relay hub not started", node)
	}
	_ = notification.GetGlobalNotificationManager().SendInfoAlert(ctx, "node-relay-started", []notification.FieldPair{
		{Key: "node", Value: prof.NodeCode},
		{Key: "local", Value: prof.LocalPlatform},
		{Key: "listen", Value: prof.RelayListen},
	})
	return nil
}

func (e *Entry) onRelayBook(p noderelay.BookPayload) {
	if e == nil {
		return
	}
	if e.relayRisk != nil {
		e.relayRisk.RecordMirrorBook(p.Platform)
	}
	mapKey := strings.TrimSpace(p.MapKey)
	if mapKey == "" {
		return
	}
	e.mu.Lock()
	r := e.watches[mapKey]
	e.mu.Unlock()
	if r == nil || r.ws == nil {
		return
	}
	r.ws.ingestMirrorBook(p)
}

func (e *Entry) onRelayHedge(cmd noderelay.HedgeCmd) error {
	if e == nil {
		return fmt.Errorf("entry nil")
	}
	mapKey := strings.TrimSpace(cmd.MapKey)
	e.mu.Lock()
	r := e.watches[mapKey]
	e.mu.Unlock()
	if r == nil || r.ws == nil {
		return fmt.Errorf("no watch for map_key=%s", mapKey)
	}
	idx := -1
	for i := range r.ws.Outcomes {
		if r.ws.Outcomes[i].ID == cmd.OutcomeID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("outcome_id %d not in watch", cmd.OutcomeID)
	}
	prof := e.NodeProfile()
	if prof != nil && !prof.IsLocalPlatform(cmd.Platform) {
		return fmt.Errorf("hedge platform %s not local on this node", cmd.Platform)
	}
	res, err := r.ws.sendOnOutcome(idx, cmd.BuyYes, cmd.Px, cmd.Qty)
	persist := yesno.PersistGrab{
		OppRepo:            e.oppRepo,
		OrderRepo:          e.orderRepo,
		OnOrdersWritten:    e.MarkMapKeyOpenOrders,
		ScopeForOutcomeIDs: r.ws.ScopeForOutcomeIDs,
	}
	yesno.SaveInboundHedgeAsync(persist, mapKey, cmd, res, err)
	return err
}

func (e *Entry) stopRelay() {
	if e == nil {
		return
	}
	if e.relay != nil {
		e.relay.Stop()
		e.relay = nil
	}
	if e.relayRisk != nil {
		e.relayRisk.Stop()
		e.relayRisk = nil
	}
	e.nodeProf.Store(nil)
}