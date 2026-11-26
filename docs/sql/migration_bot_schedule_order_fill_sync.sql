-- 成交回补：按 orders.platform_order_id 查各平台，补 trade_qty / trade_price / trade_amount / status / fee
INSERT INTO `bot_schedule_config` (`module`, `task_name`, `interval_seconds`, `exe_sort`, `concurrency`, `description`, `is_enabled`, `is_strategy_enabled`, `is_primary_machine_run`)
SELECT 'less_one', 'order_fill_sync', 30, 45, 1,
  '回查 Poly/Predict/SX 成交，补全 orders.trade_qty/trade_price/trade_amount/status/fee', 1, 1, 0
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM `bot_schedule_config` WHERE `module` = 'less_one' AND `task_name` = 'order_fill_sync'
);
