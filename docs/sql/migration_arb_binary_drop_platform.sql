-- 机会表一行=跨平台两边；platform/market_id 列废弃（信息在 outcomes_json / order_books_json）
ALTER TABLE `arb_binary_opportunity`
  DROP COLUMN `platform`,
  DROP COLUMN `market_id`;