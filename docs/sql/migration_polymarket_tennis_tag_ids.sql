-- Polymarket 网球：ATP/WTA primary tag(101232/102123) + keyset moneyline 返回 0；
-- 单场盘在 Gamma Tennis tag 864（再靠 sport_id 前缀 atp-/wta- 过滤）。
-- 执行后：market_sync（POLYMARKET Tennis）→ market_map

UPDATE `market_category`
SET `config_json` = '{
  "sport_cn": "网球",
  "sub_category": "Tennis",
  "items": [
    {"name": "ATP", "name_en": "ATP Tour", "name_desc": "ATP Tour", "sport_id": "atp", "tag_ids": [864]},
    {"name": "WTA", "name_en": "WTA Tour", "name_desc": "WTA Tour", "sport_id": "wta", "tag_ids": [864]}
  ]
}',
    `updated_at` = NOW()
WHERE `platform` = 'POLYMARKET'
  AND `sub_category` = 'Tennis';