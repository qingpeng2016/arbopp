-- stat_kv 查询示例（见 migration_stat_kv.sql 建表）

-- map_key_audit：应对场次 + audit D 违规计数（不含 total_map_keys / ok_groups）
SELECT item_key, item_value, description, updated_at
FROM stat_kv
WHERE group_name = 'map_key_audit'
ORDER BY sort_no;

-- 每平台 1 条 JSON：各子类 market 场次 + __total__
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

-- strategy_entry WS→套利检测耗时（每 5 分钟心跳刷一次；yesno / onextwo 各一条）
-- SELECT item_key, updated_at, LEFT(item_value, 120) FROM stat_kv
-- WHERE group_name = 'strategy_entry_detect_latency'
--   AND item_key IN ('ws_to_arb_check_ms_yesno', 'ws_to_arb_check_ms_onextwo');