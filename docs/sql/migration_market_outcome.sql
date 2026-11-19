-- 场次 / 盘口分层：market=Event，market_outcome=可交易 condition/ticker。
-- 部署后请清空或迁移旧数据并全量重跑 market_sync（旧行一条件一行无法无损拆分）。

CREATE TABLE IF NOT EXISTS `market_outcome` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `platform` varchar(32) NOT NULL DEFAULT '',
  `market_row_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '父场次 market.id',
  `platform_market_id` varchar(255) NOT NULL DEFAULT '' COMMENT '父场次平台 ID，对应 market.platform_market_id',
  `name` varchar(2000) DEFAULT NULL,
  `desc` varchar(2000) DEFAULT NULL,
  `platform_market_outcome_id` varchar(255) NOT NULL DEFAULT '' COMMENT '平台盘口 ID',
  `market_address` varchar(255) DEFAULT NULL,
  `market_url` varchar(255) NOT NULL DEFAULT '',
  `yes_token_id` varchar(255) NOT NULL DEFAULT '',
  `no_token_id` varchar(255) NOT NULL DEFAULT '',
  `selection_id` varchar(255) DEFAULT NULL,
  `market_start_time` varchar(255) DEFAULT NULL,
  `market_end_time` varchar(255) DEFAULT NULL,
  `status` varchar(20) NOT NULL DEFAULT 'ACTIVE',
  `category` varchar(200) DEFAULT NULL,
  `sub_category` varchar(200) DEFAULT NULL,
  `volume_total` double DEFAULT NULL,
  `volume24hr` double DEFAULT NULL,
  `slug` varchar(255) DEFAULT NULL,
  `competition` varchar(64) DEFAULT NULL,
  `config` text,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_outcome_sub` (`platform`,`platform_market_outcome_id`,`category`,`sub_category`),
  KEY `idx_outcome_market_row` (`market_row_id`),
  KEY `idx_outcome_platform_market` (`platform`,`platform_market_id`,`category`,`sub_category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='可交易盘口';