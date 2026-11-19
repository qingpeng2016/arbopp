INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`) VALUES
  ('strategy_entry', 'max_groups', '50', 'int', 'strategy_entry 同时 WebSocket 监控的 map_key 组数上限；按成交额降序取前 N 组', 2, 1)
ON DUPLICATE KEY UPDATE
  `item_value` = VALUES(`item_value`),
  `value_type` = VALUES(`value_type`),
  `description` = VALUES(`description`),
  `is_enabled` = VALUES(`is_enabled`);