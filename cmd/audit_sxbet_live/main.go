// 无 DB 时：抽多场 SX 活跃盘口，用与落库同一套 MessageFromSnapshot，并打印网页 URL。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	sxbetws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/sxbet"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/mysql"
)

func main() {
	_ = conf.NewCfg()
	catalog, _ := boot.NewPlatformCatalog(mysql.NewPlatformImpl(boot.NewDBClient()))
	api := boot.NewSxbetRepoFromPlatform(catalog, mysql.NewAccountImpl(boot.NewDBClient()))
	ctx := context.Background()

	// 英超 Coventry vs Palace 等当前在跑场次（event 从公开 markets 取几条 mainLine / 1X2）
	hashes := discoverHashes(ctx, api, 10)
	if len(hashes) == 0 {
		fmt.Println("未发现活跃 hash")
		os.Exit(1)
	}

	for i, h := range hashes {
		snap, err := api.GetOrderbookSnapshot(ctx, h.Hash)
		if err != nil {
			fmt.Printf("\n[%d] %s err %v\n", i+1, h.Title, err)
			continue
		}
		msg, ok := sxbetws.MessageFromSnapshot(1, snap)
		if !ok {
			fmt.Printf("\n[%d] %s empty\n", i+1, h.Title)
			continue
		}
		ask, bid := 0.0, 0.0
		if len(msg.Asks) > 0 {
			ask = msg.Asks[0].Price
		}
		if len(msg.Bids) > 0 {
			bid = msg.Bids[0].Price
		}
		// 落库 JSON 第一档（与 watch levelsJSON 一致）
		fmt.Printf("\n[%d] %s\n", i+1, h.Title)
		fmt.Printf("  url: %s\n", h.URL)
		fmt.Printf("  STORE best ask p=%.5f → 页面约 %.2f 赔率 | bid p=%.5f → %.2f\n",
			ask, inv(ask), bid, inv(bid))
		fmt.Printf("  levels ask=%d bid=%d (与 REST 整本替换，落库语义)\n", len(msg.Asks), len(msg.Bids))
	}
	fmt.Println("\n说明：打开 url 右侧 Orderbook，顶价 ask 的 1/p 应接近页面该侧十进制赔率（±几个 tick）。")
}

type pick struct {
	Title, Hash, URL string
}

func discoverHashes(ctx context.Context, api httprepo.SxbetRepo, want int) []pick {
	hc := &http.Client{Timeout: 25 * time.Second}
	const url = "https://api.sx.bet/markets/active?onlyMainLine=true&pageSize=50"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := hc.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var raw struct {
		Data struct {
			Markets []httpentity.SxbetMarket `json:"markets"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &raw) != nil {
		return nil
	}
	var out []pick
	seen := map[string]bool{}
	for _, m := range raw.Data.Markets {
		if strings.ToUpper(m.Status) != "ACTIVE" {
			continue
		}
		hash := strings.TrimSpace(m.MarketHash)
		if hash == "" || seen[hash] {
			continue
		}
		t := strings.TrimSpace(m.OutcomeOneName) + " vs " + strings.TrimSpace(m.OutcomeTwoName)
		if t == " vs " {
			t = "market"
		}
		out = append(out, pick{
			Title: t + " type=" + strconv.Itoa(m.Type),
			Hash:  hash,
			URL:   "https://sx.bet/sports/soccer/premiership/" + strings.TrimSpace(m.SportXeventID) + "/market/" + hash,
		})
		seen[hash] = true
		if len(out) >= want {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

func inv(p float64) float64 {
	if p <= 0 {
		return 0
	}
	return 1 / p
}