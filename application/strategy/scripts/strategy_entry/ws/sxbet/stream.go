package sxbet

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/centrifugal/centrifuge-go"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	"go.uber.org/zap"
)

const realtimeWSURL = "wss://realtime.sx.bet/connection/websocket"

// Stream 订阅 orderbook_v3:{marketHash}，按 version 整本替换。
func Stream(ctx context.Context, api httprepo.SxbetRepo, marketHashes []string, gen int, apply func(Message)) error {
	if api == nil {
		return fmt.Errorf("sxbet ws: nil api")
	}
	if strings.TrimSpace(api.APIKey()) == "" {
		return fmt.Errorf("sxbet ws: missing api key")
	}
	if len(marketHashes) == 0 {
		return fmt.Errorf("sxbet ws: no market hashes")
	}
	client := centrifuge.NewJsonClient(realtimeWSURL, centrifuge.Config{
		GetToken: func(centrifuge.ConnectionTokenEvent) (string, error) {
			return api.GetRealtimeToken(ctx)
		},
	})
	var versionsMu sync.Mutex
	versions := map[string]string{}

	applyBook := func(hash string, data []byte) {
		ob, err := parseOrderbookV3(data)
		if err != nil || ob == nil {
			return
		}
		if strings.TrimSpace(ob.MarketHash) == "" {
			ob.MarketHash = hash
		}
		ver := strings.TrimSpace(ob.Version)
		if ver == "" {
			return
		}
		versionsMu.Lock()
		cur := versions[hash]
		if cur != "" && ver <= cur {
			versionsMu.Unlock()
			return
		}
		versions[hash] = ver
		versionsMu.Unlock()
		msg, ok := messageFromBook(gen, ob)
		if !ok {
			return
		}
		apply(msg)
	}

	seedFromREST := func(hash string) {
		snap, err := api.GetOrderbookSnapshot(ctx, hash)
		if err != nil || snap == nil {
			return
		}
		ver := strings.TrimSpace(snap.Version)
		if ver != "" {
			versionsMu.Lock()
			if cur := versions[hash]; cur == "" || ver > cur {
				versions[hash] = ver
			}
			versionsMu.Unlock()
		}
		msg, ok := messageFromBook(gen, snap)
		if !ok {
			return
		}
		apply(msg)
	}

	client.OnConnected(func(e centrifuge.ConnectedEvent) {
		apply(Message{Gen: gen, Kind: "heartbeat"})
		_ = e
	})
	client.OnError(func(e centrifuge.ErrorEvent) {
		logger.WarnZ(ctx, "strategy_entry-sxbet-ws-client-error", zap.Error(e.Error))
	})

	if err := client.Connect(); err != nil {
		client.Close()
		return err
	}
	defer client.Close()

	subCfg := centrifuge.SubscriptionConfig{Recoverable: true}

	for _, h := range marketHashes {
		hash := strings.TrimSpace(h)
		if hash == "" {
			continue
		}
		ch := "orderbook_v3:" + hash
		sub, err := client.NewSubscription(ch, subCfg)
		if err != nil {
			return err
		}
		hCopy := hash
		sub.OnPublication(func(e centrifuge.PublicationEvent) {
			applyBook(hCopy, e.Data)
		})
		sub.OnSubscribed(func(e centrifuge.SubscribedEvent) {
			apply(Message{Gen: gen, Kind: "heartbeat"})
			if e.Recovered {
				return
			}
			seedFromREST(hCopy)
		})
		sub.OnError(func(e centrifuge.SubscriptionErrorEvent) {
			logger.WarnZ(ctx, "strategy_entry-sxbet-ws-sub-error",
				zap.String("channel", ch),
				zap.Error(e.Error))
		})
		if err := sub.Subscribe(); err != nil {
			return err
		}
	}

	<-ctx.Done()
	return ctx.Err()
}