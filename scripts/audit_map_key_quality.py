#!/usr/bin/env python3
"""基于 market 表审计 map_key：漏映射 / 错映射 / 可多映射平台。"""
import re
import sys
from collections import defaultdict
from datetime import datetime

import pymysql

HOST, USER, PW, DB = "47.84.121.116", "mm", "2kHjniEMBGh7rRWz", "fgmm_strategy_less_one"
MAX_START_H = 18
STOP = {"the", "and", "club", "fc", "sc", "cf", "ac", "afc", "de", "da", "do", "of"}

RE_VS = re.compile(r"\s+vs\.?\s+", re.I)


def fold_name(s: str) -> str:
    out = []
    for c in s.strip().lower():
        if c.isalnum() or c in " -":
            out.append(c if c != "-" else " ")
    return "".join(out)


def fixture_sides(name: str):
    parts = RE_VS.split(fold_name(name))
    if len(parts) != 2:
        return None
    def tokens(side):
        t = []
        seen = set()
        for w in side.split():
            w = w.strip()
            if len(w) < 3 or w in STOP or w in seen:
                continue
            seen.add(w)
            t.append(w)
        return t
    a, b = tokens(parts[0]), tokens(parts[1])
    if not a or not b:
        return None
    return a, b


def names_match(na: str, nb: str) -> bool:
    pa, pb = fixture_sides(na), fixture_sides(nb)
    if not pa or not pb:
        return False
    k1, k2, p1, p2 = pa[0], pa[1], pb[0], pb[1]

    def overlap(x, y):
        return bool(set(x) & set(y))

    return (overlap(k1, p1) and overlap(k2, p2)) or (overlap(k1, p2) and overlap(k2, p1))


def parse_start(raw: str):
    raw = (raw or "").strip()
    if not raw:
        return None
    for fmt in ("%Y-%m-%d %H:%M:%S", "%Y-%m-%dT%H:%M:%SZ", "%Y-%m-%dT%H:%M:%S%z"):
        try:
            return datetime.strptime(raw[:19].replace("T", " ")[:19], "%Y-%m-%d %H:%M:%S")
        except ValueError:
            pass
    if len(raw) >= 10:
        try:
            return datetime.strptime(raw[:10], "%Y-%m-%d")
        except ValueError:
            pass
    return None


def start_ok(sa: str, sb: str) -> bool:
    ta, tb = parse_start(sa), parse_start(sb)
    if not ta or not tb:
        return True
    if ta.date() != tb.date():
        return False
    return abs((ta - tb).total_seconds()) <= MAX_START_H * 3600


def scope(row):
    return "|".join([
        (row["category"] or "").strip(),
        (row["sub_category"] or "").strip(),
        (row["competition_en"] or "").strip(),
    ])


def load_rows(conn):
    cur = conn.cursor(pymysql.cursors.DictCursor)
    cur.execute("""
        SELECT id, platform, name, map_key, map_verified, event_start_time,
               category, sub_category, competition_en, slug
        FROM market WHERE status='ACTIVE'
    """)
    return cur.fetchall()


