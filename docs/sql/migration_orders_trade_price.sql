-- orders：实际成交均价（order_fill_sync 回填）
-- 注：后续 migration_orders_trade_qty_amount.sql 将 fill_qty 重命名为 trade_qty
ALTER TABLE `orders`
  ADD COLUMN `trade_price` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '实际成交均价（回补；无成交为 0）' AFTER `fill_qty`;
