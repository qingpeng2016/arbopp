// 离线扫 WS 盘口：只读 ./data 下 JSONL，不连数据库。
//
//	go run ./data_analysis/ws_arb_scan -data=./data
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	dataDir := flag.String("data", "data", "本地数据目录（map_keys.txt / market_outcome.jsonl / ws_book.jsonl）")
	keys := flag.String("keys", "", "map_key 列表文件，默认 data/map_keys.txt")
	out := flag.String("out", "data_analysis/ws_arb_scan/out/report.jsonl", "分析结果 JSONL")
	flag.Parse()
	if err := runScan(*dataDir, *keys, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}