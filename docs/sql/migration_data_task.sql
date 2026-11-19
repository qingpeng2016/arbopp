-- 历史数据/回测任务：按 map_key 拉取，完成后更新状态

CREATE TABLE IF NOT EXISTS `data_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `task_type` varchar(32) NOT NULL DEFAULT 'history_l2' COMMENT '如 history_l2、backtest_replay',
  `map_key` varchar(192) NOT NULL DEFAULT '' COMMENT 'market.map_key',
  `status` varchar(24) NOT NULL DEFAULT 'pending' COMMENT 'pending|running|completed|failed',
  `error_text` varchar(512) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_data_task_status` (`task_type`, `status`),
  KEY `idx_data_task_map_key` (`map_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 示例：国际足球场次（待 PMData 拉取）
INSERT INTO `data_task` (`task_type`, `map_key`, `status`)
SELECT 'history_l2', 'v1|国际足球|fif-chn-bah1-2026-11-17', 'pending'
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM `data_task`
  WHERE `task_type` = 'history_l2' AND `map_key` = 'v1|国际足球|fif-chn-bah1-2026-11-17'
);

-- 秒级伪造回测示例（执行 migration_data_history_order_book_second.sql 后可用）
-- INSERT INTO `data_task` (`task_type`, `map_key`, `status`)
-- VALUES ('synthetic_l2', 'v1|俱乐部友谊赛|clf-tur-jab-2026-11-17', 'pending');