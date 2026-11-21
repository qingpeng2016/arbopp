-- SXBET market_category 与 POLYMARKET/PREDICT 联赛名对齐（league_id 来自当时 GET /leagues/active）
-- 修正：sport_cn（NFL/MMA）、足球 31 联赛、网球全 active、NCAA橄榄球、电竞/板球
-- 可重复执行

UPDATE `market_category`
SET `sport_cn` = '足球',
    `config_json` = '{"sport_cn":"足球","sub_category":"Football","items":[{"name":"英超","name_en":"Premier League","name_desc":"Premier League","sport_id":"epl","pair_poly_sport_id":"epl","league_id":29},{"name":"欧冠","name_en":"UEFA Champions League","name_desc":"UEFA Champions League","sport_id":"ucl","pair_poly_sport_id":"ucl","league_id":30},{"name":"西甲","name_en":"LaLiga","name_desc":"LaLiga","sport_id":"lal","pair_poly_sport_id":"lal","league_id":1114},{"name":"德甲","name_en":"Bundesliga","name_desc":"Bundesliga","sport_id":"bun","pair_poly_sport_id":"bun","league_id":244},{"name":"意甲","name_en":"Serie A","name_desc":"Serie A","sport_id":"sea","pair_poly_sport_id":"sea","league_id":1113},{"name":"法甲","name_en":"Ligue 1","name_desc":"Ligue 1","sport_id":"fl1","pair_poly_sport_id":"fl1","league_id":1112},{"name":"欧联","name_en":"UEFA Europa League","name_desc":"UEFA Europa League","sport_id":"uel","pair_poly_sport_id":"uel","league_id":31},{"name":"荷甲","name_en":"Eredivisie","name_desc":"Eredivisie","sport_id":"ere","pair_poly_sport_id":"ere","league_id":1330},{"name":"葡超","name_en":"Primeira Liga","name_desc":"Primeira Liga","sport_id":"por","pair_poly_sport_id":"por","league_id":1236},{"name":"土超","name_en":"Süper Lig","name_desc":"Süper Lig","sport_id":"tur","pair_poly_sport_id":"tur","league_id":1331},{"name":"俄超","name_en":"Russian Premier League","name_desc":"Russian Premier League","sport_id":"rus","pair_poly_sport_id":"rus","league_id":1785},{"name":"美职联","name_en":"MLS","name_desc":"MLS","sport_id":"mls","pair_poly_sport_id":"mls","league_id":1115},{"name":"墨超","name_en":"Liga MX","name_desc":"Liga MX","sport_id":"mex","pair_poly_sport_id":"mex","league_id":1776},{"name":"巴甲","name_en":"Brasileirão Série A","name_desc":"Brasileirão Série A","sport_id":"bra","pair_poly_sport_id":"bra","league_id":1110},{"name":"阿甲","name_en":"Liga Profesional de Fútbol","name_desc":"Liga Profesional de Fútbol","sport_id":"arg","pair_poly_sport_id":"arg","league_id":1777},{"name":"沙特联","name_en":"Saudi Pro League","name_desc":"Saudi Pro League","sport_id":"spl","pair_poly_sport_id":"spl","league_id":1815},{"name":"K联赛","name_en":"K League 1","name_desc":"K League 1","sport_id":"kor","pair_poly_sport_id":"kor","league_id":1622},{"name":"J1","name_en":"J1 League","name_desc":"J1 League","sport_id":"jap","pair_poly_sport_id":"jap","league_id":1234},{"name":"澳超","name_en":"A-League Men","name_desc":"A-League Men","sport_id":"aus","pair_poly_sport_id":"aus","league_id":1833},{"name":"挪超","name_en":"Eliteserien","name_desc":"Eliteserien","sport_id":"nor","pair_poly_sport_id":"nor","league_id":1624},{"name":"丹超","name_en":"Danish Superliga","name_desc":"Danish Superliga","sport_id":"den","pair_poly_sport_id":"den","league_id":1783},{"name":"欧协联","name_en":"UEFA Conference League","name_desc":"UEFA Conference League","sport_id":"col","pair_poly_sport_id":"col","league_id":1309},{"name":"南美杯","name_en":"Copa Sudamericana","name_desc":"Copa Sudamericana","sport_id":"sud","pair_poly_sport_id":"sud","league_id":1781},{"name":"巴乙","name_en":"Brasileirão Série B","name_desc":"Brasileirão Série B","sport_id":"bra2","pair_poly_sport_id":"bra2","league_id":1778},{"name":"德乙","name_en":"2. Bundesliga","name_desc":"2. Bundesliga","sport_id":"bl2","pair_poly_sport_id":"bl2","league_id":1836},{"name":"英冠","name_en":"EFL Championship","name_desc":"EFL Championship","sport_id":"elc","pair_poly_sport_id":"elc","league_id":1313},{"name":"苏超","name_en":"Scottish Premiership","name_desc":"Scottish Premiership","sport_id":"scop","pair_poly_sport_id":"scop","league_id":1786},{"name":"意乙","name_en":"Serie B","name_desc":"Serie B","sport_id":"itsb","pair_poly_sport_id":"itsb","league_id":1439},{"name":"比甲","name_en":"Belgium Pro League","name_desc":"Belgium Pro League","sport_id":"bel1","pair_poly_sport_id":"bel1","league_id":1235},{"name":"哥伦甲","name_en":"Categoría Primera A","name_desc":"Categoría Primera A","sport_id":"col1","pair_poly_sport_id":"col1","league_id":1780},{"name":"英甲","name_en":"League One","name_desc":"League One","sport_id":"el1","pair_poly_sport_id":"el1","league_id":1837}]}',
    `updated_at` = NOW()
