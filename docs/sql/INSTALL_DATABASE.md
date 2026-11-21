# 删库 / 空库后完整恢复

## 推荐：一条 SQL（表结构 + 联赛配置 + 手续费 + Predict + 全联赛对齐）

```bash
# 1. 建库（库名按 conf 配置，如 fgmm_strategy_less_one）
mysql -h HOST -u USER -p -e \
  "CREATE DATABASE IF NOT EXISTS fgmm_strategy_less_one CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 2. 导入（约 115KB，按顺序内嵌 5 段子脚本）
mysql -h HOST -u USER -p --default-character-set=utf8mb4 fgmm_strategy_less_one \
  < docs/sql/install_database_all.sql
```

或使用脚本（与上面等价，分文件执行）：

```bash
export MYSQL_HOST=127.0.0.1   # 服务器上连本机 MySQL 时用 127.0.0.1
export MYSQL_USER=mm
export MYSQL_PWD='...'
./docs/sql/rebuild_database.sh fgmm_strategy_less_one
# 再单独跑联赛对齐（install_database_all 已含）：
# mysql ... fgmm_strategy_less_one < docs/sql/migration_predict_all_league_align.sql
```

## 本 bundle 包含什么

| 段 | 来源 | 说明 |
|----|------|------|
| 1 | `rebuild_database_full.sql` | 全部表 + 平台/定时任务/阈值等种子；`market_order_book` / `_ws` 为 **历史插入**、`snapshot_at_ms`、**无** token 唯一键 |
| 2 | `config_market_category_install.sql` | `market_category` 足篮网等联赛 JSON |
| 3 | `config_strategy_fee.sql` | Poly/Kalshi/Predict 吃单系数 |
| 4 | `predict_market_sync.sql` | Predict 平台行 + `predict_sports` 占位 |
| 5 | `migration_predict_all_league_align.sql` | Predict 各子类 items 与 Poly 对齐 + slug 回填 |

**不含**（需业务重跑）：`market`、`market_outcome`、`market_order_book*`、`stat_kv`、`account` 密钥、`platform.extra_config`。

## 导入后必做

1. **账号**：`INSERT` 或界面配置 `account`（Poly/Kalshi/Predict API）。
2. **Predict**：`platform.slug=PREDICT` 填好 `extra_config` 或 `account.api_key` 后 `status=ACTIVE`。
3. **定时任务**（`bot_schedule_config`）按环境改 `is_enabled` / `is_primary_machine_run`：
   - 生产 REST 盘口：开 `orderbook` 或 `orderbook_sync`（二选一，代码任务名不同）。
   - WS 落库：`ws_book_flush` + `strategy_entry`。
   - 市场：`market_sync`、`market_map`、`stat_collect`。
4. **Redis**：`strategy_entry` / `ws_book_flush` 需可连 Redis（本机开发若 Redis 只 bind 127.0.0.1，需 SSH 隧道或改 bind）。

## 旧库增量迁移（未删库时）

勿跑 `install_database_all.sql`。按时间只跑 `docs/sql/migration_*.sql`，尤其：

- `migration_market_order_book_snapshot_ms_only.sql` — 去 `snapshot_second`、加 `snapshot_at_ms`
- `migration_market_order_book_history.sql` — 指向上面的 DROP 唯一索引说明
- `migration_bot_schedule_ws_book_flush.sql` — 若无 `ws_book_flush` 任务

## 重新生成单文件

```bash
./docs/sql/generate_install_database_all.sh
```