-- Predict 全子类联赛名与 POLYMARKET 对齐（config items + 存量 slug 回填）
-- 前提：migration_market_outcome_competition_en.sql 已执行
-- 已单独跑过：migration_predict_cricket_league_items.sql / migration_predict_tennis_mma_league_items.sql 可跳过对应段或重复执行无害
-- 执行后：market_map

-- ========== Basketball：补全 items（保留 nba/ncaab/fiba 原有 tag_ids）==========
UPDATE `market_category`
SET `config_json` = '{
  "sport_cn": "篮球",
  "sub_category": "Basketball",
  "sport_en": "Basketball",
  "market_variants": ["SPORTS_MATCH", "SPORTS_PROPS"],
  "items": [
    {"name": "NBA", "name_en": "NBA", "name_desc": "NBA", "sport_id": "nba", "tag_ids": ["78"]},
    {"name": "NCAA篮球", "name_en": "NCAA Basketball", "name_desc": "NCAA Basketball", "sport_id": "cbb", "tag_ids": []},
    {"name": "NCAA篮球", "name_en": "NCAA Basketball", "name_desc": "NCAA Basketball", "sport_id": "cwbb", "tag_ids": []},
    {"name": "NCAA篮球", "name_en": "NCAA Basketball", "name_desc": "NCAA Basketball", "sport_id": "ncaab", "tag_ids": ["82"]},
    {"name": "WNBA", "name_en": "WNBA", "name_desc": "WNBA", "sport_id": "wnba", "tag_ids": []},
    {"name": "FIBA", "name_en": "FIBA", "name_desc": "FIBA", "sport_id": "fiba", "tag_ids": ["98"]},
    {"name": "Liga ACB", "name_en": "Liga ACB", "name_desc": "Liga ACB", "sport_id": "bkligend", "tag_ids": []},
    {"name": "NBL", "name_en": "NBL", "name_desc": "NBL", "sport_id": "bknbl", "tag_ids": []},
    {"name": "Basketball Champions League", "name_en": "Basketball Champions League", "name_desc": "Basketball Champions League", "sport_id": "bkcl", "tag_ids": []},
    {"name": "Lega Basket Serie A", "name_en": "Lega Basket Serie A", "name_desc": "Lega Basket Serie A", "sport_id": "bkseriea", "tag_ids": []},
    {"name": "CBA", "name_en": "CBA", "name_desc": "CBA", "sport_id": "bkcba", "tag_ids": []},
    {"name": "Betclic Élite", "name_en": "Betclic Élite", "name_desc": "Betclic Élite", "sport_id": "bkfr1", "tag_ids": []},
    {"name": "Liga Nacional de Básquet", "name_en": "Liga Nacional de Básquet", "name_desc": "Liga Nacional de Básquet", "sport_id": "bkarg", "tag_ids": []},
    {"name": "KBL", "name_en": "KBL", "name_desc": "KBL", "sport_id": "bkkbl", "tag_ids": []},
    {"name": "EuroLeague", "name_en": "EuroLeague", "name_desc": "EuroLeague", "sport_id": "euroleague", "tag_ids": []},
    {"name": "EuroCup", "name_en": "EuroCup", "name_desc": "EuroCup", "sport_id": "bkeurocup", "tag_ids": []},
    {"name": "B.League", "name_en": "B.League", "name_desc": "B.League", "sport_id": "bkjpn", "tag_ids": []},
    {"name": "NBB", "name_en": "NBB", "name_desc": "NBB", "sport_id": "bknbb", "tag_ids": []},
    {"name": "VTB United League", "name_en": "VTB United League", "name_desc": "VTB United League", "sport_id": "bkvtb", "tag_ids": []},
    {"name": "ABA League", "name_en": "ABA League", "name_desc": "ABA League", "sport_id": "bkaba", "tag_ids": []},
    {"name": "Liga Argentina", "name_en": "Liga Argentina", "name_desc": "Liga Argentina", "sport_id": "bkligarg", "tag_ids": []},
    {"name": "LKL", "name_en": "LKL", "name_desc": "LKL", "sport_id": "bklkl", "tag_ids": []},
    {"name": "Basketball Bundesliga", "name_en": "Basketball Bundesliga", "name_desc": "Basketball Bundesliga", "sport_id": "bkbbl", "tag_ids": []}
  ]
}',
    `updated_at` = NOW()
WHERE `platform` = 'PREDICT' AND `sub_category` = 'Basketball';


