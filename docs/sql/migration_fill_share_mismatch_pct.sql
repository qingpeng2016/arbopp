-- 下单结果通知：多平台成交份额 (max-min)/max 容忍比例（0.01=1%）
INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`, `status`)
VALUES
  ('fill_share_mismatch_pct', 0.01000000, 'DECIMAL', '下单结果判定套利成功：各平台成交份额 (max-min)/max 上限（0.01=1%）', 1)
ON DUPLICATE KEY UPDATE
  `config_value` = VALUES(`config_value`),
  `description` = VALUES(`description`),
  `status` = 1;
