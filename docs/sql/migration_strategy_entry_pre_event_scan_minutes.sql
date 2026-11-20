INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`) VALUES
  ('strategy_entry', 'pre_event_scan_minutes', '30', 'int', '仅扫描 event_start_time<=now+该分钟数(UTC)的场次；0=不按时开过滤', 4, 1)
ON DUPLICATE KEY UPDATE
  `description` = VALUES(`description`),
  `sort_no` = VALUES(`sort_no`);