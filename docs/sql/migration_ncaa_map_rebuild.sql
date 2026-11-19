-- NCAA 篮球误配 map_key（含 cbb/cwbb 混并）：清空核对，用新版 market_map 重关联。无需重跑 market_sync。

UPDATE `market`
SET `map_verified` = 0,
    `map_key` = NULL,
    `updated_at` = NOW()
WHERE `status` = 'ACTIVE'
  AND `competition` = 'NCAA篮球'
  AND `map_verified` = 1;