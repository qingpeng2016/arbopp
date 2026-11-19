#!/usr/bin/env python3
"""生成 market_category_polymarket.json / market_category_predict.json（Poly + Predict，7 个运动子类）。"""
from __future__ import annotations

import json
import urllib.request
from pathlib import Path

DOCS = Path(__file__).resolve().parents[1]
MANIFEST = DOCS / "market_category_manifest.json"
PREDICT_ROOTS = DOCS / "predict_sport_roots.json"
POLY_API_CACHE = DOCS / ".polymarket_sports_api.json"
POLY_FOOTBALL = DOCS / "polymarket_sports.json"
PREDICT_FOOTBALL = DOCS / "predict_sports.json"
OUT_POLY = DOCS / "market_category_polymarket.json"
OUT_PRED = DOCS / "market_category_predict.json"

# 与 common/enum/crypto/prediction/polymarket_enum_tool.go 一致
POLY_SPORT_IDS: dict[str, list[str]] = {
    "Football": [
        "epl", "lal", "bun", "fl1", "sea", "ucl", "uel", "mls", "fif", "uef", "con", "caf", "cof", "bra", "arg",
        "mex", "lib", "sud", "tur", "rus", "efa", "efl", "ere", "itc", "cdr", "dfb", "cde", "col", "afc", "ofc",
        "jap", "ja2", "kor", "spl", "chi", "aus", "idn1", "nor", "den", "por", "acn", "ssc",
    ],
    "Basketball": [
        "nba", "ncaab", "cbb", "wnba", "cwbb", "bkligend", "bknbl", "bkcl", "bkseriea", "bkcba", "bkfr1", "bkarg", "bkkbl",
    ],
    "NFL": ["nfl", "cfb"],
    "NHL": ["nhl", "ahl", "shl", "cehl", "dehl", "snhl", "khl"],
    "Tennis": ["atp", "wta"],
    "MMA": ["ufc"],
    "Cricket": [
        "ipl", "odi", "t20", "abb", "csa", "crban", "crint", "craus", "creng", "crnew", "crind", "crsou", "crpak", "cruae",
        "test", "she", "sasa", "lpl", "psp", "cru19wc", "crwpl20",
    ],
}

SPORT_EN = {
    "Football": "Soccer",
    "Basketball": "Basketball",
    "Cricket": "Cricket",
    "Tennis": "Tennis",
    "NFL": "American Football",
    "NHL": "Ice Hockey",
    "MMA": "UFC",
}

# Poly 联赛中文名（与 predict 足球配置对齐）；未列出则用 API 英文名
LEAGUE_EN: dict[str, str] = {
    "ncaab": "NCAA Basketball",
    "cbb": "NCAA Basketball",
    "cwbb": "NCAA Basketball",
    "atp": "ATP Tour",
    "wta": "WTA Tour",
}

LEAGUE_CN: dict[str, str] = {
    "nfl": "NFL",
    "cfb": "NCAA橄榄球",
    "nba": "NBA",
    "ncaab": "NCAA篮球",
    "cbb": "NCAA篮球",
    "wnba": "WNBA",
    "cwbb": "NCAA篮球",
    "nhl": "NHL",
    "ahl": "AHL",
    "atp": "ATP",
    "wta": "WTA",
    "ufc": "UFC",
    "ipl": "IPL",
    "odi": "ODI",
    "t20": "T20",
    "abb": "Big Bash League",
    "test": "Test Cricket",
    "ucl": "欧冠",
    "epl": "英超",
}


def load_poly_meta() -> dict[str, dict]:
    if POLY_API_CACHE.exists():
        raw = POLY_API_CACHE.read_text(encoding="utf-8")
    else:
        with urllib.request.urlopen("https://gamma-api.polymarket.com/sports", timeout=60) as r:
            raw = r.read().decode()
        POLY_API_CACHE.write_text(raw, encoding="utf-8")
    data = json.loads(raw)
    out = {}
    for row in data:
        sid = str(row.get("sport") or "").lower()
        if sid:
            out[sid] = row
    return out


def poly_group(sport_cn: str, sub: str, poly_meta: dict[str, dict], football_group: dict | None) -> dict:
    if sub == "Football" and football_group:
        return dict(football_group)

    items = []
    for sid in POLY_SPORT_IDS.get(sub, []):
        meta = poly_meta.get(sid)
        if not meta:
            continue
        en = LEAGUE_EN.get(sid, (meta.get("name") or sid).strip())
        cn = LEAGUE_CN.get(sid, en)
        item = {
            "name": cn,
            "name_en": en,
            "name_desc": en,
            "sport_id": sid,
        }
        ptag = meta.get("primaryTagId")
        if ptag:
            try:
                item["tag_ids"] = [int(ptag)]
            except (TypeError, ValueError):
                pass
        items.append(item)

    g = {"sport_cn": sport_cn, "sub_category": sub, "items": items}
    if sub == "Football":
        g["sync_all_soccer_games"] = True
        g["soccer_game_tag_ids"] = []
    return g


def predict_group(sport_cn: str, sub: str, football_group: dict | None, roots: dict) -> dict:
    if sub == "Football" and football_group:
        return dict(football_group)

    spec = (roots.get("by_sub_category") or {}).get(sub)
    if not spec:
        raise SystemExit(f"predict_sport_roots.json 缺少 {sub}，请先运行 fetch_predict_sport_roots.py")

    g = {
        "sport_cn": sport_cn,
        "sub_category": sub,
        "sport_en": SPORT_EN.get(sub, sub),
    }
    if spec.get("market_variants"):
        g["market_variants"] = spec["market_variants"]
    if spec.get("sync_all_soccer"):
        g["sync_all_soccer"] = True
        g["soccer_root_tag_id"] = spec.get("soccer_root_tag_id", "14")
    items = spec.get("items") or []
    g["items"] = items
    if spec.get("sync_all_sub") and not items:
        g["sync_all_sub"] = True
        g["sync_root_tag_id"] = spec["sync_root_tag_id"]
    if not items and not spec.get("sync_all_soccer") and not spec.get("sync_all_sub"):
        raise SystemExit(f"{sub} predict 配置无 items 且无 sync 根 tag")
    return g


def main() -> None:
    manifest = json.loads(MANIFEST.read_text(encoding="utf-8"))
    poly_meta = load_poly_meta()
    football_poly = json.loads(POLY_FOOTBALL.read_text(encoding="utf-8"))[0]
    football_pred = json.loads(PREDICT_FOOTBALL.read_text(encoding="utf-8"))[0]

    roots: dict = {}
    if PREDICT_ROOTS.exists():
        roots = json.loads(PREDICT_ROOTS.read_text(encoding="utf-8"))

    poly_out = []
    pred_out = []
    for row in sorted(manifest, key=lambda x: x["sort_no"]):
        sub = row["sub_category"]
        sport_cn = row["sport_cn"]
        poly_out.append(poly_group(sport_cn, sub, poly_meta, football_poly))
        pred_out.append(predict_group(sport_cn, sub, football_pred, roots))

    OUT_POLY.write_text(json.dumps(poly_out, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    OUT_PRED.write_text(json.dumps(pred_out, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"wrote {OUT_POLY} ({len(poly_out)} groups)")
    print(f"wrote {OUT_PRED} ({len(pred_out)} groups)")


if __name__ == "__main__":
    main()