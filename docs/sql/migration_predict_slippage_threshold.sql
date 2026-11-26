-- Predict / Polymarket / SX.bet 共用滑点 bps（100=1%）
-- Predict：slippageBps；Poly/SX：限价 = 卖一 × (1 + bps/10000)
INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`, `status`)
VALUES
  ('predict_polymarket_sxbet_slippage_bps', 200.00000000, 'DECIMAL', 'Predict+Polymarket+SX 共用滑点 bps（100=1%；Predict→slippageBps，Poly/SX→限价×(1+bps/10000)）', 1),
  ('predict_is_min_amount_out', 1.00000000, 'DECIMAL', 'Predict MARKET isMinAmountOut（1=true 花固定 USDT、少收份额；0=false 多付钱拿满份额）', 1)
ON DUPLICATE KEY UPDATE
  `config_value` = VALUES(`config_value`),
  `description` = VALUES(`description`),
  `status` = 1;

-- 旧键迁移：predict_slippage_bps / predict_polymarket_slippage_bps → 新键后禁用
UPDATE `config_strategy_threshold` AS n
INNER JOIN `config_strategy_threshold` AS o
  ON o.config_key IN ('predict_slippage_bps', 'predict_polymarket_slippage_bps')
 AND n.config_key = 'predict_polymarket_sxbet_slippage_bps'
SET n.config_value = o.config_value, n.status = 1
WHERE o.status = 1;

UPDATE `config_strategy_threshold`
SET `status` = 0
WHERE `config_key` IN ('predict_slippage_bps', 'predict_polymarket_slippage_bps');
