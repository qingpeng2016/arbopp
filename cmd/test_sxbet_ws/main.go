// 独立诊断 SX Bet realtime：API Key → token → 快照 → Centrifugo 订阅。
//
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/test_sxbet_ws -market-id=2329370 -seconds=25
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/test_sxbet_ws -api-key=YOUR_KEY -hash=0x... -seconds=15
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	sxbetws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/sxbet"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	sxbethttp "github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http/crypto/prediction/sxbet"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/mysql"
)

func main() {
	var (
		marketID uint
		hash     string
		seconds  int
		apiKey   string
		baseURL  string
	)
	flag.UintVar(&marketID, "market-id", 0, "market 表 id，从 market_outcome 取 SX hash")
	flag.StringVar(&hash, "hash", "", "marketHash（0x...）")
	flag.IntVar(&seconds, "seconds", 20, "WS 观察时长（秒）")
	flag.StringVar(&apiKey, "api-key", "", "覆盖 platform.extra api_key（默认读库）")
	flag.StringVar(&baseURL, "base-url", "", "覆盖 API 基址（默认读 platform 或 https://api.sx.bet）")
	flag.Parse()

	_ = conf.NewCfg()
	ctx := context.Background()

	var client sxbethttp.Client
	if apiKey != "" {
		client = *sxbethttp.NewClientWithOptions(baseURL, apiKey, nil)
	} else {
		db := boot.NewDBClient()
		catalog, err := boot.NewPlatformCatalog(mysql.NewPlatformImpl(db))
		if err != nil {
			die("platform catalog: %v", err)
		}
		repo := boot.NewSxbetRepoFromPlatform(catalog, mysql.NewAccountImpl(db))
		if repo == nil {
			die("sxbet repo nil")
		}
		if c, ok := repo.(*sxbethttp.Client); ok {
			client = *c
		} else {
			die("sxbet repo type %T", repo)
		}
	}

	key := strings.TrimSpace(client.APIKey())
	fmt.Printf("api_key_present=%v\n", key != "")

	tok, err := client.GetRealtimeToken(ctx)
	if err != nil {
		die("realtime token: %v", err)
	}
	fmt.Printf("realtime_token_ok len=%d prefix=%s…\n", len(tok), trunc(tok, 12))

	if hash == "" && marketID > 0 {
		db := boot.NewDBClient()
		ocRepo := mysql.NewMarketOutcomeImpl(db)
		_, rows, err := ocRepo.List(ctx, nil, map[string]interface{}{
			"market_row_id = ?": marketID,
			"platform = ?":      appcommon.PlatformSxbet,
		}, "", "")
		if err != nil {
			die("load outcomes market_id=%d: %v", marketID, err)
		}
		hashes := sxbetws.MarketHashes(rows)
		if len(hashes) == 0 {
			for _, oc := range rows {
				fmt.Printf("  outcome id=%d platform_market_outcome_id=%q yes_token=%q\n",
					oc.ID, oc.PlatformMarketOutcomeID, oc.YesTokenID)
			}
			die("market-id=%d: 无有效 marketHash（见上列 outcome 字段）", marketID)
		}
		hash = hashes[0]
	}
	hash = strings.TrimSpace(hash)
	if hash == "" {
		die("需要 -hash 或 -market-id")
	}

	snap, err := client.GetOrderbookSnapshot(ctx, hash)
	if err != nil {
		die("snapshot marketHash=%s: %v", hash, err)
	}
	fmt.Printf("snapshot_ok hash=%s version=%s outcome1=%d outcome2=%d\n",
		hash, strings.TrimSpace(snap.Version), len(snap.OutcomeOne), len(snap.OutcomeTwo))

	var books, hb, pause int64
	apply := func(msg sxbetws.Message) {
		switch msg.Kind {
		case "book":
			atomic.AddInt64(&books, 1)
			fmt.Printf("  book hash=%s bids=%d asks=%d version=%s\n", msg.ID, len(msg.Bids), len(msg.Asks), msg.Version)
		case "heartbeat":
			atomic.AddInt64(&hb, 1)
		case "pause":
			atomic.AddInt64(&pause, 1)
		}
	}

	wsCtx, cancel := context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
	defer cancel()
	fmt.Printf("ws_connect hash=%s channel=orderbook_v3:%s (%ds)…\n", hash, hash, seconds)
	err = sxbetws.Stream(wsCtx, &client, []string{hash}, 1, apply)
	if err != nil && wsCtx.Err() == nil {
		die("ws stream: %v", err)
	}
	fmt.Printf("done books=%d heartbeat=%d pause=%d (err=%v)\n", books, hb, pause, err)
	if books+hb == 0 {
		os.Exit(2)
	}
}

func trunc(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
	os.Exit(1)
}