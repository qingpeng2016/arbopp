-- 同一联赛的中文名与 Polymarket 对齐。只改 Kalshi 配置里的 name。
UPDATE `config_strategy`
SET
  `item_value` = REPLACE(REPLACE(REPLACE(REPLACE(
      `item_value`,
      '"name":"国际友谊赛"', '"name":"国际足球"'),
      '"name":"巴拉圭荣誉甲"', '"name":"巴拉圭甲"'),
      '"name":"秘鲁甲冠军"', '"name":"秘鲁甲"'),
      '"name":"乌拉圭甲冠军"', '"name":"乌拉圭甲"'),
  `updated_at` = NOW()
WHERE `group_name` = 'market_sync'
  AND `item_key` = 'kalshi_game_series';
