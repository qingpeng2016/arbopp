-- market_map 修 Poly↔Predict 误配后：去掉「同 map_key + 同平台」重复父场次的核对标记，再跑 market_map。

UPDATE `market` m
INNER JOIN (
  SELECT `map_key`, `platform`
  FROM `market`
  WHERE `map_verified` = 1
    AND `map_key` IS NOT NULL
    AND TRIM(`map_key`) <> ''
    AND `status` = 'ACTIVE'
  GROUP BY `map_key`, `platform`
  HAVING COUNT(*) > 1
) d ON d.`map_key` = m.`map_key` AND d.`platform` = m.`platform`
SET m.`map_verified` = 0,
    m.`map_key` = NULL,
    m.`updated_at` = NOW()
WHERE m.`status` = 'ACTIVE';