WHERE `platform` = 'SXBET' AND `sub_category` = 'Football';

UPDATE `market_category`
SET `sport_cn` = '篮球',
    `config_json` = '{"sport_cn":"篮球","sub_category":"Basketball","items":[{"name":"NBA","name_en":"NBA","name_desc":"NBA","sport_id":"nba","pair_poly_sport_id":"nba","league_id":1},{"name":"NCAA篮球","name_en":"NCAA Basketball","name_desc":"NCAA Basketball","sport_id":"cbb","pair_poly_sport_id":"cbb","league_id":2},{"name":"Liga ACB","name_en":"Liga ACB","name_desc":"Liga ACB","sport_id":"bkligend","pair_poly_sport_id":"bkligend","league_id":1839},{"name":"NBL","name_en":"NBL","name_desc":"NBL","sport_id":"bknbl","pair_poly_sport_id":"bknbl","league_id":1838}]}',
    `updated_at` = NOW()
WHERE `platform` = 'SXBET' AND `sub_category` = 'Basketball';

UPDATE `market_category`
SET `sport_cn` = '冰球',
    `config_json` = '{"sport_cn":"冰球","sub_category":"NHL","items":[{"name":"NHL","name_en":"NHL","name_desc":"NHL","sport_id":"nhl","pair_poly_sport_id":"nhl","league_id":3},{"name":"KHL","name_en":"KHL","name_desc":"KHL","sport_id":"khl","pair_poly_sport_id":"khl","league_id":1840}]}',
    `updated_at` = NOW()
WHERE `platform` = 'SXBET' AND `sub_category` = 'NHL';

UPDATE `market_category`
SET `sport_cn` = '美式足球',
    `config_json` = '{"sport_cn":"美式足球","sub_category":"NFL","items":[{"name":"NFL","name_en":"NFL","name_desc":"NFL","sport_id":"nfl","pair_poly_sport_id":"nfl","league_id":243},{"name":"NCAA橄榄球","name_en":"College Football","name_desc":"College Football","sport_id":"cfb","pair_poly_sport_id":"cfb","league_id":246}]}',
    `updated_at` = NOW()
WHERE `platform` = 'SXBET' AND `sub_category` = 'NFL';

UPDATE `market_category`
SET `sport_cn` = 'UFC',
    `config_json` = '{"sport_cn":"UFC","sub_category":"MMA","items":[{"name":"UFC","name_en":"UFC","name_desc":"UFC","sport_id":"ufc","pair_poly_sport_id":"ufc","league_id":34}]}',
    `updated_at` = NOW()
WHERE `platform` = 'SXBET' AND `sub_category` = 'MMA';

