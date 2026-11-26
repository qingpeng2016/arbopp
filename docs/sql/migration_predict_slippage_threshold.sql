-- 历史：曾用相对 bps。现已改为绝对价格滑点，见 migration_slippage_absolute.sql
INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`, `status`)
VALUES
  ('predict_is_min_amount_out', 1.00000000, 'DECIMAL', 'Predict MARKET isMinAmountOut（1=true 花固定 USDT、少收份额；0=false 多付钱拿满份额）', 1)
ON DUPLICATE KEY UPDATE
  `config_value` = VALUES(`config_value`),
  `description` = VALUES(`description`),
  `status` = 1;
