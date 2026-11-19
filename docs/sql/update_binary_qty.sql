-- 下单份额改为配置值。较薄一边挂单份额 × 倍数 < 下单份额则不下单。
DELETE FROM `config_strategy_threshold` WHERE `config_key` = 'binary_yes_no_max_qty';

UPDATE `config_strategy_threshold`
SET `config_key` = 'binary_yes_no_order_amount',
    `description` = '命中后这一对下单的金额，按两边卖一换算成份额'
WHERE `config_key` = 'binary_yes_no_min_qty';

UPDATE `config_strategy_threshold`
SET `config_key` = 'binary_yes_no_level_factor'
WHERE `config_key` = 'binary_yes_no_size_factor';
