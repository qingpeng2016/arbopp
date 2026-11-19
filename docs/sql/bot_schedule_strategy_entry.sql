-- 用 strategy_entry 替换原来的 arb_scan_binary 调度。
INSERT INTO `bot_schedule_config` (
  `module`, `task_name`, `interval_seconds`, `exe_sort`, `concurrency`,
  `description`, `is_enabled`, `is_strategy_enabled`, `is_primary_machine_run`
) VALUES (
  'less_one', 'strategy_entry', 20, 40, 1,
  '已核对场次 WebSocket 盘口，二元 YES+NO 全覆盖', 1, 1, 0
) ON DUPLICATE KEY UPDATE
  `interval_seconds` = VALUES(`interval_seconds`),
  `description` = VALUES(`description`),
  `is_enabled` = 1,
  `is_strategy_enabled` = 1;

UPDATE `bot_schedule_config`
SET `is_enabled` = 0
WHERE `module` = 'less_one' AND `task_name` = 'arb_scan_binary';
