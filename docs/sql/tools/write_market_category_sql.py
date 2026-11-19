#!/usr/bin/env python3
"""等价于 go run ./cmd/gen_market_category_sql（无 Go 模块下载时本地生成 SQL）。"""
from __future__ import annotations

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))

DOCS = Path(__file__).resolve().parents[1]


def sql_str(s: str) -> str:
    s = s.replace("\\", "\\\\").replace("'", "''")
    return "'" + s + "'"


def main() -> None:
    manifest = json.loads((DOCS / "market_category_manifest.json").read_text(encoding="utf-8"))
    poly = json.loads((DOCS / "market_category_polymarket.json").read_text(encoding="utf-8"))
    pred = json.loads((DOCS / "market_category_predict.json").read_text(encoding="utf-8"))
    poly_by = {g["sub_category"]: g for g in poly}
    pred_by = {g["sub_category"]: g for g in pred}
    manifest = sorted(manifest, key=lambda x: x["sort_no"])

    rows = []
    rid = 0
    for m in manifest:
        sub = m["sub_category"]
        otypes = json.dumps(m["outcome_types"], ensure_ascii=False, separators=(",", ":"))
        sort_no = m["sort_no"]
        sport_cn = m["sport_cn"]
        for plat, conc, grp in (
            ("POLYMARKET", m["polymarket_concurrency"], poly_by[sub]),
            ("PREDICT", m["predict_concurrency"], pred_by[sub]),
        ):
            rid += 1
            cfg = json.dumps(grp, ensure_ascii=False, separators=(",", ":"))
            rows.append((rid, sport_cn, sub, plat, conc, cfg, otypes, sort_no))

    lines = [
        "INSERT INTO `market_category` (`id`, `sport_cn`, `sub_category`, `platform`, `concurrency`, `config_json`, `outcome_types_json`, `sort_no`, `is_enabled`, `created_at`, `updated_at`)",
        "VALUES",
    ]
    for i, row in enumerate(rows):
        comma = "," if i < len(rows) - 1 else ""
        lines.append(
            f"\t({row[0]}, {sql_str(row[1])}, {sql_str(row[2])}, {sql_str(row[3])}, {row[4]}, "
            f"{sql_str(row[5])}, {sql_str(row[6])}, {row[7]}, 1, NOW(), NOW()){comma}"
        )
    lines += [
        "ON DUPLICATE KEY UPDATE",
        "`sport_cn`=VALUES(`sport_cn`), `concurrency`=VALUES(`concurrency`), `config_json`=VALUES(`config_json`),",
        "`outcome_types_json`=VALUES(`outcome_types_json`),",
        "`sort_no`=VALUES(`sort_no`), `is_enabled`=1, `updated_at`=NOW();",
        "",
    ]
    data_sql = "\n".join(lines)

    ddl_path = DOCS / "migration_config_market_category.sql"
    data_path = DOCS / "migration_config_market_category_data.sql"
    install_path = DOCS / "config_market_category_install.sql"

    data_path.write_text(data_sql, encoding="utf-8")
    install_path.write_text(ddl_path.read_text(encoding="utf-8") + "\n" + data_sql, encoding="utf-8")
    print(f"wrote {data_path}")
    print(f"wrote {install_path}")


if __name__ == "__main__":
    main()