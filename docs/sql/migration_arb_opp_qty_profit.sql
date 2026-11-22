-- 套利机会：各腿卖一深度最小值 × 单位利润
ALTER TABLE `arb_binary_opportunity`
  ADD COLUMN `arb_qty` decimal(20,8) NOT NULL DEFAULT '0' COMMENT '各腿卖一挂单量最小值（可套利份数上限）' AFTER `profit_space`,
  ADD COLUMN `arb_profit` decimal(20,8) NOT NULL DEFAULT '0' COMMENT '套利金额 arb_qty * profit_space' AFTER `arb_qty`;

ALTER TABLE `backtest_arb_binary_opportunity`
  ADD COLUMN `arb_qty` decimal(20,8) NOT NULL DEFAULT '0' COMMENT '各腿卖一挂单量最小值（可套利份数上限）' AFTER `profit_space`,
  ADD COLUMN `arb_profit` decimal(20,8) NOT NULL DEFAULT '0' COMMENT '套利金额 arb_qty * profit_space' AFTER `arb_qty`;