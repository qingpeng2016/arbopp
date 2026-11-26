-- Predict MARKET 滑点：slippageBps（100=1%）+ isMinAmountOut（1=true 固定 USDT 少收份额）
INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`, `status`)
VALUES
  ('predict_slippage_bps', 200.00000000, 'DECIMAL', 'Predict MARKET slippageBps（基点，100=1%，默认 200=2%）', 1),
  ('predict_is_min_amount_out', 1.00000000, 'DECIMAL', 'Predict MARKET isMinAmountOut（1=true 花固定 USDT、少收份额；0=false 多付钱拿满份额）', 1)
ON DUPLICATE KEY UPDATE
  `config_value` = VALUES(`config_value`),
  `description` = VALUES(`description`),
  `status` = 1;
