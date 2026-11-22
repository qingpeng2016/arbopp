# 分析数据目录

连库跑 `ws_arb_scan`（`-map_key` / `-keys`）后自动在本目录生成：

| 文件 | 说明 |
|------|------|
| `<场次>_盘口outcome.csv` | 查到的全部 `market_outcome` |
| `<场次>_订单簿WS.csv` | 查到的全部 `market_order_book_ws` |
| `<场次>_套利分析报告.txt` | 中文字段、每行一项，含各平台 WS 条数 |

`*.csv` / 大 `*.txt` 已在 `.gitignore`，勿提交。

## 连库

```bash
RUN_CONF=conf/local.conf.yaml go run ./data_analysis/ws_arb_scan \
  -map_key='v1|NBA|nba-mem-min-2026-11-21'
```

## 离线（自-export JSONL 后）

本目录放置：

| 文件 | 说明 |
|------|------|
| `map_keys.txt` | 一行一个 map_key |
| `market_outcome.jsonl` | 每行一条 outcome JSON |
| `ws_book.jsonl` | 每行一条 WS 盘口 JSON |

```bash
go run ./data_analysis/ws_arb_scan \
  -data=./data_analysis/ws_arb_scan/data
```

导出 SQL 见仓库根目录 `data/export.sql`（路径仅作参考，产出请放本目录）。