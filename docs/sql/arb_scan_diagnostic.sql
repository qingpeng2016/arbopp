-- 离线核对 arb_scan_binary 口径（与脚本 resolveYesAsk / resolveNoAsk 一致）
-- 用法：mysql ... fgmm_strategy_less_one < docs/sql/arb_scan_diagnostic.sql

SET @max_cost := (SELECT config_value FROM config_strategy_threshold WHERE config_key='binary_yes_no_max_cost' AND status=1 LIMIT 1);

-- 粗看：跨平台同 map_key 下 Poly YES卖一 + 对手 Kalshi NO互补（1-买一）是否曾低于阈值
-- 完整重算建议用应用日志 arb_scan_binary-ScanStats 或 go test

SELECT 'threshold' AS what, @max_cost AS v
UNION ALL
SELECT 'verified_fixtures', COUNT(*) FROM market WHERE status='ACTIVE' AND map_verified=1 AND map_key <> ''
UNION ALL
SELECT 'scannable_outcomes', COUNT(*) FROM market_outcome o
  INNER JOIN market m ON m.id = o.market_row_id
  WHERE o.status='ACTIVE' AND m.map_verified=1 AND m.map_key <> ''
UNION ALL
SELECT 'market_order_book_rows', COUNT(*) FROM market_order_book;

-- 临时放宽阈值（观察用，勿在生产长期开着）：
-- UPDATE config_strategy_threshold SET config_value=1.05000000 WHERE config_key='binary_yes_no_max_cost';
