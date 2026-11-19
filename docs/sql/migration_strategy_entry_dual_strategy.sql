-- 二元 + 1X2 双策略：元数据与注释（结构见 migration_orders_*, migration_backtest_opp_leg3_unique）

INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`, `status`)
VALUES
  ('1x2_yes_triple_max_cost', 0.95000000, 'DECIMAL', '1X2 三路 YES 买齐最大总成本', 1),
  ('1x2_no_triple_max_cost', 1.90000000, 'DECIMAL', '1X2 三路 NO 买齐最大总成本', 1)
ON DUPLICATE KEY UPDATE
  `config_value` = VALUES(`config_value`),
  `description` = VALUES(`description`),
  `status` = VALUES(`status`);

ALTER TABLE `backtest_task`
  MODIFY COLUMN `strategy_code` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'strategy_entry'
    COMMENT 'strategy_entry|yesno=二元+1X2；onextwo=仅1X2；binary_yes_no=仅二元';

ALTER TABLE `orders`
  COMMENT='strategy_entry 下单（实盘/回测，二元与1X2）';

ALTER TABLE `orders`
  MODIFY COLUMN `side_no` tinyint unsigned NOT NULL COMMENT '组合腿序号 1..3（1X2 为三路）';

ALTER TABLE `orders`
  MODIFY COLUMN `arb_opp_id` bigint unsigned NOT NULL COMMENT '机会 id：实盘 arb_binary_opportunity；回测 backtest_arb_binary_opportunity';

ALTER TABLE `arb_binary_opportunity`
  MODIFY COLUMN `opp_kind` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL
    COMMENT 'BINARY_YES_NO | ONE_X2_YES_TRIPLE | ONE_X2_NO_TRIPLE';

ALTER TABLE `backtest_arb_binary_opportunity`
  MODIFY COLUMN `opp_kind` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL
    COMMENT 'BINARY_YES_NO | ONE_X2_YES_TRIPLE | ONE_X2_NO_TRIPLE';

ALTER TABLE `arb_binary_opportunity`
  COMMENT='strategy_entry 套利机会（二元 YES+NO 与 1X2）';

ALTER TABLE `backtest_arb_binary_opportunity`
  COMMENT='回测套利机会（二元与 1X2，订单见 backtest_orders）';

UPDATE `bot_schedule_config`
SET `description` = 'backtest_task(pending)：重放 data_history_order_book，驱动 yesno+onextwo；机会 backtest_arb_binary_opportunity，订单 backtest_orders'
WHERE `module` = 'less_one' AND `task_name` = 'backtest_replay';

UPDATE `bot_schedule_config`
SET `description` = '已核对场次 WebSocket：二元 YES+NO 与足球 1X2 三路全覆盖'
WHERE `module` = 'less_one' AND `task_name` = 'strategy_entry';