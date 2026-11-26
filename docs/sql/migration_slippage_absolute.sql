-- 滑点改为绝对价格上浮（例 0.05：限价 0.20→0.25、0.90→0.95）
-- Predict：按价格反算 slippageBps；Poly/SX：限价 = 卖一 + abs
INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`, `status`)
VALUES
  ('predict_polymarket_sxbet_slippage', 0.05000000, 'DECIMAL', 'Predict+Polymarket+SX 共用绝对价格滑点（0.05=限价+0.05；Predict 按价反算 slippageBps）', 1)
ON DUPLICATE KEY UPDATE
  `config_value` = VALUES(`config_value`),
  `description` = VALUES(`description`),
  `status` = 1;

-- 旧相对 bps 键停用（勿把 200/500 当成绝对价）
UPDATE `config_strategy_threshold`
SET `status` = 0
WHERE `config_key` IN (
  'predict_polymarket_sxbet_slippage_bps',
  'predict_slippage_bps',
  'predict_polymarket_slippage_bps'
);
