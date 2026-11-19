# market_category

一行 = **运动子类 + 一个平台**（如 Football + KALSHI）。足球默认 **3 行**：POLYMARKET / KALSHI / PREDICT。

- 建表：`migration_config_market_category.sql`
- 数据：`migration_config_market_category_data.sql`（3 条 INSERT 值）
- 一键：`config_market_category_install.sql`
- 生成：`go run ./cmd/gen_market_category_sql/main.go`