def main():
    conn = pymysql.connect(host=HOST, user=USER, password=PW, database=DB, charset="utf8mb4")
    rows = load_rows(conn)
    conn.close()

    by_scope = defaultdict(list)
    for r in rows:
        if scope(r):
            by_scope[scope(r)].append(r)

  # --- 漏映射：同组条件应对上 map_key，但没有 ---
    missing_third = []  # 已有2+平台同key，第三平台有同窗孤行
    unlinked_pairs = []

    for sc, listr in by_scope.items():
        for i in range(len(listr)):
            for j in range(i + 1, len(listr)):
                a, b = listr[i], listr[j]
                if a["platform"].upper() == b["platform"].upper():
                    continue
                na, nb = (a["name"] or ""), (b["name"] or "")
                if not names_match(na, nb) or not start_ok(a["event_start_time"], b["event_start_time"]):
                    continue
                ka = (a["map_key"] or "").strip()
                kb = (b["map_key"] or "").strip()
                if ka and kb and ka == kb:
                    continue
                unlinked_pairs.append((sc, a, b))

    # 组内缺平台：按 map_key 聚合，找同窗应匹配但未进组的平台
    by_key = defaultdict(list)
    for r in rows:
        k = (r["map_key"] or "").strip()
        if k:
            by_key[k].append(r)

    for mk, members in by_key.items():
        plats = {m["platform"].upper() for m in members}
        if len(plats) < 2:
            continue
        rep = members[0]
        sc = scope(rep)
        day = (rep["event_start_time"] or "")[:10]
        ref_name = rep["name"] or ""
        for r in by_scope.get(sc, []):
            p = r["platform"].upper()
            if p in plats:
                continue
            if (r["event_start_time"] or "")[:10] != day:
                continue
            rk = (r["map_key"] or "").strip()
            if rk == mk:
                continue
            if not names_match(ref_name, r["name"] or ""):
                continue
            if not start_ok(rep["event_start_time"], r["event_start_time"]):
                continue
            missing_third.append({
                "map_key": mk,
                "has_plats": sorted(plats),
                "missing": p,
                "orphan_id": r["id"],
                "orphan_key": rk or None,
                "orphan_name": (r["name"] or "")[:80],
                "league": rep["competition_en"],
            })

    # --- 错映射 ---
    wrong_in_group = []
    for mk, members in by_key.items():
        for i in range(len(members)):
            for j in range(i + 1, len(members)):
                a, b = members[i], members[j]
                if a["platform"].upper() == b["platform"].upper():
                    continue
                reasons = []
                if not names_match(a["name"] or "", b["name"] or ""):
                    reasons.append("队名不匹配")
                if not start_ok(a["event_start_time"], b["event_start_time"]):
                    reasons.append("开赛不一致")
                if scope(a) != scope(b):
                    reasons.append("联赛桶不一致")
                if reasons:
                    wrong_in_group.append((mk, reasons, a, b))

    dup_same_plat = []
    for mk, members in by_key.items():
        by_p = defaultdict(list)
        for m in members:
            by_p[m["platform"].upper()].append(m["id"])
        for p, ids in by_p.items():
            if len(ids) > 1:
                dup_same_plat.append((mk, p, ids))

    # 输出
    print(f"ACTIVE rows: {len(rows)}")
    print(f"\n## 1) 漏映射（应对上但 map_key 不同/一方无）: {len(unlinked_pairs)} 对")
    for i, (sc, a, b) in enumerate(unlinked_pairs[:25]):
        print(f"  [{i+1}] {a['competition_en']} | {a['platform']} id={a['id']} key={repr((a['map_key'] or '')[:60])}")
        print(f"       {b['platform']} id={b['id']} key={repr((b['map_key'] or '')[:60])}")
        print(f"       A: {(a['name'] or '')[:70]}")
        print(f"       B: {(b['name'] or '')[:70]}")

    print(f"\n## 2) 已有跨平台 map_key 但少映了第三平台: {len(missing_third)} 条")
    by_pattern = defaultdict(int)
    for x in missing_third:
        by_pattern[(tuple(x["has_plats"]), x["missing"])] += 1
    for (has, miss), cnt in sorted(by_pattern.items(), key=lambda x: -x[1])[:15]:
        print(f"  {cnt}x 组内已有 {has} 缺 {miss}")
    for x in missing_third[:20]:
        print(f"  map_key={x['map_key'][:55]}... +{x['missing']} id={x['orphan_id']} orphan_key={x['orphan_key']}")
        print(f"    {x['orphan_name']}")

    print(f"\n## 3) 错映射（同 map_key 内队名/开赛对不上）: {len(wrong_in_group)} 对")
    for i, (mk, reasons, a, b) in enumerate(wrong_in_group[:20]):
        print(f"  [{i+1}] {','.join(reasons)} | {mk[:70]}")
        print(f"    {a['platform']}: {(a['name'] or '')[:65]}")
        print(f"    {b['platform']}: {(b['name'] or '')[:65]}")

    print(f"\n## 4) 多映（同 map_key 同平台重复行）: {len(dup_same_plat)} 组")
    for mk, p, ids in dup_same_plat[:15]:
        print(f"  {p} dup ids={ids} key={mk[:60]}")

    # 分布：2平台组本可多3平台
    two_plat = sum(1 for mk, m in by_key.items() if len({x['platform'].upper() for x in m}) == 2)
    three_plat = sum(1 for mk, m in by_key.items() if len({x['platform'].upper() for x in m}) == 3)
    print(f"\n## 5) 现有 map_key 平台数分布: 2平台={two_plat} 3平台={three_plat} (其余见 compliant 统计)")


if __name__ == "__main__":
    main()