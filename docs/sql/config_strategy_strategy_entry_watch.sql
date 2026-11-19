-- 执行前把 item_value 里的 map_key 换成你在 market 表查到的值（可多个）
INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`) VALUES
  ('strategy_entry', 'watch_map_keys', '["REPLACE_WITH_YOUR_MAP_KEY"]', 'json', 'JSON map_key 数组；每项须对应 ACTIVE 且至少 Poly+Kalshi 各一条 market，否则忽略', 1, 1)
ON DUPLICATE KEY UPDATE
  `item_value` = VALUES(`item_value`),
  `description` = VALUES(`description`),
  `is_enabled` = VALUES(`is_enabled`);