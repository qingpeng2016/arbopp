-- predict_sports：tag_ids 来自主网 GET /v1/tags（Soccer 树下联赛标签，2026-11-16 核对）
-- account: platform=PREDICT + ACTIVE 的 api_key 可复现拉取

UPDATE `config_strategy`
SET
  `item_value` = '[{"sport_cn":"足球","sub_category":"Football","market_variants":["SPORTS_MATCH","SPORTS_TEAM_MATCH"],"items":[{"name":"欧国联","name_en":"UEFA Nations League","sport_id":"unl","tag_ids":["2417"]},{"name":"英超","name_en":"Premier League","sport_id":"epl","tag_ids":["546"]},{"name":"欧冠","name_en":"UEFA Champions League","sport_id":"ucl","tag_ids":["342"]},{"name":"西甲","name_en":"LaLiga","sport_id":"lal","tag_ids":["852"]},{"name":"德甲","name_en":"Bundesliga","sport_id":"bun","tag_ids":["80"]},{"name":"意甲","name_en":"Serie A","sport_id":"sea","tag_ids":["545"]},{"name":"法甲","name_en":"Ligue 1","sport_id":"fl1","tag_ids":["544"]},{"name":"欧联","name_en":"UEFA Europa League","sport_id":"uel","tag_ids":["343"]},{"name":"美职联","name_en":"MLS","sport_id":"mls","tag_ids":["376"]}]}]',
  `value_type` = 'json',
  `description` = 'Predict 足球联赛；tag 名 EPL/UCL/UEL/UEFA Nations League 等（见 GET /v1/tags parent=14 Soccer）',
  `is_enabled` = 1,
  `updated_at` = NOW()
WHERE `group_name` = 'market_sync'
  AND `item_key` = 'predict_sports';