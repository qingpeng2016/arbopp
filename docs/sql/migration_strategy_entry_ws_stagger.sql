INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`) VALUES
  ('strategy_entry', 'ws_connect_stagger_ms', '{"min_ms":0,"max_ms":2000}', 'json', '每场各平台首次 WebSocket 建连前随机等待区间（毫秒）；仅首轮握手，断线重连仍为 1s', 3, 1)
ON DUPLICATE KEY UPDATE
  `item_value` = VALUES(`item_value`),
  `value_type` = VALUES(`value_type`),
  `description` = VALUES(`description`),
  `is_enabled` = VALUES(`is_enabled`);