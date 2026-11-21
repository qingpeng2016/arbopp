#!/usr/bin/env bash
# 将空库重建所需 SQL 合并为 install_database_all.sql
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OUT="$ROOT/docs/sql/install_database_all.sql"

emit_header() {
  cat <<'HDR'
-- =============================================================================
-- install_database_all.sql — 空库一键导入（自动生成，勿手改）
-- 生成：docs/sql/generate_install_database_all.sh
-- 说明：docs/sql/INSTALL_DATABASE.md
-- =============================================================================
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

HDR
}

emit_section() {
  local title="$1" file="$2"
  {
    echo ""
    echo "-- ########## $title ##########"
    echo "-- from: docs/sql/$(basename "$file")"
    echo ""
    cat "$file"
    echo ""
  }
}

{
  emit_header
  emit_section "1/5 schema + 全局种子" "$ROOT/docs/sql/rebuild_database_full.sql"
  emit_section "2/5 market_category 联赛" "$ROOT/docs/sql/config_market_category_install.sql"
  emit_section "3/5 config_strategy_fee" "$ROOT/docs/sql/config_strategy_fee.sql"
  emit_section "4/5 predict_market_sync" "$ROOT/docs/sql/predict_market_sync.sql"
  emit_section "5/5 predict 全联赛对齐" "$ROOT/docs/sql/migration_predict_all_league_align.sql"
  echo "SET FOREIGN_KEY_CHECKS = 1;"
  echo "-- done"
} > "$OUT"

echo "Wrote $OUT ($(wc -c < "$OUT") bytes)"