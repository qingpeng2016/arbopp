-- 补全 Poly 篮球 config_json 联赛（EuroLeague / EuroCup / B.League 等）
UPDATE `market_category`
SET `config_json` = JSON_SET(
  `config_json`,
  '$.items', CAST('[
    {"name":"NBA","name_en":"NBA","name_desc":"NBA","sport_id":"nba","tag_ids":[745]},
    {"name":"NCAA男子篮球","name_en":"NCAA Men''s Basketball","name_desc":"NCAA Men''s Basketball","sport_id":"ncaab","tag_ids":[100149]},
    {"name":"NCAA篮球","name_en":"College Basketball","name_desc":"College Basketball","sport_id":"cbb","tag_ids":[101178]},
    {"name":"WNBA","name_en":"WNBA","name_desc":"WNBA","sport_id":"wnba","tag_ids":[100254]},
    {"name":"NCAA女子篮球","name_en":"Women''s College Basketball","name_desc":"Women''s College Basketball","sport_id":"cwbb","tag_ids":[102003]},
    {"name":"Liga ACB","name_en":"Liga ACB","name_desc":"Liga ACB","sport_id":"bkligend","tag_ids":[103094]},
    {"name":"NBL","name_en":"NBL","name_desc":"NBL","sport_id":"bknbl","tag_ids":[103096]},
    {"name":"Basketball Champions League","name_en":"Basketball Champions League","name_desc":"Basketball Champions League","sport_id":"bkcl","tag_ids":[103093]},
    {"name":"Lega Basket Serie A","name_en":"Lega Basket Serie A","name_desc":"Lega Basket Serie A","sport_id":"bkseriea","tag_ids":[103095]},
    {"name":"CBA","name_en":"CBA","name_desc":"CBA","sport_id":"bkcba","tag_ids":[103097]},
    {"name":"Betclic Élite","name_en":"Betclic Élite","name_desc":"Betclic Élite","sport_id":"bkfr1","tag_ids":[103098]},
    {"name":"Liga Nacional de Básquet","name_en":"Liga Nacional de Básquet","name_desc":"Liga Nacional de Básquet","sport_id":"bkarg","tag_ids":[103099]},
    {"name":"KBL","name_en":"KBL","name_desc":"KBL","sport_id":"bkkbl","tag_ids":[103100]},
    {"name":"EuroLeague","name_en":"EuroLeague","name_desc":"EuroLeague","sport_id":"euroleague","tag_ids":[102669]},
    {"name":"EuroCup","name_en":"EuroCup","name_desc":"EuroCup","sport_id":"bkeurocup","tag_ids":[105266]},
    {"name":"B.League","name_en":"B.League","name_desc":"B.League","sport_id":"bkjpn","tag_ids":[104349]},
    {"name":"NBB","name_en":"NBB","name_desc":"NBB","sport_id":"bknbb","tag_ids":[105271]},
    {"name":"VTB United League","name_en":"VTB United League","name_desc":"VTB United League","sport_id":"bkvtb","tag_ids":[104344]},
    {"name":"ABA League","name_en":"ABA League","name_desc":"ABA League","sport_id":"bkaba","tag_ids":[104346]},
    {"name":"Liga Argentina","name_en":"Liga Argentina","name_desc":"Liga Argentina","sport_id":"bkligarg","tag_ids":[105276]},
    {"name":"LKL","name_en":"LKL","name_desc":"LKL","sport_id":"bklkl","tag_ids":[105267]},
    {"name":"Basketball Bundesliga","name_en":"Basketball Bundesliga","name_desc":"Basketball Bundesliga","sport_id":"bkbbl","tag_ids":[104345]}
  ]' AS JSON)
),
`updated_at` = NOW()
WHERE `platform` = 'POLYMARKET' AND `sub_category` = 'Basketball';