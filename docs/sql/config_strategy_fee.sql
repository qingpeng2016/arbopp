-- 吃单手续费系数。费用 = taker_fee_coef * 价格 * (1 - 价格)，按每个份额。
-- sub_category 为空：该平台这个 category 下，没有单独行的运动用这一行。
--
-- Polymarket 官方 Sports 类别吃单系数 0.05，各运动相同。Maker 为 0。
--   https://docs.polymarket.com/trading/fees
-- Kalshi 吃单费用 = 系列乘数 * 0.07 * 份额 * 价格 * (1-价格)。
--   2026-07-07 价目表里已列出的运动系列，乘数多为 1，所以系数写成 0.07。
--   某个系列改了乘数，改对应运动这一行。
--   https://kalshi.com/docs/kalshi-fee-schedule.pdf

CREATE TABLE IF NOT EXISTS `config_strategy_fee` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `platform` varchar(32) NOT NULL DEFAULT '' COMMENT 'POLYMARKET / KALSHI',
  `category` varchar(64) NOT NULL DEFAULT '' COMMENT '市场大类，运动为 Sports',
  `sub_category` varchar(64) NOT NULL DEFAULT '' COMMENT '运动。空字符串表示该大类的默认系数',
  `taker_fee_coef` decimal(20,8) NOT NULL COMMENT '吃单系数。费用 = 系数 * 价格 * (1-价格)',
  `description` varchar(500) NOT NULL DEFAULT '',
  `status` tinyint NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_fee_platform_category_sub` (`platform`, `category`, `sub_category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='按平台和运动的吃单手续费系数';

INSERT INTO `config_strategy_fee` (`platform`, `category`, `sub_category`, `taker_fee_coef`, `description`, `status`) VALUES
  ('POLYMARKET', 'Sports', '', 0.05000000, 'Polymarket 体育默认。官方 Sports 吃单系数 0.05，各运动相同', 1),
  ('POLYMARKET', 'Sports', 'Football', 0.05000000, '足球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Basketball', 0.05000000, '篮球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'NFL', 0.05000000, '美式橄榄球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'MLB', 0.05000000, '棒球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'NHL', 0.05000000, '冰球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Tennis', 0.05000000, '网球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'MMA', 0.05000000, '综合格斗。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Boxing', 0.05000000, '拳击。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Golf', 0.05000000, '高尔夫。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Esports', 0.05000000, '电竞。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Rugby', 0.05000000, '橄榄球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Cricket', 0.05000000, '板球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'WinterOlympics', 0.05000000, '冬奥。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'TableTennis', 0.05000000, '乒乓球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Formula1', 0.05000000, 'F1。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Chess', 0.05000000, '国际象棋。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Pickleball', 0.05000000, '匹克球。Polymarket Sports 吃单系数 0.05', 1),
  ('KALSHI', 'Sports', '', 0.07000000, 'Kalshi 体育默认。系列乘数 1 时吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Football', 0.07000000, '足球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Basketball', 0.07000000, '篮球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'NFL', 0.07000000, '美式橄榄球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'MLB', 0.07000000, '棒球。Kalshi 系列乘数 1，吃单系数 0.07。某系列乘数变了就改这一行', 1),
  ('KALSHI', 'Sports', 'NHL', 0.07000000, '冰球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Tennis', 0.07000000, '网球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'MMA', 0.07000000, '综合格斗。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Boxing', 0.07000000, '拳击。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Golf', 0.07000000, '高尔夫。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Esports', 0.07000000, '电竞。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Rugby', 0.07000000, '橄榄球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Cricket', 0.07000000, '板球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'WinterOlympics', 0.07000000, '冬奥。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'TableTennis', 0.07000000, '乒乓球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Formula1', 0.07000000, 'F1。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Chess', 0.07000000, '国际象棋。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Pickleball', 0.07000000, '匹克球。Kalshi 系列乘数 1，吃单系数 0.07', 1)
ON DUPLICATE KEY UPDATE
  `taker_fee_coef` = VALUES(`taker_fee_coef`),
  `description` = VALUES(`description`),
  `status` = VALUES(`status`);

UPDATE `config_strategy_threshold`
SET `status` = 0,
    `description` = '已停用。手续费改看 config_strategy_fee'
WHERE `config_key` IN ('binary_yes_no_poly_fee_coef', 'binary_yes_no_kalshi_fee_coef');