UPDATE `market`
SET
  `competition` = CASE
    WHEN LOWER(`slug`) LIKE 'euroleague-%' OR LOWER(`slug`) = 'euroleague' THEN 'EuroLeague'
    WHEN LOWER(`slug`) LIKE 'bkeurocup-%' OR LOWER(`slug`) = 'bkeurocup' THEN 'EuroCup'
    WHEN LOWER(`slug`) LIKE 'bkligend-%' OR LOWER(`slug`) = 'bkligend' THEN 'Liga ACB'
    WHEN LOWER(`slug`) LIKE 'bkseriea-%' OR LOWER(`slug`) = 'bkseriea' THEN 'Lega Basket Serie A'
    WHEN LOWER(`slug`) LIKE 'bkligarg-%' OR LOWER(`slug`) = 'bkligarg' THEN 'Liga Argentina'
    WHEN LOWER(`slug`) LIKE 'ncaab-%' OR LOWER(`slug`) = 'ncaab' THEN 'NCAA篮球'
    WHEN LOWER(`slug`) LIKE 'bknbl-%' OR LOWER(`slug`) = 'bknbl' THEN 'NBL'
    WHEN LOWER(`slug`) LIKE 'bkcba-%' OR LOWER(`slug`) = 'bkcba' THEN 'CBA'
    WHEN LOWER(`slug`) LIKE 'bkfr1-%' OR LOWER(`slug`) = 'bkfr1' THEN 'Betclic Élite'
    WHEN LOWER(`slug`) LIKE 'bkarg-%' OR LOWER(`slug`) = 'bkarg' THEN 'Liga Nacional de Básquet'
    WHEN LOWER(`slug`) LIKE 'bkkbl-%' OR LOWER(`slug`) = 'bkkbl' THEN 'KBL'
    WHEN LOWER(`slug`) LIKE 'bkjpn-%' OR LOWER(`slug`) = 'bkjpn' THEN 'B.League'
    WHEN LOWER(`slug`) LIKE 'bknbb-%' OR LOWER(`slug`) = 'bknbb' THEN 'NBB'
    WHEN LOWER(`slug`) LIKE 'bkvtb-%' OR LOWER(`slug`) = 'bkvtb' THEN 'VTB United League'
    WHEN LOWER(`slug`) LIKE 'bkaba-%' OR LOWER(`slug`) = 'bkaba' THEN 'ABA League'
    WHEN LOWER(`slug`) LIKE 'bklkl-%' OR LOWER(`slug`) = 'bklkl' THEN 'LKL'
    WHEN LOWER(`slug`) LIKE 'bkbbl-%' OR LOWER(`slug`) = 'bkbbl' THEN 'Basketball Bundesliga'
    WHEN LOWER(`slug`) LIKE 'cwbb-%' OR LOWER(`slug`) = 'cwbb' THEN 'NCAA篮球'
    WHEN LOWER(`slug`) LIKE 'wnba-%' OR LOWER(`slug`) = 'wnba' THEN 'WNBA'
    WHEN LOWER(`slug`) LIKE 'fiba-%' OR LOWER(`slug`) = 'fiba' THEN 'FIBA'
    WHEN LOWER(`slug`) LIKE 'bkcl-%' OR LOWER(`slug`) = 'bkcl' THEN 'Basketball Champions League'
    WHEN LOWER(`slug`) LIKE 'nba-%' OR LOWER(`slug`) = 'nba' THEN 'NBA'
    WHEN LOWER(`slug`) LIKE 'cbb-%' OR LOWER(`slug`) = 'cbb' THEN 'NCAA篮球'
    ELSE `competition`
  END,
  `competition_en` = CASE
    WHEN LOWER(`slug`) LIKE 'euroleague-%' OR LOWER(`slug`) = 'euroleague' THEN 'EuroLeague'
    WHEN LOWER(`slug`) LIKE 'bkeurocup-%' OR LOWER(`slug`) = 'bkeurocup' THEN 'EuroCup'
    WHEN LOWER(`slug`) LIKE 'bkligend-%' OR LOWER(`slug`) = 'bkligend' THEN 'Liga ACB'
    WHEN LOWER(`slug`) LIKE 'bkseriea-%' OR LOWER(`slug`) = 'bkseriea' THEN 'Lega Basket Serie A'
    WHEN LOWER(`slug`) LIKE 'bkligarg-%' OR LOWER(`slug`) = 'bkligarg' THEN 'Liga Argentina'
    WHEN LOWER(`slug`) LIKE 'ncaab-%' OR LOWER(`slug`) = 'ncaab' THEN 'NCAA Basketball'
    WHEN LOWER(`slug`) LIKE 'bknbl-%' OR LOWER(`slug`) = 'bknbl' THEN 'NBL'
    WHEN LOWER(`slug`) LIKE 'bkcba-%' OR LOWER(`slug`) = 'bkcba' THEN 'CBA'
    WHEN LOWER(`slug`) LIKE 'bkfr1-%' OR LOWER(`slug`) = 'bkfr1' THEN 'Betclic Élite'
    WHEN LOWER(`slug`) LIKE 'bkarg-%' OR LOWER(`slug`) = 'bkarg' THEN 'Liga Nacional de Básquet'
    WHEN LOWER(`slug`) LIKE 'bkkbl-%' OR LOWER(`slug`) = 'bkkbl' THEN 'KBL'
    WHEN LOWER(`slug`) LIKE 'bkjpn-%' OR LOWER(`slug`) = 'bkjpn' THEN 'B.League'
    WHEN LOWER(`slug`) LIKE 'bknbb-%' OR LOWER(`slug`) = 'bknbb' THEN 'NBB'
    WHEN LOWER(`slug`) LIKE 'bkvtb-%' OR LOWER(`slug`) = 'bkvtb' THEN 'VTB United League'
    WHEN LOWER(`slug`) LIKE 'bkaba-%' OR LOWER(`slug`) = 'bkaba' THEN 'ABA League'
    WHEN LOWER(`slug`) LIKE 'bklkl-%' OR LOWER(`slug`) = 'bklkl' THEN 'LKL'
    WHEN LOWER(`slug`) LIKE 'bkbbl-%' OR LOWER(`slug`) = 'bkbbl' THEN 'Basketball Bundesliga'
    WHEN LOWER(`slug`) LIKE 'cwbb-%' OR LOWER(`slug`) = 'cwbb' THEN 'NCAA Basketball'
    WHEN LOWER(`slug`) LIKE 'wnba-%' OR LOWER(`slug`) = 'wnba' THEN 'WNBA'
    WHEN LOWER(`slug`) LIKE 'fiba-%' OR LOWER(`slug`) = 'fiba' THEN 'FIBA'
    WHEN LOWER(`slug`) LIKE 'bkcl-%' OR LOWER(`slug`) = 'bkcl' THEN 'Basketball Champions League'
    WHEN LOWER(`slug`) LIKE 'nba-%' OR LOWER(`slug`) = 'nba' THEN 'NBA'
    WHEN LOWER(`slug`) LIKE 'cbb-%' OR LOWER(`slug`) = 'cbb' THEN 'NCAA Basketball'
    ELSE `competition_en`
  END,
  `map_key` = NULL,
  `map_verified` = 0,
  `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'Basketball'
  AND `slug` IS NOT NULL
  AND TRIM(`slug`) <> '';

-- ========== NHL：Poly 7 联赛词典（仅 nhl 保留 tag 79）==========
UPDATE `market_category`
SET `config_json` = '{
  "sport_cn": "冰球",
  "sub_category": "NHL",
  "sport_en": "Ice Hockey",
  "market_variants": ["SPORTS_MATCH", "SPORTS_PROPS"],
  "items": [
    {"name": "NHL", "name_en": "NHL", "name_desc": "NHL", "sport_id": "nhl", "tag_ids": ["79"]},
    {"name": "AHL", "name_en": "AHL", "name_desc": "AHL", "sport_id": "ahl", "tag_ids": []},
    {"name": "SHL", "name_en": "SHL", "name_desc": "SHL", "sport_id": "shl", "tag_ids": []},
    {"name": "Czech Extraliga", "name_en": "Czech Extraliga", "name_desc": "Czech Extraliga", "sport_id": "cehl", "tag_ids": []},
    {"name": "DEL", "name_en": "DEL", "name_desc": "DEL", "sport_id": "dehl", "tag_ids": []},
    {"name": "National League", "name_en": "National League", "name_desc": "National League", "sport_id": "snhl", "tag_ids": []},
    {"name": "KHL", "name_en": "KHL", "name_desc": "KHL", "sport_id": "khl", "tag_ids": []}
  ]
}',
    `updated_at` = NOW()
WHERE `platform` = 'PREDICT' AND `sub_category` = 'NHL';


UPDATE `market`
SET
  `competition` = CASE
    WHEN LOWER(`slug`) LIKE 'cehl-%' OR LOWER(`slug`) = 'cehl' THEN 'Czech Extraliga'
    WHEN LOWER(`slug`) LIKE 'dehl-%' OR LOWER(`slug`) = 'dehl' THEN 'DEL'
    WHEN LOWER(`slug`) LIKE 'snhl-%' OR LOWER(`slug`) = 'snhl' THEN 'National League'
    WHEN LOWER(`slug`) LIKE 'nhl-%' OR LOWER(`slug`) = 'nhl' THEN 'NHL'
    WHEN LOWER(`slug`) LIKE 'ahl-%' OR LOWER(`slug`) = 'ahl' THEN 'AHL'
    WHEN LOWER(`slug`) LIKE 'shl-%' OR LOWER(`slug`) = 'shl' THEN 'SHL'
    WHEN LOWER(`slug`) LIKE 'khl-%' OR LOWER(`slug`) = 'khl' THEN 'KHL'
    ELSE `competition`
  END,
  `competition_en` = CASE
    WHEN LOWER(`slug`) LIKE 'cehl-%' OR LOWER(`slug`) = 'cehl' THEN 'Czech Extraliga'
    WHEN LOWER(`slug`) LIKE 'dehl-%' OR LOWER(`slug`) = 'dehl' THEN 'DEL'
    WHEN LOWER(`slug`) LIKE 'snhl-%' OR LOWER(`slug`) = 'snhl' THEN 'National League'
    WHEN LOWER(`slug`) LIKE 'nhl-%' OR LOWER(`slug`) = 'nhl' THEN 'NHL'
    WHEN LOWER(`slug`) LIKE 'ahl-%' OR LOWER(`slug`) = 'ahl' THEN 'AHL'
    WHEN LOWER(`slug`) LIKE 'shl-%' OR LOWER(`slug`) = 'shl' THEN 'SHL'
    WHEN LOWER(`slug`) LIKE 'khl-%' OR LOWER(`slug`) = 'khl' THEN 'KHL'
    ELSE `competition_en`
  END,
  `map_key` = NULL,
  `map_verified` = 0,
  `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'NHL'
  AND `slug` IS NOT NULL
  AND TRIM(`slug`) <> '';

