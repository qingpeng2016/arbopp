-- Predict MARKET + Polymarket LIMIT FAK 共用滑点 bps（100=1%）
-- Predict：slippageBps；Poly：LimitPrice = 卖一 × (1 + bps/10000)
INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`, `status`)
VALUES
  ('predict_polymarket_slippage_bps', 200.00000000, 'DECIMAL', 'Predict+Polymarket 共用滑点 bps（100=1%；Predict→slippageBps，Poly→LimitPrice×(1+bps/10000)）', 1),
  ('predict_is_min_amount_out', 1.00000000, 'DECIMAL', 'Predict MARKET isMinAmountOut（1=true 花固定 USDT、少收份额；0=false 多付钱拿满份额）', 1)
ON DUPLICATE KEY UPDATE
  `config_value` = VALUES(`config_value`),
  `description` = VALUES(`description`),
  `status` = 1;

-- 旧键迁移：有 predict_slippage_bps 则拷到新键后禁用旧键
UPDATE `config_strategy_threshold` AS n
INNER JOIN `config_strategy_threshold` AS o
  ON o.config_key = 'predict_slippage_bps' AND n.config_key = 'predict_polymarket_slippage_bps'
SET n.config_value = o.config_value, n.status = 1
WHERE o.status = 1;

UPDATE `config_strategy_threshold`
SET `status` = 0
WHERE `config_key` = 'predict_slippage_bps';
