-- 在 data_history_order_book 人为制造跨所套利（验证 yesno / backtest_replay）
-- 目标：Poly YES 卖一 0.38 + Predict NO 卖一 = 1 - YES买一 0.55 = 0.45 → per≈0.83+费 < max_cost 0.95
-- 场次：v1|俱乐部友谊赛|clf-tur-jab-2026-11-17
-- 赛果：Turnov win — POLY market_outcome_id=9172，PREDICT market_outcome_id=6384

UPDATE `data_history_order_book`
SET `order_book_bids` = '[[0.36,800.00],[0.35,320.00]]',
    `order_book_asks` = '[[0.38,720.00],[0.39,280.00]]'
WHERE `map_key` = 'v1|俱乐部友谊赛|clf-tur-jab-2026-11-17'
  AND `market_outcome_id` = 9172
  AND `error_text` REGEXP '^sec=(100|101|102|103|104|105)$';

UPDATE `data_history_order_book`
SET `order_book_bids` = '[[0.55,1400.00],[0.54,560.00]]',
    `order_book_asks` = '[[0.56,1260.00],[0.57,490.00]]'
WHERE `map_key` = 'v1|俱乐部友谊赛|clf-tur-jab-2026-11-17'
  AND `market_outcome_id` = 6384
  AND `error_text` REGEXP '^sec=(100|101|102|103|104|105)$';