-- ========== NFL：补 cfb（College Football）==========
UPDATE `market_category`
SET `config_json` = '{
  "sport_cn": "美式足球",
  "sub_category": "NFL",
  "sport_en": "American Football",
  "market_variants": ["SPORTS_MATCH", "SPORTS_PROPS"],
  "items": [
    {"name": "NFL", "name_en": "NFL", "name_desc": "NFL", "sport_id": "nfl", "tag_ids": ["45"]},
    {"name": "NCAA橄榄球", "name_en": "College Football", "name_desc": "College Football", "sport_id": "cfb", "tag_ids": []}
  ]
}',
    `updated_at` = NOW()
WHERE `platform` = 'PREDICT' AND `sub_category` = 'NFL';


UPDATE `market`
SET
  `competition` = CASE
    WHEN LOWER(`slug`) LIKE 'nfl-%' OR LOWER(`slug`) = 'nfl' THEN 'NFL'
    WHEN LOWER(`slug`) LIKE 'cfb-%' OR LOWER(`slug`) = 'cfb' THEN 'NCAA橄榄球'
    ELSE `competition`
  END,
  `competition_en` = CASE
    WHEN LOWER(`slug`) LIKE 'nfl-%' OR LOWER(`slug`) = 'nfl' THEN 'NFL'
    WHEN LOWER(`slug`) LIKE 'cfb-%' OR LOWER(`slug`) = 'cfb' THEN 'College Football'
    ELSE `competition_en`
  END,
  `map_key` = NULL,
  `map_verified` = 0,
  `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'NFL'
  AND `slug` IS NOT NULL
  AND TRIM(`slug`) <> '';

-- ========== Football：config 已齐，仅存量回填 ==========

UPDATE `market`
SET
  `competition` = CASE
    WHEN LOWER(`slug`) LIKE 'argcopa-%' OR LOWER(`slug`) = 'argcopa' THEN '阿根廷杯'
    WHEN LOWER(`slug`) LIKE 'copaam-%' OR LOWER(`slug`) = 'copaam' THEN '美洲杯'
    WHEN LOWER(`slug`) LIKE 'argpn-%' OR LOWER(`slug`) = 'argpn' THEN '阿根廷全国联赛'
    WHEN LOWER(`slug`) LIKE 'afcq-%' OR LOWER(`slug`) = 'afcq' THEN '非洲杯预选赛'
    WHEN LOWER(`slug`) LIKE 'bra2-%' OR LOWER(`slug`) = 'bra2' THEN '巴乙'
    WHEN LOWER(`slug`) LIKE 'chl2-%' OR LOWER(`slug`) = 'chl2' THEN '智利乙'
    WHEN LOWER(`slug`) LIKE 'mex2-%' OR LOWER(`slug`) = 'mex2' THEN '墨扩军联赛'
    WHEN LOWER(`slug`) LIKE 'chi1-%' OR LOWER(`slug`) = 'chi1' THEN '智利甲'
    WHEN LOWER(`slug`) LIKE 'nor2-%' OR LOWER(`slug`) = 'nor2' THEN '挪威甲'
    WHEN LOWER(`slug`) LIKE 'conl-%' OR LOWER(`slug`) = 'conl' THEN '中北美国家联赛'
    WHEN LOWER(`slug`) LIKE 'serc-%' OR LOWER(`slug`) = 'serc' THEN '意丙'
    WHEN LOWER(`slug`) LIKE 'itsb-%' OR LOWER(`slug`) = 'itsb' THEN '意乙'
    WHEN LOWER(`slug`) LIKE 'scop-%' OR LOWER(`slug`) = 'scop' THEN '苏超'
    WHEN LOWER(`slug`) LIKE 'wsl2-%' OR LOWER(`slug`) = 'wsl2' THEN '英格兰女足甲级'
    WHEN LOWER(`slug`) LIKE 'uwcl-%' OR LOWER(`slug`) = 'uwcl' THEN '女足欧冠'
    WHEN LOWER(`slug`) LIKE 'uslc-%' OR LOWER(`slug`) = 'uslc' THEN '美国足球冠军联赛'
    WHEN LOWER(`slug`) LIKE 'ukr1-%' OR LOWER(`slug`) = 'ukr1' THEN '乌克兰超'
    WHEN LOWER(`slug`) LIKE 'uru1-%' OR LOWER(`slug`) = 'uru1' THEN '乌拉圭甲'
    WHEN LOWER(`slug`) LIKE 'uzb1-%' OR LOWER(`slug`) = 'uzb1' THEN '乌兹别克超'
    WHEN LOWER(`slug`) LIKE 'lva1-%' OR LOWER(`slug`) = 'lva1' THEN '拉脱维亚超'
    WHEN LOWER(`slug`) LIKE 'bel1-%' OR LOWER(`slug`) = 'bel1' THEN '比甲'
    WHEN LOWER(`slug`) LIKE 'bol1-%' OR LOWER(`slug`) = 'bol1' THEN '玻利维亚甲'
    WHEN LOWER(`slug`) LIKE 'cze1-%' OR LOWER(`slug`) = 'cze1' THEN '捷克联赛'
    WHEN LOWER(`slug`) LIKE 'egy1-%' OR LOWER(`slug`) = 'egy1' THEN '埃及超'
    WHEN LOWER(`slug`) LIKE 'ned2-%' OR LOWER(`slug`) = 'ned2' THEN '荷乙'
    WHEN LOWER(`slug`) LIKE 'nwsl-%' OR LOWER(`slug`) = 'nwsl' THEN '美国女足联赛'
    WHEN LOWER(`slug`) LIKE 'snhl-%' OR LOWER(`slug`) = 'snhl' THEN '英格兰国家联赛'
    WHEN LOWER(`slug`) LIKE 'svk1-%' OR LOWER(`slug`) = 'svk1' THEN '斯洛伐克超'
    WHEN LOWER(`slug`) LIKE 'par1-%' OR LOWER(`slug`) = 'par1' THEN '巴拉圭甲'
    WHEN LOWER(`slug`) LIKE 'per1-%' OR LOWER(`slug`) = 'per1' THEN '秘鲁甲'
    WHEN LOWER(`slug`) LIKE 'rou1-%' OR LOWER(`slug`) = 'rou1' THEN '罗马尼亚超'
    WHEN LOWER(`slug`) LIKE 'uae1-%' OR LOWER(`slug`) = 'uae1' THEN '阿联酋超'
    WHEN LOWER(`slug`) LIKE 'acle-%' OR LOWER(`slug`) = 'acle' THEN '亚冠'
    WHEN LOWER(`slug`) LIKE 'col1-%' OR LOWER(`slug`) = 'col1' THEN '哥伦甲'
    WHEN LOWER(`slug`) LIKE 'idn1-%' OR LOWER(`slug`) = 'idn1' THEN '印尼联'
    WHEN LOWER(`slug`) LIKE 'ncag-%' OR LOWER(`slug`) = 'ncag' THEN '金杯赛'
    WHEN LOWER(`slug`) LIKE 'es3-%' OR LOWER(`slug`) = 'es3' THEN '西协甲'
    WHEN LOWER(`slug`) LIKE 'elc-%' OR LOWER(`slug`) = 'elc' THEN '英冠'
    WHEN LOWER(`slug`) LIKE 'el1-%' OR LOWER(`slug`) = 'el1' THEN '英甲'
    WHEN LOWER(`slug`) LIKE 'el2-%' OR LOWER(`slug`) = 'el2' THEN '英乙'
    WHEN LOWER(`slug`) LIKE 'es2-%' OR LOWER(`slug`) = 'es2' THEN '西乙'
    WHEN LOWER(`slug`) LIKE 'fr2-%' OR LOWER(`slug`) = 'fr2' THEN '法乙'
    WHEN LOWER(`slug`) LIKE 'fpd-%' OR LOWER(`slug`) = 'fpd' THEN '哥斯达黎加甲'
    WHEN LOWER(`slug`) LIKE 'gtm-%' OR LOWER(`slug`) = 'gtm' THEN '危地马拉甲'
    WHEN LOWER(`slug`) LIKE 'pt2-%' OR LOWER(`slug`) = 'pt2' THEN '葡甲'
    WHEN LOWER(`slug`) LIKE 'ptc-%' OR LOWER(`slug`) = 'ptc' THEN '葡萄牙杯'
    WHEN LOWER(`slug`) LIKE 'bul-%' OR LOWER(`slug`) = 'bul' THEN '保加利亚超'
    WHEN LOWER(`slug`) LIKE 'fif-%' OR LOWER(`slug`) = 'fif' THEN '国际足球'
    WHEN LOWER(`slug`) LIKE 'ja3-%' OR LOWER(`slug`) = 'ja3' THEN 'J3联赛'
    WHEN LOWER(`slug`) LIKE 'isp-%' OR LOWER(`slug`) = 'isp' THEN '印度超'
    WHEN LOWER(`slug`) LIKE 'clf-%' OR LOWER(`slug`) = 'clf' THEN '俱乐部友谊赛'
    WHEN LOWER(`slug`) LIKE 'bl2-%' OR LOWER(`slug`) = 'bl2' THEN '德乙'
    WHEN LOWER(`slug`) LIKE 'bl3-%' OR LOWER(`slug`) = 'bl3' THEN '德丙'
    WHEN LOWER(`slug`) LIKE 'wsl-%' OR LOWER(`slug`) = 'wsl' THEN '英格兰女足超'
    WHEN LOWER(`slug`) LIKE 'hr1-%' OR LOWER(`slug`) = 'hr1' THEN '克罗地亚甲'
    WHEN LOWER(`slug`) LIKE 'epl-%' OR LOWER(`slug`) = 'epl' THEN '英超'
    WHEN LOWER(`slug`) LIKE 'ucl-%' OR LOWER(`slug`) = 'ucl' THEN '欧冠'
    WHEN LOWER(`slug`) LIKE 'lal-%' OR LOWER(`slug`) = 'lal' THEN '西甲'
    WHEN LOWER(`slug`) LIKE 'bun-%' OR LOWER(`slug`) = 'bun' THEN '德甲'
    WHEN LOWER(`slug`) LIKE 'sea-%' OR LOWER(`slug`) = 'sea' THEN '意甲'
    WHEN LOWER(`slug`) LIKE 'fl1-%' OR LOWER(`slug`) = 'fl1' THEN '法甲'
    WHEN LOWER(`slug`) LIKE 'uel-%' OR LOWER(`slug`) = 'uel' THEN '欧联'
    WHEN LOWER(`slug`) LIKE 'ere-%' OR LOWER(`slug`) = 'ere' THEN '荷甲'
    WHEN LOWER(`slug`) LIKE 'por-%' OR LOWER(`slug`) = 'por' THEN '葡超'
    WHEN LOWER(`slug`) LIKE 'tur-%' OR LOWER(`slug`) = 'tur' THEN '土超'
    WHEN LOWER(`slug`) LIKE 'rus-%' OR LOWER(`slug`) = 'rus' THEN '俄超'
    WHEN LOWER(`slug`) LIKE 'mls-%' OR LOWER(`slug`) = 'mls' THEN '美职联'
    WHEN LOWER(`slug`) LIKE 'mex-%' OR LOWER(`slug`) = 'mex' THEN '墨超'
    WHEN LOWER(`slug`) LIKE 'bra-%' OR LOWER(`slug`) = 'bra' THEN '巴甲'
    WHEN LOWER(`slug`) LIKE 'arg-%' OR LOWER(`slug`) = 'arg' THEN '阿甲'
    WHEN LOWER(`slug`) LIKE 'spl-%' OR LOWER(`slug`) = 'spl' THEN '沙特联'
    WHEN LOWER(`slug`) LIKE 'kor-%' OR LOWER(`slug`) = 'kor' THEN 'K联赛'
    WHEN LOWER(`slug`) LIKE 'jap-%' OR LOWER(`slug`) = 'jap' THEN 'J1'
    WHEN LOWER(`slug`) LIKE 'ja2-%' OR LOWER(`slug`) = 'ja2' THEN 'J2'
    WHEN LOWER(`slug`) LIKE 'aus-%' OR LOWER(`slug`) = 'aus' THEN '澳超'
    WHEN LOWER(`slug`) LIKE 'nor-%' OR LOWER(`slug`) = 'nor' THEN '挪超'
    WHEN LOWER(`slug`) LIKE 'den-%' OR LOWER(`slug`) = 'den' THEN '丹超'
    WHEN LOWER(`slug`) LIKE 'unl-%' OR LOWER(`slug`) = 'unl' THEN '欧国联'
    WHEN LOWER(`slug`) LIKE 'lib-%' OR LOWER(`slug`) = 'lib' THEN '解放者杯'
    WHEN LOWER(`slug`) LIKE 'chi-%' OR LOWER(`slug`) = 'chi' THEN '中超'
    WHEN LOWER(`slug`) LIKE 'sud-%' OR LOWER(`slug`) = 'sud' THEN '南美杯'
    WHEN LOWER(`slug`) LIKE 'cdr-%' OR LOWER(`slug`) = 'cdr' THEN '国王杯'
    WHEN LOWER(`slug`) LIKE 'cde-%' OR LOWER(`slug`) = 'cde' THEN '法国杯'
    WHEN LOWER(`slug`) LIKE 'dfb-%' OR LOWER(`slug`) = 'dfb' THEN '德国杯'
    WHEN LOWER(`slug`) LIKE 'itc-%' OR LOWER(`slug`) = 'itc' THEN '意大利杯'
    WHEN LOWER(`slug`) LIKE 'efa-%' OR LOWER(`slug`) = 'efa' THEN '英格兰足总杯'
    WHEN LOWER(`slug`) LIKE 'efl-%' OR LOWER(`slug`) = 'efl' THEN '英格兰联赛杯'
    WHEN LOWER(`slug`) LIKE 'ssc-%' OR LOWER(`slug`) = 'ssc' THEN '西班牙超级杯'
    WHEN LOWER(`slug`) LIKE 'acn-%' OR LOWER(`slug`) = 'acn' THEN '非洲国家杯'
    WHEN LOWER(`slug`) LIKE 'col-%' OR LOWER(`slug`) = 'col' THEN '欧协联'
    WHEN LOWER(`slug`) LIKE 'pol-%' OR LOWER(`slug`) = 'pol' THEN '波兰超'
    ELSE `competition`
  END,
  `competition_en` = CASE
    WHEN LOWER(`slug`) LIKE 'argcopa-%' OR LOWER(`slug`) = 'argcopa' THEN 'Copa Argentina'
    WHEN LOWER(`slug`) LIKE 'copaam-%' OR LOWER(`slug`) = 'copaam' THEN 'Copa America'
    WHEN LOWER(`slug`) LIKE 'argpn-%' OR LOWER(`slug`) = 'argpn' THEN 'Primera Nacional'
    WHEN LOWER(`slug`) LIKE 'afcq-%' OR LOWER(`slug`) = 'afcq' THEN 'Africa Cup of Nations Qualifiers'
    WHEN LOWER(`slug`) LIKE 'bra2-%' OR LOWER(`slug`) = 'bra2' THEN 'Brasileirão Série B'
    WHEN LOWER(`slug`) LIKE 'chl2-%' OR LOWER(`slug`) = 'chl2' THEN 'Primera B (Chile)'
    WHEN LOWER(`slug`) LIKE 'mex2-%' OR LOWER(`slug`) = 'mex2' THEN 'Liga de Expansión MX'
    WHEN LOWER(`slug`) LIKE 'chi1-%' OR LOWER(`slug`) = 'chi1' THEN 'Liga de Primera'
    WHEN LOWER(`slug`) LIKE 'nor2-%' OR LOWER(`slug`) = 'nor2' THEN 'OBOS-ligaen'
    WHEN LOWER(`slug`) LIKE 'conl-%' OR LOWER(`slug`) = 'conl' THEN 'CONCACAF Nations League'
    WHEN LOWER(`slug`) LIKE 'serc-%' OR LOWER(`slug`) = 'serc' THEN 'Serie C'
    WHEN LOWER(`slug`) LIKE 'itsb-%' OR LOWER(`slug`) = 'itsb' THEN 'Serie B'
    WHEN LOWER(`slug`) LIKE 'scop-%' OR LOWER(`slug`) = 'scop' THEN 'Scottish Premiership'
    WHEN LOWER(`slug`) LIKE 'wsl2-%' OR LOWER(`slug`) = 'wsl2' THEN 'Women''s Super League 2'
    WHEN LOWER(`slug`) LIKE 'uwcl-%' OR LOWER(`slug`) = 'uwcl' THEN 'UEFA Women''s Champions League'
    WHEN LOWER(`slug`) LIKE 'uslc-%' OR LOWER(`slug`) = 'uslc' THEN 'USL Championship'
    WHEN LOWER(`slug`) LIKE 'ukr1-%' OR LOWER(`slug`) = 'ukr1' THEN 'Ukrainian Premier League'
    WHEN LOWER(`slug`) LIKE 'uru1-%' OR LOWER(`slug`) = 'uru1' THEN 'Uruguayan Primera División'
    WHEN LOWER(`slug`) LIKE 'uzb1-%' OR LOWER(`slug`) = 'uzb1' THEN 'Uzbekistan Super League'
    WHEN LOWER(`slug`) LIKE 'lva1-%' OR LOWER(`slug`) = 'lva1' THEN 'Virslīga'
    WHEN LOWER(`slug`) LIKE 'bel1-%' OR LOWER(`slug`) = 'bel1' THEN 'Belgium Pro League'
    WHEN LOWER(`slug`) LIKE 'bol1-%' OR LOWER(`slug`) = 'bol1' THEN 'Bolivian Primera División'
    WHEN LOWER(`slug`) LIKE 'cze1-%' OR LOWER(`slug`) = 'cze1' THEN 'Chance Liga'
    WHEN LOWER(`slug`) LIKE 'egy1-%' OR LOWER(`slug`) = 'egy1' THEN 'Egyptian Premier League'
    WHEN LOWER(`slug`) LIKE 'ned2-%' OR LOWER(`slug`) = 'ned2' THEN 'Eerste Divisie'
    WHEN LOWER(`slug`) LIKE 'nwsl-%' OR LOWER(`slug`) = 'nwsl' THEN 'NWSL'
    WHEN LOWER(`slug`) LIKE 'snhl-%' OR LOWER(`slug`) = 'snhl' THEN 'National League'
    WHEN LOWER(`slug`) LIKE 'svk1-%' OR LOWER(`slug`) = 'svk1' THEN 'Niké liga'
    WHEN LOWER(`slug`) LIKE 'par1-%' OR LOWER(`slug`) = 'par1' THEN 'Paraguay Primera División'
    WHEN LOWER(`slug`) LIKE 'per1-%' OR LOWER(`slug`) = 'per1' THEN 'Liga 1'
    WHEN LOWER(`slug`) LIKE 'rou1-%' OR LOWER(`slug`) = 'rou1' THEN 'SuperLiga'
    WHEN LOWER(`slug`) LIKE 'uae1-%' OR LOWER(`slug`) = 'uae1' THEN 'UAE Pro League'
    WHEN LOWER(`slug`) LIKE 'acle-%' OR LOWER(`slug`) = 'acle' THEN 'AFC Champions League Elite'
    WHEN LOWER(`slug`) LIKE 'col1-%' OR LOWER(`slug`) = 'col1' THEN 'Categoría Primera A'
    WHEN LOWER(`slug`) LIKE 'idn1-%' OR LOWER(`slug`) = 'idn1' THEN 'Indonesia Liga 1'
    WHEN LOWER(`slug`) LIKE 'ncag-%' OR LOWER(`slug`) = 'ncag' THEN 'CONCACAF Gold Cup'
    WHEN LOWER(`slug`) LIKE 'es3-%' OR LOWER(`slug`) = 'es3' THEN 'Primera Federación'
    WHEN LOWER(`slug`) LIKE 'elc-%' OR LOWER(`slug`) = 'elc' THEN 'EFL Championship'
    WHEN LOWER(`slug`) LIKE 'el1-%' OR LOWER(`slug`) = 'el1' THEN 'League One'
    WHEN LOWER(`slug`) LIKE 'el2-%' OR LOWER(`slug`) = 'el2' THEN 'League Two'
    WHEN LOWER(`slug`) LIKE 'es2-%' OR LOWER(`slug`) = 'es2' THEN 'LaLiga2'
    WHEN LOWER(`slug`) LIKE 'fr2-%' OR LOWER(`slug`) = 'fr2' THEN 'Ligue 2'
    WHEN LOWER(`slug`) LIKE 'fpd-%' OR LOWER(`slug`) = 'fpd' THEN 'Liga FPD'
    WHEN LOWER(`slug`) LIKE 'gtm-%' OR LOWER(`slug`) = 'gtm' THEN 'Liga Nacional de Guatemala'
    WHEN LOWER(`slug`) LIKE 'pt2-%' OR LOWER(`slug`) = 'pt2' THEN 'Liga Portugal 2'
    WHEN LOWER(`slug`) LIKE 'ptc-%' OR LOWER(`slug`) = 'ptc' THEN 'Taça de Portugal'
    WHEN LOWER(`slug`) LIKE 'bul-%' OR LOWER(`slug`) = 'bul' THEN 'efbet Liga'
    WHEN LOWER(`slug`) LIKE 'fif-%' OR LOWER(`slug`) = 'fif' THEN 'FIFA Friendlies'
    WHEN LOWER(`slug`) LIKE 'ja3-%' OR LOWER(`slug`) = 'ja3' THEN 'J3 League'
    WHEN LOWER(`slug`) LIKE 'isp-%' OR LOWER(`slug`) = 'isp' THEN 'Indian Super League'
    WHEN LOWER(`slug`) LIKE 'clf-%' OR LOWER(`slug`) = 'clf' THEN 'Club Friendlies'
    WHEN LOWER(`slug`) LIKE 'bl2-%' OR LOWER(`slug`) = 'bl2' THEN '2. Bundesliga'
    WHEN LOWER(`slug`) LIKE 'bl3-%' OR LOWER(`slug`) = 'bl3' THEN '3. Liga'
    WHEN LOWER(`slug`) LIKE 'wsl-%' OR LOWER(`slug`) = 'wsl' THEN 'Women''s Super League'
    WHEN LOWER(`slug`) LIKE 'hr1-%' OR LOWER(`slug`) = 'hr1' THEN 'Prva Liga'
    WHEN LOWER(`slug`) LIKE 'epl-%' OR LOWER(`slug`) = 'epl' THEN 'Premier League'
    WHEN LOWER(`slug`) LIKE 'ucl-%' OR LOWER(`slug`) = 'ucl' THEN 'UEFA Champions League'
    WHEN LOWER(`slug`) LIKE 'lal-%' OR LOWER(`slug`) = 'lal' THEN 'LaLiga'
    WHEN LOWER(`slug`) LIKE 'bun-%' OR LOWER(`slug`) = 'bun' THEN 'Bundesliga'
    WHEN LOWER(`slug`) LIKE 'sea-%' OR LOWER(`slug`) = 'sea' THEN 'Serie A'
    WHEN LOWER(`slug`) LIKE 'fl1-%' OR LOWER(`slug`) = 'fl1' THEN 'Ligue 1'
    WHEN LOWER(`slug`) LIKE 'uel-%' OR LOWER(`slug`) = 'uel' THEN 'UEFA Europa League'
    WHEN LOWER(`slug`) LIKE 'ere-%' OR LOWER(`slug`) = 'ere' THEN 'Eredivisie'
    WHEN LOWER(`slug`) LIKE 'por-%' OR LOWER(`slug`) = 'por' THEN 'Primeira Liga'
    WHEN LOWER(`slug`) LIKE 'tur-%' OR LOWER(`slug`) = 'tur' THEN 'Süper Lig'
    WHEN LOWER(`slug`) LIKE 'rus-%' OR LOWER(`slug`) = 'rus' THEN 'Russian Premier League'
    WHEN LOWER(`slug`) LIKE 'mls-%' OR LOWER(`slug`) = 'mls' THEN 'MLS'
    WHEN LOWER(`slug`) LIKE 'mex-%' OR LOWER(`slug`) = 'mex' THEN 'Liga MX'
    WHEN LOWER(`slug`) LIKE 'bra-%' OR LOWER(`slug`) = 'bra' THEN 'Brasileirão Série A'
    WHEN LOWER(`slug`) LIKE 'arg-%' OR LOWER(`slug`) = 'arg' THEN 'Liga Profesional de Fútbol'
    WHEN LOWER(`slug`) LIKE 'spl-%' OR LOWER(`slug`) = 'spl' THEN 'Saudi Pro League'
    WHEN LOWER(`slug`) LIKE 'kor-%' OR LOWER(`slug`) = 'kor' THEN 'K League 1'
    WHEN LOWER(`slug`) LIKE 'jap-%' OR LOWER(`slug`) = 'jap' THEN 'J1 League'
    WHEN LOWER(`slug`) LIKE 'ja2-%' OR LOWER(`slug`) = 'ja2' THEN 'J2 League'
    WHEN LOWER(`slug`) LIKE 'aus-%' OR LOWER(`slug`) = 'aus' THEN 'A-League Men'
    WHEN LOWER(`slug`) LIKE 'nor-%' OR LOWER(`slug`) = 'nor' THEN 'Eliteserien'
    WHEN LOWER(`slug`) LIKE 'den-%' OR LOWER(`slug`) = 'den' THEN 'Danish Superliga'
    WHEN LOWER(`slug`) LIKE 'unl-%' OR LOWER(`slug`) = 'unl' THEN 'UEFA Nations League'
    WHEN LOWER(`slug`) LIKE 'lib-%' OR LOWER(`slug`) = 'lib' THEN 'Copa Libertadores'
    WHEN LOWER(`slug`) LIKE 'chi-%' OR LOWER(`slug`) = 'chi' THEN 'Chinese Super League'
    WHEN LOWER(`slug`) LIKE 'sud-%' OR LOWER(`slug`) = 'sud' THEN 'Copa Sudamericana'
    WHEN LOWER(`slug`) LIKE 'cdr-%' OR LOWER(`slug`) = 'cdr' THEN 'Copa del Rey'
    WHEN LOWER(`slug`) LIKE 'cde-%' OR LOWER(`slug`) = 'cde' THEN 'Coupe de France'
    WHEN LOWER(`slug`) LIKE 'dfb-%' OR LOWER(`slug`) = 'dfb' THEN 'DFB-Pokal'
    WHEN LOWER(`slug`) LIKE 'itc-%' OR LOWER(`slug`) = 'itc' THEN 'Coppa Italia'
    WHEN LOWER(`slug`) LIKE 'efa-%' OR LOWER(`slug`) = 'efa' THEN 'FA Cup'
    WHEN LOWER(`slug`) LIKE 'efl-%' OR LOWER(`slug`) = 'efl' THEN 'EFL CUP'
    WHEN LOWER(`slug`) LIKE 'ssc-%' OR LOWER(`slug`) = 'ssc' THEN 'Supercopa de España'
    WHEN LOWER(`slug`) LIKE 'acn-%' OR LOWER(`slug`) = 'acn' THEN 'Africa Cup of Nations'
    WHEN LOWER(`slug`) LIKE 'col-%' OR LOWER(`slug`) = 'col' THEN 'UEFA Conference League'
    WHEN LOWER(`slug`) LIKE 'pol-%' OR LOWER(`slug`) = 'pol' THEN 'Ekstraklasa'
    ELSE `competition_en`
  END,
  `map_key` = NULL,
  `map_verified` = 0,
  `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'Football'
  AND `slug` IS NOT NULL
  AND TRIM(`slug`) <> '';

-- ========== market_outcome ==========
UPDATE `market_outcome` AS mo
INNER JOIN `market` AS m ON m.`id` = mo.`market_row_id`
SET
  mo.`competition` = m.`competition`,
  mo.`competition_en` = m.`competition_en`,
  mo.`updated_at` = NOW()
WHERE m.`platform` = 'PREDICT'
  AND m.`sub_category` IN ('Football', 'Basketball', 'Cricket', 'Tennis', 'NFL', 'NHL', 'MMA');

-- SELECT platform, sub_category, competition_en, COUNT(*) FROM market
-- WHERE platform='PREDICT' GROUP BY 1,2,3 ORDER BY 2,3;

