INSERT INTO `bot_schedule_config` (`module`, `task_name`, `interval_seconds`, `exe_sort`, `concurrency`, `description`, `is_enabled`, `is_strategy_enabled`, `is_primary_machine_run`)
SELECT 'less_one', 'stat_collect', 300, 25, 1,
  '刷新 stat_kv：map_key 审计 + 各平台 JSON(market/outcome/map_key 按子类)', 1, 1, 0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM `bot_schedule_config` WHERE `module` = 'less_one' AND `task_name` = 'stat_collect'
);