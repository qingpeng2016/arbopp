-- 启用 market_sync（模块 less_one，周期 600s = 10min）
INSERT INTO `bot_schedule_config` (
  `module`, `task_name`, `interval_seconds`, `exe_sort`, `concurrency`,
  `description`, `is_enabled`, `is_strategy_enabled`, `is_primary_machine_run`
) VALUES (
  'less_one', 'market_sync', 600, 10, 1,
  '各平台拉运动市场列表，写入 market 表', 1, 1, 0
)
ON DUPLICATE KEY UPDATE
  `interval_seconds` = VALUES(`interval_seconds`),
  `exe_sort` = VALUES(`exe_sort`),
  `description` = VALUES(`description`),
  `is_enabled` = VALUES(`is_enabled`);
