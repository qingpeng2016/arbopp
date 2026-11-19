-- 已废弃：请改用 docs/sql/update_polymarket_sports_full.sql（91 项 + name_en + 修正 sport_id）
-- 本文件仅保留作历史参考。

UPDATE `config_strategy`
SET
  `item_value` = '[{"sport_cn":"足球","sub_category":"Football","sync_all_soccer_games":true,"soccer_game_tag_ids":[],"items":[{"name":"英超","sport_id":"epl","pair_kalshi_ticker":"KXEPLGAME"},{"name":"欧冠","sport_id":"ucl","pair_kalshi_ticker":"KXUCLGAME","tag_ids":[100977]},{"name":"西甲","sport_id":"lal","pair_kalshi_ticker":"KXLALIGAGAME"},{"name":"德甲","sport_id":"bun","pair_kalshi_ticker":"KXBUNDESLIGAGAME"},{"name":"意甲","sport_id":"sea","pair_kalshi_ticker":"KXSERIEAGAME"},{"name":"法甲","sport_id":"fl1"},{"name":"欧联","sport_id":"uel"},{"name":"美职联","sport_id":"mls"},{"name":"国际足球","sport_id":"fif"},{"name":"欧国联","sport_id":"uef"},{"name":"美洲杯","sport_id":"con"},{"name":"非洲杯","sport_id":"caf"},{"name":"金杯赛","sport_id":"cof"},{"name":"巴甲","sport_id":"bra"},{"name":"阿甲","sport_id":"arg"},{"name":"墨超","sport_id":"mex"},{"name":"解放者杯","sport_id":"lib"},{"name":"南美杯","sport_id":"sud"},{"name":"土超","sport_id":"tur"},{"name":"俄超","sport_id":"rus"},{"name":"英格兰足总杯","sport_id":"efa"},{"name":"英格兰联赛杯","sport_id":"efl"},{"name":"荷甲","sport_id":"ere"},{"name":"意大利杯","sport_id":"itc"},{"name":"国王杯","sport_id":"cdr"},{"name":"德国杯","sport_id":"dfb"},{"name":"法国杯","sport_id":"cde"},{"name":"哥伦甲","sport_id":"col"},{"name":"亚冠","sport_id":"afc"},{"name":"大洋洲","sport_id":"ofc"},{"name":"J1","sport_id":"jap"},{"name":"J2","sport_id":"ja2"},{"name":"K联赛","sport_id":"kor"},{"name":"沙特联","sport_id":"spl"},{"name":"中超","sport_id":"chi"},{"name":"澳超","sport_id":"aus"},{"name":"印尼联","sport_id":"idn1"},{"name":"挪超","sport_id":"nor"},{"name":"丹超","sport_id":"den"},{"name":"葡超","sport_id":"por"},{"name":"非洲国家杯","sport_id":"acn"},{"name":"西班牙超级杯","sport_id":"ssc"}]}]',
  `value_type` = 'json',
  `description` = 'Polymarket 足球联赛 JSON（见 docs/sql/polymarket_sports.json）',
  `sort_no` = 2,
  `is_enabled` = 1,
  `updated_at` = NOW()
WHERE `group_name` = 'market_sync'
  AND `item_key` = 'polymarket_sports';