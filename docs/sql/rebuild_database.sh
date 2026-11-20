#!/usr/bin/env bash
# 空库重建：表结构 + 配置（含 market_category 联赛 JSON）
# 用法：./docs/sql/rebuild_database.sh DATABASE_NAME
# 环境：MYSQL_HOST MYSQL_USER MYSQL_PWD（或交互 -p）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DB="${1:?用法: rebuild_database.sh <database_name>}"
HOST="${MYSQL_HOST:-127.0.0.1}"
USER="${MYSQL_USER:-root}"
PASS="${MYSQL_PWD:-}"
MYSQL=(mysql -h "$HOST" -u "$USER" --default-character-set=utf8mb4)
if [[ -n "$PASS" ]]; then MYSQL+=(-p"$PASS"); fi

echo "==> $DB: rebuild_database_full.sql (schema + config seeds)"
"${MYSQL[@]}" "$DB" < "$ROOT/docs/sql/rebuild_database_full.sql"
echo "==> $DB: config_market_category_install.sql (联赛同步配置，较大)"
"${MYSQL[@]}" "$DB" < "$ROOT/docs/sql/config_market_category_install.sql"
echo "==> $DB: config_strategy_fee.sql (手续费系数补全)"
"${MYSQL[@]}" "$DB" < "$ROOT/docs/sql/config_strategy_fee.sql"
echo "==> $DB: predict_market_sync.sql (Predict 平台与并发，可选)"
"${MYSQL[@]}" "$DB" < "$ROOT/docs/sql/predict_market_sync.sql"
echo "完成。请配置 account / platform.extra_config，再跑 market_sync。"