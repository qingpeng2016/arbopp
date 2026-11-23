-- 加列并部署新写入代码后执行。按同编号的盘口快照回填，兼容不同版本的二元腿位。
-- 只补缺失的平台，保留原记录的更新时间，可重复执行。
UPDATE backtest_arb_binary_opportunity
SET leg_1_platform = UPPER(TRIM(JSON_UNQUOTE(JSON_EXTRACT(leg_1_order_books, '$.platform')))),
    updated_at = updated_at
WHERE leg_1_platform IS NULL
  AND CASE WHEN JSON_VALID(leg_1_order_books) THEN
        JSON_TYPE(JSON_EXTRACT(leg_1_order_books, '$.platform')) = 'STRING'
        AND CHAR_LENGTH(UPPER(TRIM(JSON_UNQUOTE(JSON_EXTRACT(leg_1_order_books, '$.platform'))))) BETWEEN 1 AND 32
      ELSE FALSE END;

UPDATE backtest_arb_binary_opportunity
SET leg_2_platform = UPPER(TRIM(JSON_UNQUOTE(JSON_EXTRACT(leg_2_order_books, '$.platform')))),
    updated_at = updated_at
WHERE leg_2_platform IS NULL
  AND CASE WHEN JSON_VALID(leg_2_order_books) THEN
        JSON_TYPE(JSON_EXTRACT(leg_2_order_books, '$.platform')) = 'STRING'
        AND CHAR_LENGTH(UPPER(TRIM(JSON_UNQUOTE(JSON_EXTRACT(leg_2_order_books, '$.platform'))))) BETWEEN 1 AND 32
      ELSE FALSE END;

UPDATE backtest_arb_binary_opportunity
SET leg_3_platform = UPPER(TRIM(JSON_UNQUOTE(JSON_EXTRACT(leg_3_order_books, '$.platform')))),
    updated_at = updated_at
WHERE leg_3_platform IS NULL
  AND CASE WHEN JSON_VALID(leg_3_order_books) THEN
        JSON_TYPE(JSON_EXTRACT(leg_3_order_books, '$.platform')) = 'STRING'
        AND CHAR_LENGTH(UPPER(TRIM(JSON_UNQUOTE(JSON_EXTRACT(leg_3_order_books, '$.platform'))))) BETWEEN 1 AND 32
      ELSE FALSE END;
