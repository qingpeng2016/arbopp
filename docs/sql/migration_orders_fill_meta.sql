-- orders / backtest_orders：客户端单号、平台单号、手续费（供成交回补脚本使用）
ALTER TABLE `orders`
  ADD COLUMN `client_order_id` varchar(64) NOT NULL DEFAULT '' COMMENT '本地生成的客户端单号' AFTER `token_id`,
  ADD COLUMN `platform_order_id` varchar(128) NOT NULL DEFAULT '' COMMENT '平台返回订单 ID/hash' AFTER `client_order_id`,
  ADD COLUMN `fee` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '手续费（结算币，USDC/USDT）' AFTER `fill_qty`;

ALTER TABLE `orders`
  ADD KEY `idx_orders_client_oid` (`client_order_id`),
  ADD KEY `idx_orders_platform_oid` (`platform`, `platform_order_id`);

ALTER TABLE `backtest_orders`
  ADD COLUMN `client_order_id` varchar(64) NOT NULL DEFAULT '' COMMENT '本地生成的客户端单号' AFTER `token_id`,
  ADD COLUMN `platform_order_id` varchar(128) NOT NULL DEFAULT '' COMMENT '平台返回订单 ID/hash' AFTER `client_order_id`,
  ADD COLUMN `fee` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '手续费（结算币）' AFTER `fill_qty`;
