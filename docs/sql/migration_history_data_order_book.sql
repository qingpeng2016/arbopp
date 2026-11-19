-- 第三方历史订单簿（PMData 等）与 data_platform 配置

CREATE TABLE IF NOT EXISTS `data_platform` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL DEFAULT '' COMMENT '供应商标识，如 pmdata',
  `display_name` varchar(128) NOT NULL DEFAULT '',
  `vendor_type` varchar(32) NOT NULL DEFAULT 'third_party',
  `api_base_url` varchar(512) NOT NULL DEFAULT '' COMMENT '如 https://api.pmdata.dev/v1',
  `api_key` varchar(512) NOT NULL DEFAULT '' COMMENT '请求头 api_key',
  `extra_config` text COMMENT 'JSON：storage_dir、utc_days_*、max_snapshots_per_day 等',
  `status` varchar(20) NOT NULL DEFAULT 'ACTIVE',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_data_platform_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `data_history_order_book` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `data_platform_id` bigint unsigned NOT NULL DEFAULT 0,
  `map_key` varchar(192) NOT NULL DEFAULT '',
  `market_row_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT 'market.id',
  `market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT 'market_outcome.id',
  `venue_platform` varchar(32) NOT NULL DEFAULT '' COMMENT 'POLYMARKET / PREDICT',
  `vendor_exchange` varchar(32) NOT NULL DEFAULT '' COMMENT 'pmdata 路径段 polymarket / predict_fun',
  `vendor_symbol` varchar(255) NOT NULL DEFAULT '' COMMENT 'poly market_slug 或 predict market_id',
  `utc_date` date NOT NULL COMMENT 'Parquet 文件 UTC 日',
  `data_type` varchar(32) NOT NULL DEFAULT 'l2',
  `source` varchar(16) NOT NULL DEFAULT 'pmdata' COMMENT 'pmdata|synthetic',
  `record_kind` varchar(16) NOT NULL DEFAULT 'snapshot' COMMENT 'file=仅落文件元数据；snapshot=簿/秒级快照',
  `snapshot_at_ms` bigint NOT NULL DEFAULT 0 COMMENT '快照毫秒时间戳；0 表示 file 级记录',
  `order_book_bids` mediumtext NOT NULL COMMENT 'JSON [[price,size],...]',
  `order_book_asks` mediumtext NOT NULL,
  `local_file_path` varchar(512) NOT NULL DEFAULT '',
  `file_bytes` bigint unsigned NOT NULL DEFAULT 0,
  `file_sha256` char(64) NOT NULL DEFAULT '',
  `ingest_status` varchar(24) NOT NULL DEFAULT 'ok' COMMENT 'ok|skipped|failed',
  `error_text` varchar(512) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_history_ob` (
    `data_platform_id`,
    `market_outcome_id`,
    `utc_date`,
    `data_type`,
    `record_kind`,
    `snapshot_at_ms`
  ),
  KEY `idx_history_map_key` (`map_key`),
  KEY `idx_history_market_row` (`market_row_id`),
  KEY `idx_history_venue` (`venue_platform`, `utc_date`),
  KEY `idx_history_replay` (`map_key`, `snapshot_at_ms`, `venue_platform`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO `data_platform` (`code`, `display_name`, `vendor_type`, `api_base_url`, `api_key`, `extra_config`, `status`)
SELECT 'pmdata', 'PMData', 'third_party', 'https://api.pmdata.dev/v1', '',
       '{"storage_dir":"var/pmdata","data_type":"l2","utc_days_before_event":1,"utc_days_after_event":1,"outcome_types":["moneyline"],"polymarket_outcome":"yes","max_snapshots_per_day":500}',
       'ACTIVE'
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `data_platform` WHERE `code` = 'pmdata');

INSERT INTO `bot_schedule_config` (`module`, `task_name`, `interval_seconds`, `exe_sort`, `concurrency`, `description`, `is_enabled`, `is_strategy_enabled`, `is_primary_machine_run`)
SELECT 'less_one', 'history_data_sync', 3600, 35, 1, 'data_task(pending) 场次：从 data_platform(PMData) 拉 L2 写入 data_history_order_book，完成后任务 status=completed', 0, 1, 0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM `bot_schedule_config` WHERE `module` = 'less_one' AND `task_name` = 'history_data_sync'
);