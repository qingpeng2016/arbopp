-- account 初始总资金（手工维护，无业务逻辑）
ALTER TABLE `account`
  ADD COLUMN `origin_total_amount` decimal(20,8) DEFAULT NULL COMMENT '总的初始资金' AFTER `order_max_pending_time`;
