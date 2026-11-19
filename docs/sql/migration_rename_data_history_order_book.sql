-- 若已执行旧版 migration 创建了 history_data_order_book，执行本句重命名
RENAME TABLE `history_data_order_book` TO `data_history_order_book`;