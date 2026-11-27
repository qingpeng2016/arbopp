-- stat_kv 查询示例（见 migration_stat_kv.sql 建表）

-- map_key_audit：groups 一行（value_type=json，item_value 为缩进 JSON）
--   + strategy_entry 心跳写的 market_running_count_strategy
SELECT item_key, value_type, item_value, description, updated_at
FROM stat_kv
WHERE group_name = 'map_key_audit'
ORDER BY sort_no;

-- MySQL 8：解析 groups.item_value（需 CAST 为 JSON）
-- SELECT
--   CAST(item_value AS JSON)->>'$.compliant.total' AS compliant_total,
--   CAST(item_value AS JSON)->>'$.strategy.watchable' AS watchable
-- FROM stat_kv
-- WHERE group_name = 'map_key_audit' AND item_key = 'groups';

-- 每平台 1 条 JSON（2 空格缩进）：各子类 market 场次 + __total__
SELECT item_key, item_value, updated_at
FROM stat_kv
WHERE group_name = 'market_count'
ORDER BY sort_no;

-- 每平台 1 条 JSON：各子类 outcome 盘口 + __total__
SELECT item_key, item_value, updated_at
FROM stat_kv
WHERE group_name = 'outcome_count'
ORDER BY sort_no;

-- 每平台 1 条 JSON：各子类 map_key 去重数 + __total__
SELECT item_key, item_value, updated_at
FROM stat_kv
WHERE group_name = 'map_key_count'
ORDER BY sort_no;

-- MySQL 8：解析 JSON 中 Football 场次
-- SELECT item_key, item_value->>'$.Football' AS football_markets
-- FROM stat_kv WHERE group_name = 'market_count';

-- 元信息
SELECT * FROM stat_kv WHERE group_name = 'stat_meta';

-- strategy_entry：WS→下单前 + compute_* 纯试算(ms，不含发单)；心跳刷
-- SELECT item_key, updated_at, LEFT(item_value, 200) FROM stat_kv
-- WHERE group_name = 'strategy_entry_detect_latency'
--   AND item_key IN ('ws_to_arb_check_ms_yesno', 'ws_to_arb_check_ms_onextwo');
-- JSON: min_ms/max_ms = WS→下单前；compute_min_ms/compute_max_ms = 纯试算
--
-- Fire 发单墙钟（试算记完 → Fire 返回）
-- SELECT item_key, updated_at, LEFT(item_value, 200) FROM stat_kv
-- WHERE group_name = 'strategy_entry_fire_latency'
--   AND item_key IN ('fire_order_ms_yesno', 'fire_order_ms_onextwo');
-- JSON: by_map_key / by_platform → {min_ms,max_ms,samples}