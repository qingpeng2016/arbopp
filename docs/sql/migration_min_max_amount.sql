-- 下单份额改用 min_amount / max_amount（替换 binary_yes_no_order_amount / binary_yes_no_level_factor）
INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`, `status`)
VALUES
  ('min_amount', 5.00000000, 'DECIMAL', '最小下单份额金额（÷各平台最低 best_ask → min_qty）', 1),
  ('max_amount', 5.00000000, 'DECIMAL', '最大下单份额金额（÷各平台最低 best_ask → max_qty）', 1)
ON DUPLICATE KEY UPDATE
  `config_value` = VALUES(`config_value`),
  `description` = VALUES(`description`),
  `status` = 1;

UPDATE `config_strategy_threshold`
SET `status` = 0
WHERE `config_key` IN ('binary_yes_no_order_amount', 'binary_yes_no_level_factor');
