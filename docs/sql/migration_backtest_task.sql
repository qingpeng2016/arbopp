-- 回测任务：按 map_key 重放 data_history_order_book，驱动 strategy 并写入 backtest_* 表

CREATE TABLE IF NOT EXISTS `backtest_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `map_key` varchar(192) NOT NULL DEFAULT '' COMMENT '场次 map_key，与 data_task 相同含义',
  `strategy_code` varchar(32) NOT NULL DEFAULT 'yesno' COMMENT '回测策略，当前仅 yesno',
  `status` varchar(24) NOT NULL DEFAULT 'pending' COMMENT 'pending|running|completed|failed',
  `progress` tinyint unsigned NOT NULL DEFAULT 0 COMMENT '回放进度 0-100',
  `error_text` varchar(512) NOT NULL DEFAULT '' COMMENT '失败摘要或进度文案',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_backtest_task_status` (`status`, `strategy_code`),
  KEY `idx_backtest_task_map` (`map_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='历史盘口回测任务';

CREATE TABLE IF NOT EXISTS `backtest_arb_binary_opportunity` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `backtest_task_id` bigint unsigned NOT NULL COMMENT 'backtest_task.id',
  `opp_kind` varchar(32) NOT NULL COMMENT 'BINARY_YES_NO | ONE_X2_YES_TRIPLE | ONE_X2_NO_TRIPLE',
  `map_key` varchar(192) DEFAULT NULL COMMENT '跨平台场次对齐键',
  `category` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sub_category` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `competition` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '联赛/杯赛名称（如英超）',
  `competition_en` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '统一英文名，来自配置 name_en',
  `yes_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT 'YES 价腿 market_outcome.id',
  `no_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT 'NO 价腿 market_outcome.id',
  `leg3_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '第三腿 outcome id，二元为 0',
  `outcomes_json` text NOT NULL COMMENT '组合两边摘要 JSON',
  `order_books_json` mediumtext NOT NULL COMMENT '命中时盘口快照 JSON',
  `yes_best_ask` decimal(20,8) DEFAULT NULL,
  `no_best_ask` decimal(20,8) DEFAULT NULL,
  `total_cost` decimal(20,8) NOT NULL,
  `profit_space` decimal(20,8) NOT NULL,
  `threshold_key` varchar(64) NOT NULL,
  `threshold_value` decimal(20,8) NOT NULL,
  `book_updated_at` datetime DEFAULT NULL,
  `snapshot_at` datetime NOT NULL COMMENT '回放时间（UTC）',
  `snapshot_at_ms` bigint NOT NULL DEFAULT 0 COMMENT '回放毫秒时间戳，对齐 data_history_order_book.snapshot_at_ms',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_bt_opp_task` (`backtest_task_id`, `snapshot_at`),
  KEY `idx_bt_opp_map` (`map_key`, `snapshot_at`),
  KEY `idx_bt_opp_snap_ms` (`backtest_task_id`, `snapshot_at_ms`),
  UNIQUE KEY `uk_bt_opp_snap_legs` (`backtest_task_id`, `snapshot_at_ms`, `opp_kind`, `yes_market_outcome_id`, `no_market_outcome_id`, `leg3_market_outcome_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='回测套利机会';

CREATE TABLE IF NOT EXISTS `backtest_arb_binary_order` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `backtest_task_id` bigint unsigned NOT NULL COMMENT 'backtest_task.id',
  `arb_opp_id` bigint unsigned NOT NULL COMMENT 'backtest_arb_binary_opportunity.id',
  `map_key` varchar(192) NOT NULL,
  `side_no` tinyint unsigned NOT NULL COMMENT '1=组合第一边 2=组合第二边',
  `platform` varchar(32) NOT NULL,
  `platform_market_outcome_id` varchar(255) NOT NULL DEFAULT '',
  `token_id` varchar(255) NOT NULL DEFAULT '',
  `side` varchar(8) NOT NULL COMMENT 'YES|NO',
  `action` varchar(16) NOT NULL DEFAULT 'BUY',
  `price` decimal(20,8) NOT NULL,
  `qty` decimal(20,8) NOT NULL,
  `fill_qty` decimal(20,8) NOT NULL DEFAULT 0,
  `status` varchar(32) NOT NULL COMMENT 'FILLED|NO_FILL|ERROR|SIMULATED',
  `err_msg` varchar(1024) NOT NULL DEFAULT '',
  `sent_at` datetime NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_bt_order_task` (`backtest_task_id`, `sent_at`),
  KEY `idx_bt_order_opp` (`arb_opp_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='回测模拟下单记录';

INSERT INTO `bot_schedule_config` (`module`, `task_name`, `interval_seconds`, `exe_sort`, `concurrency`, `description`, `is_enabled`, `is_strategy_enabled`, `is_primary_machine_run`)
SELECT 'less_one', 'backtest_replay', 3600, 37, 1, 'backtest_task(pending)：按秒重放 data_history_order_book 驱动 yesno+onextwo，机会写 backtest_arb_binary_opportunity、订单写 orders', 0, 1, 0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM `bot_schedule_config` WHERE `module` = 'less_one' AND `task_name` = 'backtest_replay'
);

-- 示例（需先 synthetic_l2 写入 data_history_order_book）:
-- INSERT INTO `backtest_task` (`map_key`, `strategy_code`, `status`)
-- VALUES ('v1|俱乐部友谊赛|clf-tur-jab-2026-11-17', 'yesno', 'pending');