UPDATE `market_category`
SET `sport_cn` = '网球',
    `config_json` = '{"sport_cn":"网球","sub_category":"Tennis","items":[{"name":"ATP","name_en":"ATP Tour","name_desc":"ATP Tour","sport_id":"atp","pair_poly_sport_id":"atp","league_id":1437},{"name":"挑战赛·Metz","name_en":"Metz Challenger ATP","name_desc":"Metz Challenger ATP","sport_id":"atp","pair_poly_sport_id":"atp","league_id":1842},{"name":"挑战赛·Helsinki","name_en":"Helsinki Challenger ATP","name_desc":"Helsinki Challenger ATP","sport_id":"atp","pair_poly_sport_id":"atp","league_id":1841},{"name":"挑战赛·Champaign","name_en":"Champaign Challenger ATP","name_desc":"Champaign Challenger ATP","sport_id":"atp","pair_poly_sport_id":"atp","league_id":1847},{"name":"挑战赛·Montevideo","name_en":"Montevideo Challenger ATP","name_desc":"Montevideo Challenger ATP","sport_id":"atp","pair_poly_sport_id":"atp","league_id":1846},{"name":"挑战赛·Kobe","name_en":"Kobe Challenger ATP","name_desc":"Kobe Challenger ATP","sport_id":"atp","pair_poly_sport_id":"atp","league_id":1844},{"name":"挑战赛·Sydney","name_en":"Sydney Challenger ATP","name_desc":"Sydney Challenger ATP","sport_id":"atp","pair_poly_sport_id":"atp","league_id":1845},{"name":"挑战赛·Drummondville","name_en":"Drummondville Challenger ATP","name_desc":"Drummondville Challenger ATP","sport_id":"atp","pair_poly_sport_id":"atp","league_id":1843},{"name":"WTA","name_en":"WTA Tour","name_desc":"WTA Tour","sport_id":"wta","pair_poly_sport_id":"wta","league_id":1434}]}',
    `updated_at` = NOW()
WHERE `platform` = 'SXBET' AND `sub_category` = 'Tennis';

INSERT INTO `market_category` (`sport_cn`, `sub_category`, `platform`, `concurrency`, `config_json`, `outcome_types_json`, `sort_no`, `is_enabled`)
VALUES ('板球', 'Cricket', 'SXBET', 3, '{"sport_cn":"板球","sub_category":"Cricket","items":[{"name":"Test Cricket","name_en":"Test Cricket","name_desc":"Test Cricket","sport_id":"test","pair_poly_sport_id":"test","league_id":1200}]}', '["moneyline","handicap","total"]', 2, 1)
ON DUPLICATE KEY UPDATE
  `sport_cn` = VALUES(`sport_cn`),
  `config_json` = VALUES(`config_json`),
  `concurrency` = VALUES(`concurrency`),
  `outcome_types_json` = VALUES(`outcome_types_json`),
  `is_enabled` = VALUES(`is_enabled`),
  `updated_at` = NOW();

INSERT INTO `market_category` (`sport_cn`, `sub_category`, `platform`, `concurrency`, `config_json`, `outcome_types_json`, `sort_no`, `is_enabled`)
VALUES ('电子竞技', 'Esports', 'SXBET', 2, '{"sport_cn":"电子竞技","sub_category":"Esports","items":[{"name":"Dota 2 BLAST Slam","name_en":"Dota 2 BLAST Slam","name_desc":"Dota 2 BLAST Slam","sport_id":"dota2","league_id":102474},{"name":"Counter-Strike CCT Europe","name_en":"Counter-Strike CCT Europe","name_desc":"Counter-Strike CCT Europe","sport_id":"cs2","league_id":102471},{"name":"Counter-Strike Tipsport RUSH","name_en":"Counter-Strike Tipsport RUSH","name_desc":"Counter-Strike Tipsport RUSH","sport_id":"cs2","league_id":102496},{"name":"Overwatch OCS","name_en":"Overwatch OCS","name_desc":"Overwatch OCS","sport_id":"esports","league_id":102469},{"name":"Valorant VCT","name_en":"Valorant VCT","name_desc":"Valorant VCT","sport_id":"valorant","league_id":102466},{"name":"Valorant China National Tournament","name_en":"Valorant China National Tournament","name_desc":"Valorant China National Tournament","sport_id":"valorant","league_id":102497},{"name":"Counter-Strike PGL Major","name_en":"Counter-Strike PGL Major","name_desc":"Counter-Strike PGL Major","sport_id":"cs2","league_id":102494},{"name":"Counter-Strike ROG Journey","name_en":"Counter-Strike ROG Journey","name_desc":"Counter-Strike ROG Journey","sport_id":"cs2","league_id":102499}]}', '["moneyline"]', 8, 1)
ON DUPLICATE KEY UPDATE
  `sport_cn` = VALUES(`sport_cn`),
  `config_json` = VALUES(`config_json`),
  `concurrency` = VALUES(`concurrency`),
  `outcome_types_json` = VALUES(`outcome_types_json`),
  `is_enabled` = VALUES(`is_enabled`),
  `updated_at` = NOW();

