-- 回测订单表：结构与 orders 一致，回测只写本表

CREATE TABLE IF NOT EXISTS `backtest_orders` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `backtest_task_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT 'backtest_task.id',
  `strategy_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'BINARY_YES_NO' COMMENT '策略类型',
  `arb_opp_id` bigint unsigned NOT NULL COMMENT 'backtest_arb_binary_opportunity.id',
  `map_key` varchar(192) COLLATE utf8mb4_unicode_ci NOT NULL,
  `side_no` tinyint unsigned NOT NULL COMMENT '组合腿序号 1..3',
  `platform` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `platform_market_outcome_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `token_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `side` varchar(8) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'YES|NO',
  `action` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'BUY',
  `price` decimal(20,8) NOT NULL,
  `qty` decimal(20,8) NOT NULL,
  `fill_qty` decimal(20,8) NOT NULL DEFAULT 0,
  `status` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'FILLED|NO_FILL|ERROR|SIMULATED',
  `err_msg` varchar(1024) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `sent_at` datetime NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_backtest_orders_task` (`backtest_task_id`, `sent_at`),
  KEY `idx_backtest_orders_opp` (`arb_opp_id`),
  KEY `idx_backtest_orders_map_time` (`map_key`, `sent_at`),
  KEY `idx_backtest_orders_strategy` (`strategy_type`, `sent_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='回测模拟下单（与 orders 同结构）';