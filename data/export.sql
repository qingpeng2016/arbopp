-- 按 map_keys.txt 批量导出前，可把键放进临时表：
-- CREATE TEMPORARY TABLE tmp_map_key (map_key varchar(192) PRIMARY KEY);
-- LOAD DATA LOCAL INFILE 'map_keys.txt' INTO TABLE tmp_map_key ...

-- outcomes → market_outcome.jsonl（客户端：每行取 line 列，或整行 JSON）
SELECT JSON_OBJECT(
  'map_key', m.map_key,
  'id', mo.id,
  'platform', mo.platform,
  'platform_market_outcome_id', mo.platform_market_outcome_id,
  'outcome_name', IFNULL(mo.outcome_name, ''),
  'outcome_type', IFNULL(mo.outcome_type, ''),
  'yes_token_id', IFNULL(mo.yes_token_id, ''),
  'no_token_id', IFNULL(mo.no_token_id, ''),
  'slug', IFNULL(mo.slug, '')
)
FROM market_outcome mo
JOIN market m ON m.id = mo.market_row_id
WHERE m.map_key = 'v1|NBA|nba-mem-min-2026-11-21';  -- 改成 IN (SELECT map_key FROM tmp_map_key)

-- ws_book → ws_book.jsonl
SELECT JSON_OBJECT(
  'map_key', map_key,
  'platform', platform,
  'platform_market_outcome_id', platform_market_outcome_id,
  'token_id', IFNULL(token_id, ''),
  'order_book_asks', order_book_asks,
  'order_book_bids', order_book_bids,
  'snapshot_at_ms', snapshot_at_ms,
  'id', id
)
FROM market_order_book_ws
WHERE map_key = 'v1|NBA|nba-mem-min-2026-11-21'
ORDER BY id DESC
LIMIT 8000;