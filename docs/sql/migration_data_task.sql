-- 历史订单簿任务：按 map_key + ingest_source 区分 PMData / 伪造

CREATE TABLE IF NOT EXISTS `data_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ingest_source` varchar(16) NOT NULL DEFAULT 'pmdata' COMMENT 'pmdata=history_data_sync；synthetic=synthetic_history_sync',
  `map_key` varchar(192) NOT NULL DEFAULT '' COMMENT 'pmdata 可填场次 slug；synthetic/回测仍可用 v1|联赛|slug',
  `params` text NULL COMMENT 'JSON 任务覆盖，见 migration_data_task_ingest_source.sql 说明',
  `status` varchar(24) NOT NULL DEFAULT 'pending' COMMENT 'pending|running|completed|failed',
  `error_text` varchar(512) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_data_task_ingest_status` (`ingest_source`, `status`),
  KEY `idx_data_task_map_key` (`map_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 免费 PMData：只拉已超过 3 周的 UTC 日（可按场次改 map_key / params）
INSERT INTO `data_task` (`ingest_source`, `map_key`, `params`, `status`)
SELECT 'pmdata', 'v1|国际足球|fif-chn-bah1-2026-11-17',
       '{"min_utc_age_days":21,"utc_days_before":1,"utc_days_after":1}',
       'pending'
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM `data_task`
  WHERE `ingest_source` = 'pmdata' AND `map_key` = 'v1|国际足球|fif-chn-bah1-2026-11-17'
);

-- 伪造秒级回测示例
-- INSERT INTO `data_task` (`ingest_source`, `map_key`, `status`)
-- VALUES ('synthetic', 'v1|俱乐部友谊赛|clf-tur-jab-2026-11-17', 'pending');