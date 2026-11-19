-- strategy_entry 跨所套利每次下单的两腿记录
CREATE TABLE IF NOT EXISTS `arb_binary_order` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `arb_opp_id` bigint unsigned NOT NULL COMMENT 'arb_binary_opportunity.id',
  `map_key` varchar(192) NOT NULL,
  `side_no` tinyint unsigned NOT NULL COMMENT '1=组合第一边 2=组合第二边',
  `platform` varchar(32) NOT NULL,
  `platform_market_outcome_id` varchar(255) NOT NULL DEFAULT '',
  `token_id` varchar(255) NOT NULL DEFAULT '' COMMENT 'Polymarket token',
  `side` varchar(8) NOT NULL COMMENT 'YES|NO',
  `action` varchar(16) NOT NULL DEFAULT 'BUY',
  `price` decimal(20,8) NOT NULL,
  `qty` decimal(20,8) NOT NULL,
  `fill_qty` decimal(20,8) NOT NULL DEFAULT 0,
  `status` varchar(32) NOT NULL COMMENT 'FILLED|NO_FILL|ERROR',
  `err_msg` varchar(1024) NOT NULL DEFAULT '',
  `sent_at` datetime NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_arb_order_opp` (`arb_opp_id`),
  KEY `idx_arb_order_map_time` (`map_key`,`sent_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='strategy_entry 跨所二元下单记录';