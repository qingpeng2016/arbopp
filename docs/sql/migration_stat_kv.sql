-- 运行指标键值表（横向扩展，避免频繁加列）
CREATE TABLE IF NOT EXISTS `stat_kv` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `group_name` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '指标分组',
  `item_key` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '指标键',
  `item_value` text COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '指标值',
  `value_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'string' COMMENT 'string|int|float|json|datetime',
  `description` varchar(500) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '说明',
  `sort_no` int NOT NULL DEFAULT '0',
  `is_enabled` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_stat_kv_group_item` (`group_name`,`item_key`),
  KEY `idx_stat_kv_group` (`group_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运行统计键值（map_key 审计、各平台场次等）';

-- 调度见 migration_bot_schedule_stat_collect.sql