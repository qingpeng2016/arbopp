-- orders：fill_qty → trade_qty；新增 trade_amount = trade_price * trade_qty
ALTER TABLE `orders`
  CHANGE COLUMN `fill_qty` `trade_qty` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '实际成交份额（回补；无成交为 0）';

ALTER TABLE `orders`
  ADD COLUMN `trade_amount` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '实际成交金额=trade_price*trade_qty' AFTER `trade_price`;

UPDATE `orders`
SET `trade_amount` = ROUND(`trade_price` * `trade_qty`, 8)
WHERE `trade_qty` > 0 AND `trade_price` > 0 AND `trade_amount` = 0;
