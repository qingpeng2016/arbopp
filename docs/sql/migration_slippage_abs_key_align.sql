-- 仅保留 predict_polymarket_sxbet_slippage_bps：绝对价格上浮（0.03 → 0.50 可到 0.53）。
UPDATE `config_strategy_threshold`
SET
  `config_value` = 0.03000000,
  `description` = 'Predict+Poly+SX 绝对价格滑点（0.03：卖一 0.50→最差 0.53；键名 _bps 为历史命名）',
  `status` = 1,
  `updated_at` = CURRENT_TIMESTAMP
WHERE `config_key` = 'predict_polymarket_sxbet_slippage_bps';

DELETE FROM `config_strategy_threshold`
WHERE `config_key` = 'predict_polymarket_sxbet_slippage';
