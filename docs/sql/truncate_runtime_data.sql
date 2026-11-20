-- =============================================================================
-- 清空运行时业务数据，保留配置与凭据（执行前请确认库名、已备份）
--
-- 保留：platform, account, config_strategy*, bot_schedule_config,
--       market_category（联赛同步 JSON）, data_platform（PMData 等数据源配置）
--
-- 清空：市场、订单簿、机会/订单、统计 KV、回测、历史、data_task
-- =============================================================================
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 历史 / 任务（大表优先）
TRUNCATE TABLE `data_history_market_event`;
TRUNCATE TABLE `data_history_order_book`;
TRUNCATE TABLE `data_task`;

-- 回测
TRUNCATE TABLE `backtest_orders`;
TRUNCATE TABLE `backtest_arb_binary_opportunity`;
TRUNCATE TABLE `backtest_task`;

-- 实盘机会与下单
TRUNCATE TABLE `orders`;
TRUNCATE TABLE `arb_binary_opportunity`;

-- 实时订单簿
TRUNCATE TABLE `market_order_book`;

-- 市场（先 outcome 再 market，无 FK 但语义清晰）
TRUNCATE TABLE `market_outcome`;
TRUNCATE TABLE `market`;

-- 运行时统计（配置类在 config_* / bot_schedule_config，不碰）
TRUNCATE TABLE `stat_kv`;

-- 若库仍是旧表名，取消下面注释（与 orders 二选一，勿重复清）
-- TRUNCATE TABLE `arb_binary_order`;
-- TRUNCATE TABLE `backtest_arb_binary_order`;

SET FOREIGN_KEY_CHECKS = 1;

-- 执行后建议：market_sync → market_map → orderbook_sync / history_data_sync