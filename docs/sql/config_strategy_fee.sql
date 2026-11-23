-- 试算吃单系数 taker_fee_coef（非「成交额 × 百分比」）。
-- sub_category 空 = 该平台 category 下默认；仅当某运动系数不同时另插 sub_category 行。
--
-- POLYMARKET / KALSHI：每份 fee = coef × 价格 × (1−价格)；净成本 = 价格 + fee。
-- PREDICT：优先 outcome.config.feeRateBps；否则用本表 coef（按份扣费模型）。
-- SXBET：净成本 = 价格 / (1 − coef×(1−价格))（赢家净利口径）。
--
-- Polymarket Sports coef 0.05：https://docs.polymarket.com/trading/fees
-- Kalshi 系列乘数 1 时 coef 0.07：https://kalshi.com/docs/kalshi-fee-schedule.pdf

CREATE TABLE IF NOT EXISTS `config_strategy_fee` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `platform` varchar(32) NOT NULL DEFAULT '' COMMENT 'POLYMARKET / KALSHI / PREDICT / SXBET',
  `category` varchar(64) NOT NULL DEFAULT '' COMMENT '市场大类，运动为 Sports',
  `sub_category` varchar(64) NOT NULL DEFAULT '' COMMENT '运动。空字符串表示该大类的默认系数',
  `taker_fee_coef` decimal(20,8) NOT NULL COMMENT '吃单系数；Poly/Kalshi 为 p(1-p) 公式系数，Predict/SX 见文件头说明',
  `description` varchar(500) NOT NULL DEFAULT '',
  `status` tinyint NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_fee_platform_category_sub` (`platform`, `category`, `sub_category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='按平台和运动的吃单手续费系数';

INSERT INTO `config_strategy_fee` (`platform`, `category`, `sub_category`, `taker_fee_coef`, `description`, `status`) VALUES
  ('POLYMARKET', 'Sports', '', 0.05000000, 'Poly 体育吃单：每份 fee=coef×p×(1−p)，官方 Sports coef=0.05', 1),
  ('KALSHI', 'Sports', '', 0.07000000, 'Kalshi 吃单 coef=0.07（系列乘数 1）；改乘数时加 sub_category 行覆盖', 1),
  ('PREDICT', 'Sports', '', 0.02000000, '无 market feeRateBps 时回退；有 bps 时策略用 bps/10000', 1),
  ('SXBET', 'Sports', '', 0.01000000, 'SX 赢家净利试算 coef；与代码 defaultSxbetSports=0.01 一致，Premium 可改为 0.02', 1)
ON DUPLICATE KEY UPDATE
  `taker_fee_coef` = VALUES(`taker_fee_coef`),
  `description` = VALUES(`description`),
  `status` = VALUES(`status`);

UPDATE `config_strategy_threshold`
SET `status` = 0,
    `description` = '已停用。手续费改看 config_strategy_fee'
WHERE `config_key` IN ('binary_yes_no_poly_fee_coef', 'binary_yes_no_kalshi_fee_coef');
