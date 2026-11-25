-- 实盘成交回补后写入：min(各平台成交量合计) - 总花费(price*fill_qty)
ALTER TABLE `arb_binary_opportunity`
  ADD COLUMN `arb_profit_amount` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '实盘套利利润：min(平台成交量)-总花费' AFTER `arb_profit`;

ALTER TABLE `backtest_arb_binary_opportunity`
  ADD COLUMN `arb_profit_amount` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '实盘套利利润：min(平台成交量)-总花费' AFTER `arb_profit`;
