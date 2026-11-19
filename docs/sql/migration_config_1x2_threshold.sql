-- 1X2 全覆盖套利阈值（与 docs/all.sql 种子一致）
INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`, `status`)
VALUES
  ('1x2_yes_triple_max_cost', 0.95000000, 'DECIMAL', '1X2 三路 YES 买齐最大总成本', 1),
  ('1x2_no_triple_max_cost', 1.90000000, 'DECIMAL', '1X2 三路 NO 买齐最大总成本', 1)
ON DUPLICATE KEY UPDATE
  `config_value` = VALUES(`config_value`),
  `description` = VALUES(`description`),
  `status` = VALUES(`status`);