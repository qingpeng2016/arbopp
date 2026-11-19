-- arb_binary_order → orders；增加 strategy_type、backtest_task_id；删除 backtest_arb_binary_order

RENAME TABLE `arb_binary_order` TO `orders`;

ALTER TABLE `orders`
  ADD COLUMN `backtest_task_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '回测任务 id，0=实盘' AFTER `id`,
  ADD COLUMN `strategy_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'BINARY_YES_NO' COMMENT '策略类型' AFTER `backtest_task_id`,
  ADD KEY `idx_orders_bt_task` (`backtest_task_id`, `sent_at`),
  ADD KEY `idx_orders_strategy` (`strategy_type`, `sent_at`);

UPDATE `orders` SET `strategy_type` = 'BINARY_YES_NO' WHERE `strategy_type` = '' OR `strategy_type` IS NULL;

DROP TABLE IF EXISTS `backtest_arb_binary_order`;