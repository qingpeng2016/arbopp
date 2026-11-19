-- 秒级历史订单簿：来源列 + 回放索引

ALTER TABLE `data_history_order_book`
  ADD COLUMN `source` varchar(16) NOT NULL DEFAULT 'pmdata' COMMENT 'pmdata|synthetic' AFTER `data_type`;

ALTER TABLE `data_history_order_book`
  ADD KEY `idx_history_replay` (`map_key`, `snapshot_at_ms`, `venue_platform`);

INSERT INTO `data_platform` (`code`, `display_name`, `vendor_type`, `api_base_url`, `api_key`, `extra_config`, `status`)
SELECT 'synthetic', 'Synthetic L2', 'synthetic', '', '',
       '{"total_seconds":1000,"batch_seconds":100,"outcome_types":["moneyline"]}',
       'ACTIVE'
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `data_platform` WHERE `code` = 'synthetic');

INSERT INTO `bot_schedule_config` (`module`, `task_name`, `interval_seconds`, `exe_sort`, `concurrency`, `description`, `is_enabled`, `is_strategy_enabled`, `is_primary_machine_run`)
SELECT 'less_one', 'synthetic_history_sync', 3600, 36, 1, 'data_task(synthetic_l2)：按秒伪造对齐订单簿写入 data_history_order_book', 0, 1, 0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM `bot_schedule_config` WHERE `module` = 'less_one' AND `task_name` = 'synthetic_history_sync'
);