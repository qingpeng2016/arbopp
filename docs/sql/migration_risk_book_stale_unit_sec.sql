-- 将 risk_relay_book_stale 从毫秒键迁移为秒（你已在库里的 12000ms → 12s）
INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`, `status`)
VALUES
  ('risk_relay_book_stale_sec', 12.00000000, 'DECIMAL', '镜像 BOOK 超过该时长未收到则熔断（秒；建议≥2×推送间隔）', 1)
ON DUPLICATE KEY UPDATE
  `config_value` = VALUES(`config_value`),
  `description` = VALUES(`description`),
  `status` = 1;

DELETE FROM `config_strategy_threshold` WHERE `config_key` = 'risk_relay_book_stale_ms';