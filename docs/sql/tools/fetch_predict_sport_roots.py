#!/usr/bin/env python3
"""从 Predict GET /v1/tags 拉取运动根 tag 与常用联赛 tag，写入 predict_sport_roots.json。

用法:
  PREDICT_API_KEY=你的key python3 docs/sql/tools/fetch_predict_sport_roots.py
  PREDICT_API_BASE=https://api.predict.fun/v1   # 默认主网
"""
from __future__ import annotations

import json
import os
import sys
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "predict_sport_roots.json"

API_BASE = os.environ.get("PREDICT_API_BASE", "https://api.predict.fun/v1").rstrip("/")
API_KEY = os.environ.get("PREDICT_API_KEY", "").strip()

# Sports(4) 下 level-2 tag.name -> 业务 sub_category
L2_SPORTS = {
    "Soccer": "Football",
    "Basketball": "Basketball",
    "Football": "NFL",
    "Tennis": "Tennis",
    "Cricket": "Cricket",
}

# sub_category -> (L2 tag 名, 可选 L3 联赛 tag 名，仅配置 items 不扫全树)
SUB_LEAGUE_UNDER_L2 = {
    "NHL": ("Hockey", "NHL"),
}

# 仅 sync 根 tag（整棵子树）
SUB_SYNC_ROOT_L2 = {
    "MMA": "UFC",  # UI 为 UFC，Predict 主网为 Sports -> UFC
}

BASKETBALL_LEAGUES = [
    ("NBA", "nba", "NBA", "NBA"),
    ("NCAAM", "ncaab", "NCAA篮球", "NCAA Basketball"),
    ("FIBA", "fiba", "FIBA", "FIBA"),
]

NFL_LEAGUES = [("NFL", "nfl", "NFL", "NFL")]

MATCH_PROPS = ["SPORTS_MATCH", "SPORTS_PROPS"]
MATCH_TEAM_PROPS = ["SPORTS_MATCH", "SPORTS_TEAM_MATCH", "SPORTS_PROPS"]


def fetch_tags() -> list[dict]:
    req = urllib.request.Request(
        f"{API_BASE}/tags",
        headers={"User-Agent": "fgmm-market-category/1.0", "x-api-key": API_KEY},
    )
    with urllib.request.urlopen(req, timeout=60) as r:
        body = json.load(r)
    if not body.get("success"):
        raise RuntimeError(body)
    return body["data"]


def sync_root_entry(root_id: str) -> dict:
    return {
        "sync_all_sub": True,
        "sync_root_tag_id": root_id,
        "items": [],
    }


def items_entry(items: list[dict]) -> dict:
    return {"items": items}


def main() -> int:
    if not API_KEY and "testnet" not in API_BASE:
        print("warn: PREDICT_API_KEY 未设置，尝试无 key 请求（主网通常 401）", file=sys.stderr)
    try:
        tags = fetch_tags()
    except Exception as e:
        print(f"fetch tags failed: {e}", file=sys.stderr)
        return 1

    by_parent: dict[str | None, list[dict]] = {}
    for t in tags:
        by_parent.setdefault(t.get("parentId"), []).append(t)

    def by_name(parent_id: str | None, name: str) -> dict | None:
        for t in by_parent.get(parent_id, []):
            if (t.get("name") or "").strip() == name:
                return t
        return None

    sports_parent = "4"
    for t in tags:
        if (t.get("name") or "").strip() == "Sports" and t.get("parentId") in (None, ""):
            sports_parent = str(t["id"])
            break

    out: dict = {"api_base": API_BASE, "by_sub_category": {}}

    for tag_name, sub in L2_SPORTS.items():
        node = by_name(sports_parent, tag_name)
        if not node:
            print(f"missing L2 tag {tag_name!r} under Sports", file=sys.stderr)
            continue
        root_id = str(node["id"])
        if sub == "Football":
            out["by_sub_category"][sub] = {
                "sync_all_soccer": True,
                "soccer_root_tag_id": root_id,
                "market_variants": MATCH_TEAM_PROPS,
            }
            continue
        if sub == "Basketball":
            items = []
            for predict_name, sid, cn, en in BASKETBALL_LEAGUES:
                child = by_name(root_id, predict_name)
                if child:
                    items.append(
                        {
                            "name": cn,
                            "name_en": en,
                            "name_desc": en,
                            "sport_id": sid,
                            "tag_ids": [str(child["id"])],
                        }
                    )
            out["by_sub_category"][sub] = items_entry(items)
            continue
        if sub == "NFL":
            items = []
            for predict_name, sid, cn, en in NFL_LEAGUES:
                child = by_name(root_id, predict_name)
                if child:
                    items.append(
                        {
                            "name": cn,
                            "name_en": en,
                            "name_desc": en,
                            "sport_id": sid,
                            "tag_ids": [str(child["id"])],
                        }
                    )
            out["by_sub_category"][sub] = items_entry(items)
            continue
        out["by_sub_category"][sub] = sync_root_entry(root_id)

    for sub, (l2_name, l3_name) in SUB_LEAGUE_UNDER_L2.items():
        l2 = by_name(sports_parent, l2_name)
        if not l2:
            print(f"missing L2 {l2_name!r} for {sub}", file=sys.stderr)
            continue
        child = by_name(str(l2["id"]), l3_name)
        if child:
            out["by_sub_category"][sub] = items_entry(
                [
                    {
                        "name": "NHL",
                        "name_en": "NHL",
                        "name_desc": "NHL",
                        "sport_id": "nhl",
                        "tag_ids": [str(child["id"])],
                    }
                ]
            )
        else:
            out["by_sub_category"][sub] = sync_root_entry(str(l2["id"]))

    for sub, l2_name in SUB_SYNC_ROOT_L2.items():
        node = by_name(sports_parent, l2_name)
        if not node:
            print(f"missing L2 {l2_name!r} for {sub}", file=sys.stderr)
            continue
        out["by_sub_category"][sub] = sync_root_entry(str(node["id"]))

    OUT.write_text(json.dumps(out, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"wrote {OUT}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())