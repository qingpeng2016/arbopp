-- strategy_entry 结束时间宽限（分钟）
INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`)
VALUES (
  'strategy_entry',
  'end_event_scan_minutes',
  '180',
  'int',
  'event_end_time 有值则须 end+N>now(UTC)；空不滤；0=不按结束滤',
  5,
  1
)
ON DUPLICATE KEY UPDATE
  `item_value` = VALUES(`item_value`),
  `description` = VALUES(`description`),
  `sort_no` = VALUES(`sort_no`),
  `is_enabled` = VALUES(`is_enabled`);

UPDATE `config_strategy`
SET `description` = '最早开赛<=now+N(UTC)；0=不按开赛滤'
WHERE `group_name` = 'strategy_entry' AND `item_key` = 'pre_event_scan_minutes';