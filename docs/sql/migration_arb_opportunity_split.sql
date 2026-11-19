-- 将 arb_opportunity 拆为 arb_binary_opportunity（可单独执行）
DROP TABLE IF EXISTS `arb_opportunity`;

CREATE TABLE IF NOT EXISTS `arb_binary_opportunity` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `opp_kind` varchar(32) NOT NULL COMMENT '机会类型，当前固定 BINARY_YES_NO（跨所买 YES+买 NO）',
  `map_key` varchar(192) DEFAULT NULL COMMENT '跨平台场次对齐键 market.map_key',
  `yes_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT 'YES 价腿 market_outcome.id',
  `no_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT 'NO 价腿 market_outcome.id',
  `outcomes_json` text NOT NULL COMMENT '组合两边摘要 JSON 数组：platform、market_id、market_outcome_id、side(YES|NO)、best_ask、size、fill',
  `order_books_json` mediumtext NOT NULL COMMENT '命中时盘口快照 JSON：{outcomes:[{side_no,platform,platform_market_outcome_id,books:[{book_key,role,bids,asks}]}]}',
  `yes_best_ask` decimal(20,8) DEFAULT NULL COMMENT '组合中 YES 腿卖一价（美元 0~1）',
  `no_best_ask` decimal(20,8) DEFAULT NULL COMMENT '组合中 NO 腿卖一价（美元 0~1）',
  `total_cost` decimal(20,8) NOT NULL COMMENT '两腿卖一之和 per_share（含费后逻辑价）',
  `profit_space` decimal(20,8) NOT NULL COMMENT '套利空间 1 - total_cost',
  `threshold_key` varchar(64) NOT NULL COMMENT '触发阈值配置键，如 binary_yes_no_max_cost',
  `threshold_value` decimal(20,8) NOT NULL COMMENT '触发时阈值配置值',
  `book_updated_at` datetime DEFAULT NULL COMMENT '写机会时关联盘口时间（占位，与 snapshot_at 同源）',
  `snapshot_at` datetime NOT NULL COMMENT '命中/快照时间',
  `snapshot_at_ms` bigint NOT NULL DEFAULT 0 COMMENT '命中瞬间毫秒时间戳',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_arb_binary_time` (`snapshot_at`),
  KEY `idx_arb_binary_map` (`map_key`,`snapshot_at`),
  KEY `idx_arb_binary_kind` (`opp_kind`,`snapshot_at`),
  KEY `idx_arb_binary_snap_ms` (`map_key`,`snapshot_at_ms`),
  KEY `idx_arb_binary_leg_outcomes` (`yes_market_outcome_id`,`no_market_outcome_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='跨平台二元 YES+NO 套利机会（strategy_entry）';
