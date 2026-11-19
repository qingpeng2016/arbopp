ALTER TABLE `market_category`
  ADD COLUMN `concurrency` int NOT NULL DEFAULT 1 AFTER `platform`;

UPDATE `market_category`
SET `concurrency` = GREATEST(1, LEAST(8, CAST(JSON_UNQUOTE(JSON_EXTRACT(`config_json`, '$.concurrency')) AS SIGNED)))
WHERE JSON_EXTRACT(`config_json`, '$.concurrency') IS NOT NULL;

UPDATE `market_category`
SET `concurrency` = CASE `platform`
  WHEN 'POLYMARKET' THEN 5
  WHEN 'KALSHI' THEN 3
  WHEN 'PREDICT' THEN 2
  ELSE `concurrency`
END
WHERE `sub_category` = 'Football' AND `concurrency` = 1;