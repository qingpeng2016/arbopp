-- SX Bet 平台与 market_sync 联赛配置（league_id 来自 GET /leagues/active）
-- 应用后请将 platform.slug=SXBET 的 status 改为 ACTIVE。

INSERT INTO `platform` (
  `slug`, `platform_type`, `quote_style`, `display_name`, `sort`,
  `market_api_base_url`, `order_api_base_url`, `websocket_url`, `status`, `remark`
) VALUES (
  'SXBET', 'PREDICTION', 'PROBABILITY', 'SX Bet', 28,
  'https://api.sx.bet', 'https://api.sx.bet', 'wss://realtime.sx.bet/connection/websocket',
  'INACTIVE', '公开市场数据无需 api_key；下单/盘口快照在 extra_config 配置 api_key 后启用'
)
ON DUPLICATE KEY UPDATE
  `market_api_base_url` = VALUES(`market_api_base_url`),
  `order_api_base_url` = VALUES(`order_api_base_url`),
  `websocket_url` = VALUES(`websocket_url`),
  `display_name` = VALUES(`display_name`);

INSERT INTO `market_category` (`sport_cn`, `sub_category`, `platform`, `concurrency`, `config_json`, `outcome_types_json`, `sort_no`, `is_enabled`)
VALUES
  ('篮球', 'Basketball', 'SXBET', 3, '{"sport_cn":"篮球","sub_category":"Basketball","items":[{"name":"NBA","name_en":"NBA","name_desc":"NBA","league_id":1,"pair_poly_sport_id":"nba"},{"name":"NCAA","name_en":"NCAA","name_desc":"NCAA","league_id":2}]}', '["moneyline","handicap","total"]', 4, 1),
  ('冰球', 'NHL', 'SXBET', 2, '{"sport_cn":"冰球","sub_category":"NHL","items":[{"name":"NHL","name_en":"NHL","name_desc":"NHL","league_id":3}]}', '["moneyline","handicap","total"]', 5, 1),
  ('橄榄球', 'NFL', 'SXBET', 2, '{"sport_cn":"橄榄球","sub_category":"NFL","items":[{"name":"NFL","name_en":"NFL","name_desc":"NFL","league_id":243}]}', '["moneyline","handicap","total"]', 6, 1),
  ('足球', 'Football', 'SXBET', 2, '{"sport_cn":"足球","sub_category":"Football","items":[{"name":"西甲","name_en":"La Liga","name_desc":"La Liga","league_id":1114,"pair_poly_sport_id":"lal"},{"name":"德甲","name_en":"Bundesliga","name_desc":"Bundesliga","league_id":244,"pair_poly_sport_id":"bun"},{"name":"美职联","name_en":"MLS","name_desc":"MLS","league_id":1115,"pair_poly_sport_id":"mls"}]}', '["moneyline","handicap","total"]', 1, 1),
  ('综合格斗', 'MMA', 'SXBET', 2, '{"sport_cn":"综合格斗","sub_category":"MMA","items":[{"name":"UFC","name_en":"UFC","name_desc":"UFC","league_id":34}]}', '["moneyline"]', 7, 1),
  ('网球', 'Tennis', 'SXBET', 2, '{"sport_cn":"网球","sub_category":"Tennis","items":[{"name":"ATP","name_en":"ATP Tour","name_desc":"ATP Tour","league_id":1842}]}', '["moneyline","handicap","total"]', 3, 1)
ON DUPLICATE KEY UPDATE
  `config_json` = VALUES(`config_json`),
  `concurrency` = VALUES(`concurrency`),
  `outcome_types_json` = VALUES(`outcome_types_json`),
  `is_enabled` = VALUES(`is_enabled`);

UPDATE `config_strategy`
SET `item_value` = JSON_SET(
  COALESCE(NULLIF(`item_value`, ''), '{"polymarket":{"concurrency":5},"kalshi":{"concurrency":3},"predict":{"concurrency":2}}'),
  '$.sxbet', JSON_OBJECT('concurrency', 3)
)
WHERE `group_name` = 'market_sync' AND `item_key` = 'concurrency';