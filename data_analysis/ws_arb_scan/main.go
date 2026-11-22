// 套利离线/半离线分析（规则见 RULES.md，不 import 策略包）
//
// 单场连库（推荐）：
//	RUN_CONF=conf/local.conf.yaml go run ./data_analysis/ws_arb_scan -map_key='v1|NBA|nba-mem-min-2026-11-21'
//
// 多场均文件，逐场查库：
//	RUN_CONF=conf/local.conf.yaml go run ./data_analysis/ws_arb_scan -keys=./data_analysis/ws_arb_scan/map_keys.txt
//
// 读本地 JSONL（无 -map_key / -keys 且 data 目录有文件）：
//	go run ./data_analysis/ws_arb_scan -data=./data
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	mapKey := flag.String("map_key", "", "单场 map_key，从库查询并分析")
	keysFile := flag.String("keys", "", "多场：map_key 列表文件，逐场查库")
	dataDir := flag.String("data", "", "读本地 data/*.jsonl，不连库")
	wsLimit := flag.Int("ws_limit", 8000, "每场 WS 最多行数")
	out := flag.String("out", "", "输出 JSONL；默认按单场/批量自动命名")
	flag.Parse()

	outPath := *out
	if outPath == "" {
		if *mapKey != "" {
			outPath = "data_analysis/ws_arb_scan/out/report_one.jsonl"
		} else {
			outPath = "data_analysis/ws_arb_scan/out/report.jsonl"
		}
	}

	var err error
	switch {
	case *mapKey != "" || *keysFile != "":
		err = runFromDB(*mapKey, *keysFile, outPath, *wsLimit)
	case *dataDir != "":
		err = runScan(*dataDir, *keysFile, outPath)
	default:
		err = fmt.Errorf("请指定 -map_key=... 或 -keys=map_keys.txt 或 -data=./data")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}