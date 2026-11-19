CREATE TABLE IF NOT EXISTS `market_category` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `sport_cn` varchar(64) NOT NULL,
  `sub_category` varchar(64) NOT NULL,
  `platform` varchar(32) NOT NULL,
  `concurrency` int NOT NULL DEFAULT 1,
  `config_json` longtext,
  `outcome_types_json` varchar(512) NOT NULL DEFAULT '[]',
  `sort_no` int NOT NULL DEFAULT 0,
  `is_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_category_sub_platform` (`sub_category`, `platform`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;