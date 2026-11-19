-- Poly 同 map_key 下重复父场次：多为 slug 误收成 …-YYYY-MM（缺日）与完整 …-YYYY-MM-DD 并存。
-- 代码修复 polySlugFixtureBase 后跑 market_sync；本脚本关掉「截断 slug」的 ACTIVE 行（保留含完整日期 -DD 的）。

UPDATE `market` m
INNER JOIN (
  SELECT m1.id
  FROM `market` m1
  INNER JOIN `market` m2
    ON m1.map_key = m2.map_key
   AND m1.platform = m2.platform
   AND m1.platform = 'POLYMARKET'
   AND m1.id < m2.id
   AND m1.status = 'ACTIVE'
   AND m2.status = 'ACTIVE'
   AND m1.map_verified = 1
   AND m2.map_verified = 1
  WHERE m1.slug REGEXP '-[0-9]{4}-[0-9]{2}$'
    AND m2.slug REGEXP '-[0-9]{4}-[0-9]{2}-[0-9]{2}'
    AND LEFT(m2.slug, CHAR_LENGTH(m1.slug)) = m1.slug
) d ON d.id = m.id
SET m.status = 'closed',
    m.map_verified = 0,
    m.updated_at = NOW();