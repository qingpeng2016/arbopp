-- 为 arb_binary_opportunity 补全字段注释（已有表执行）
ALTER TABLE `arb_binary_opportunity` COMMENT='跨平台二元 YES+NO 套利机会（strategy_entry）';

ALTER TABLE `arb_binary_opportunity`
  MODIFY COLUMN `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  MODIFY COLUMN `opp_kind` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '机会类型，当前固定 BINARY_YES_NO（跨所买 YES+买 NO）',
  MODIFY COLUMN `map_key` varchar(192) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '跨平台场次对齐键 market.map_key',
  MODIFY COLUMN `outcomes_json` text COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '组合两边摘要 JSON 数组：platform、market_id、side(YES|NO)、best_ask、size、fill',
  MODIFY COLUMN `order_books_json` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '命中时盘口快照 JSON：{outcomes:[{side_no,platform,platform_market_outcome_id,books:[{book_key,role,bids,asks}]}]}',
  MODIFY COLUMN `yes_best_ask` decimal(20,8) DEFAULT NULL COMMENT '组合中 YES 腿卖一价（美元 0~1）',
  MODIFY COLUMN `no_best_ask` decimal(20,8) DEFAULT NULL COMMENT '组合中 NO 腿卖一价（美元 0~1）',
  MODIFY COLUMN `total_cost` decimal(20,8) NOT NULL COMMENT '两腿卖一之和 per_share（含费后逻辑价）',
  MODIFY COLUMN `profit_space` decimal(20,8) NOT NULL COMMENT '套利空间 1 - total_cost',
  MODIFY COLUMN `threshold_key` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '触发阈值配置键，如 binary_yes_no_max_cost',
  MODIFY COLUMN `threshold_value` decimal(20,8) NOT NULL COMMENT '触发时阈值配置值',
  MODIFY COLUMN `book_updated_at` datetime DEFAULT NULL COMMENT '写机会时关联盘口时间（占位，与 snapshot_at 同源）',
  MODIFY COLUMN `snapshot_at` datetime NOT NULL COMMENT '命中/快照时间',
  MODIFY COLUMN `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  MODIFY COLUMN `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间';