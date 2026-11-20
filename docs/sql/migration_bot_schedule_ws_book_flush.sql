-- WS 盘口：Redis 队列 -> market_order_book_ws 批量落库（ws_book_flush）
INSERT INTO `bot_schedule_config` (
  `module`, `task_name`, `interval_seconds`, `exe_sort`, `concurrency`,
  `description`, `is_enabled`, `is_strategy_enabled`, `is_primary_machine_run`
) VALUES (
  'less_one', 'ws_book_flush', 5, 31, 1,
  'Redis ob:ws:mysql_queue 批量写 market_order_book_ws（最多500/批）', 1, 1, 0
)
ON DUPLICATE KEY UPDATE
  `interval_seconds` = VALUES(`interval_seconds`),
  `exe_sort` = VALUES(`exe_sort`),
  `description` = VALUES(`description`),
  `is_enabled` = VALUES(`is_enabled`);