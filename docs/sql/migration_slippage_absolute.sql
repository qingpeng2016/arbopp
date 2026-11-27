-- 滑点为绝对价格上浮（例 0.03：限价 0.50→0.53）。线上键名 …_slippage_bps（历史命名，不是百分比）。
INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`, `status`)
VALUES
  ('predict_polymarket_sxbet_slippage_bps', 0.03000000, 'DECIMAL', 'Predict+Poly+SX 绝对价格滑点（0.03：卖一 0.50→最差 0.53；键名 _bps 为历史命名）', 1)
ON DUPLICATE KEY UPDATE
  `config_value` = VALUES(`config_value`),
  `description` = VALUES(`description`),
  `status` = 1;

DELETE FROM `config_strategy_threshold`
WHERE `config_key` IN (
  'predict_polymarket_sxbet_slippage',
  'predict_slippage_bps',
  'predict_polymarket_slippage_bps'
);
