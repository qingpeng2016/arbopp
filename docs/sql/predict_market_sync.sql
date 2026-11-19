-- Predict.fun 接入 market_sync（平台 + 配置 + 并发）
-- 前置：REST 需 x-api-key（任选其一）
--   1) platform.extra_config = '{"api_key":"你的key"}'
--   2) account 表 platform=PREDICT, status=ACTIVE, api_key=你的key
-- tag_ids：登录后 GET https://api.predict.fun/v1/tags，按联赛 name 填到 predict_sports.items[].tag_ids

-- 1) 平台（默认不 ACTIVE，填好 api_key 后再改为 ACTIVE）
INSERT INTO `platform` (
  `slug`, `platform_type`, `quote_style`, `display_name`, `sort`,
  `market_api_base_url`, `order_api_base_url`, `websocket_url`, `status`, `remark`, `extra_config`
) VALUES (
  'PREDICT', 'PREDICTION', 'PROBABILITY', 'Predict.fun', 25,
  'https://api.predict.fun/v1', 'https://api.predict.fun/v1', 'wss://ws.predict.fun/ws',
  'INACTIVE', 'market_sync 需 api_key；extra_config 或 account 表',
  NULL
)
ON DUPLICATE KEY UPDATE
  `platform_type` = VALUES(`platform_type`),
  `quote_style` = VALUES(`quote_style`),
  `display_name` = VALUES(`display_name`),
  `market_api_base_url` = VALUES(`market_api_base_url`),
  `order_api_base_url` = VALUES(`order_api_base_url`),
  `websocket_url` = VALUES(`websocket_url`),
  `remark` = VALUES(`remark`);

-- 2) 联赛 tag 配置（与 Poly/Kalshi 足球子类对齐；tag_ids 请按环境补全）
INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`) VALUES
(
  'market_sync',
  'predict_sports',
  '[{"sport_cn":"足球","sub_category":"Football","market_variants":["SPORTS_MATCH","SPORTS_TEAM_MATCH"],"items":[{"name":"欧国联","name_en":"UEFA Nations League","sport_id":"unl","tag_ids":[]},{"name":"英超","name_en":"Premier League","sport_id":"epl","tag_ids":[]},{"name":"欧冠","name_en":"UEFA Champions League","sport_id":"ucl","tag_ids":[]},{"name":"西甲","name_en":"LaLiga","sport_id":"lal","tag_ids":[]},{"name":"德甲","name_en":"Bundesliga","sport_id":"bun","tag_ids":[]},{"name":"意甲","name_en":"Serie A","sport_id":"sea","tag_ids":[]},{"name":"法甲","name_en":"Ligue 1","sport_id":"fl1","tag_ids":[]},{"name":"欧联","name_en":"UEFA Europa League","sport_id":"uel","tag_ids":[]},{"name":"美职联","name_en":"MLS","sport_id":"mls","tag_ids":[]}]}]',
  'json',
  'Predict 足球：按 tag_ids 过滤 GET /v1/categories；market_variants 默认 SPORTS_MATCH',
  5,
  1
)
ON DUPLICATE KEY UPDATE
  `item_value` = VALUES(`item_value`),
  `value_type` = VALUES(`value_type`),
  `description` = VALUES(`description`),
  `is_enabled` = VALUES(`is_enabled`);

-- 3) 联赛拉取并发（在原有 poly/kalshi 上增加 predict）
UPDATE `config_strategy`
SET `item_value` = JSON_SET(
  COALESCE(NULLIF(`item_value`, ''), '{"polymarket":{"concurrency":3},"kalshi":{"concurrency":3}}'),
  '$.predict',
  JSON_OBJECT('concurrency', 2)
),
    `description` = '市场同步：polymarket/kalshi/predict 的 concurrency（1-8）'
WHERE `group_name` = 'market_sync' AND `item_key` IN ('market_league_fetch', 'league_concurrency', 'concurrency');

INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`)
SELECT 'market_sync', 'concurrency',
  '{"polymarket":{"concurrency":3},"kalshi":{"concurrency":3},"predict":{"concurrency":2}}',
  'json', '市场同步：polymarket/kalshi/predict 的 concurrency（1-8）', 4, 1
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM `config_strategy` WHERE `group_name` = 'market_sync' AND `item_key` IN ('market_league_fetch', 'league_concurrency', 'concurrency')
);

-- 4) 手续费占位（与 Poly 足球一致，可后续按 Predict feeRateBps 调整）
INSERT INTO `config_strategy_fee` (`platform`, `category`, `sub_category`, `taker_fee_coef`, `description`, `status`) VALUES
  ('PREDICT', 'Sports', '', 0.02000000, 'Predict 体育默认占位（按市场 feeRateBps 微调）', 1),
  ('PREDICT', 'Sports', 'Football', 0.02000000, '足球', 1)
ON DUPLICATE KEY UPDATE
  `taker_fee_coef` = VALUES(`taker_fee_coef`),
  `description` = VALUES(`description`),
  `status` = VALUES(`status`);

-- 5) 启用平台（确认 api_key 已配置后执行）
-- UPDATE `platform` SET `status` = 'ACTIVE' WHERE `slug` = 'PREDICT';