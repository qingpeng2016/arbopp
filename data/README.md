# 本地数据（你自己导出，脚本只读这里）

导出后放本目录，再跑：

```bash
go run ./data_analysis/ws_arb_scan \
  -data=./data \
  -out=./data_analysis/ws_arb_scan/out/report.jsonl
```

## 需要 3 个文件

| 文件 | 说明 |
|------|------|
| `map_keys.txt` | 一行一个 map_key（可与 `data_analysis/ws_arb_scan/map_keys.txt` 相同，复制过来） |
| `market_outcome.jsonl` | 每行一条 JSON，见下 |
| `ws_book.jsonl` | 每行一条 WS 盘口 JSON，见下 |

## 导出 SQL（在库里跑，结果存成 JSONL / 或用客户端导出）

**1. outcomes**（把 `IN (...)` 换成你的 map_key 列表，或先导入临时表）：

（可选字段 `outcome_group_key`：与线上一致时分组最准。）

```sql
SELECT JSON_OBJECT(
  'map_key', m.map_key,
  'platform', mo.platform,
  'platform_market_outcome_id', mo.platform_market_outcome_id,
  'outcome_name', IFNULL(mo.outcome_name,''),
  'outcome_type', IFNULL(mo.outcome_type,''),
  'yes_token_id', IFNULL(mo.yes_token_id,''),
  'no_token_id', IFNULL(mo.no_token_id,''),
  'slug', IFNULL(mo.slug,''),
  'id', mo.id
) AS line
FROM market_outcome mo
JOIN market m ON m.id = mo.market_row_id
WHERE m.map_key IN ( /* 你的 map_key */ );
```

**2. WS 盘口**（数据大时建议每 map_key 只导最近 5000 行，或只导 latest）：

```sql
SELECT JSON_OBJECT(
  'map_key', map_key,
  'platform', platform,
  'platform_market_outcome_id', platform_market_outcome_id,
  'token_id', IFNULL(token_id,''),
  'order_book_asks', order_book_asks,
  'order_book_bids', order_book_bids,
  'snapshot_at_ms', snapshot_at_ms,
  'id', id
) AS line
FROM market_order_book_ws
WHERE map_key IN ( /* 同上 */ )
ORDER BY id DESC;
```

MySQL 也可直接导出 CSV，列名与 JSON 字段一致即可（脚本同时支持 `.jsonl`）。

`export.sql` 里有一份带示例的完整查询。