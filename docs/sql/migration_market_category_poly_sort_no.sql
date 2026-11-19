-- 校正 POLYMARKET 子类 sort_no（与 config_market_category_install.sql 一致）
UPDATE `market_category` SET `sort_no` = 1, `updated_at` = NOW() WHERE `platform` = 'POLYMARKET' AND `sub_category` = 'Football';
UPDATE `market_category` SET `sort_no` = 2, `updated_at` = NOW() WHERE `platform` = 'POLYMARKET' AND `sub_category` = 'Cricket';
UPDATE `market_category` SET `sort_no` = 3, `updated_at` = NOW() WHERE `platform` = 'POLYMARKET' AND `sub_category` = 'Tennis';
UPDATE `market_category` SET `sort_no` = 4, `updated_at` = NOW() WHERE `platform` = 'POLYMARKET' AND `sub_category` = 'Basketball';
UPDATE `market_category` SET `sort_no` = 5, `updated_at` = NOW() WHERE `platform` = 'POLYMARKET' AND `sub_category` = 'NFL';
UPDATE `market_category` SET `sort_no` = 6, `updated_at` = NOW() WHERE `platform` = 'POLYMARKET' AND `sub_category` = 'NHL';
UPDATE `market_category` SET `sort_no` = 7, `updated_at` = NOW() WHERE `platform` = 'POLYMARKET' AND `sub_category` = 'MMA';