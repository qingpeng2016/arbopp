-- 历史订单簿任务：按 map_key + ingest_source 区分 PMData / 伪造

CREATE TABLE IF NOT EXISTS `data_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ingest_source` varchar(16) NOT NULL DEFAULT 'pmdata' COMMENT 'pmdata=history_data_sync；synthetic=synthetic_history_sync',
  `map_key` varchar(192) NOT NULL DEFAULT '' COMMENT 'synthetic 必填 v1|联赛|slug；pmdata 留空，走 params.fixture_slug',
  `params` text NULL COMMENT 'JSON 任务覆盖，见 migration_data_task_ingest_source.sql 说明',
  `status` varchar(24) NOT NULL DEFAULT 'pending' COMMENT 'pending|running|completed|failed',
  `error_text` varchar(512) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_data_task_ingest_status` (`ingest_source`, `status`),
  KEY `idx_data_task_map_key` (`map_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- PMData：map_key 留空，参数驱动
-- INSERT INTO `data_task` (`ingest_source`, `map_key`, `params`, `status`) VALUES
-- ('pmdata', '',
--  '{"fixture_slug":"epl-eve-che-2026-10-17","min_utc_age_days":21,"utc_dates":["2026-10-16","2026-10-17","2026-10-18"],"outcome_types":["moneyline"],"market_url":{"POLYMARKET":"https://polymarket.com/event/epl-eve-che-2026-10-17","PREDICT":"https://predict.fun/market/epl-eve-che-2026-10-17"}}',
--  'pending');

-- synthetic：必须 map_key
-- INSERT INTO `data_task` (`ingest_source`, `map_key`, `status`)
-- VALUES ('synthetic', 'v1|俱乐部友谊赛|clf-tur-jab-2026-11-17', 'pending');