"""Export leg identities, settlement text and repeated-signal counts for a read-only audit."""
import argparse
import csv
import json
import re
import unicodedata
from collections import Counter
from pathlib import Path

from arb_audit_db import connect


def export(config, folder, ceiling):
    conn = connect(Path(config))
    with conn, conn.cursor() as cur:
        cur.execute("SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ")
        cur.execute("START TRANSACTION WITH CONSISTENT SNAPSHOT, READ ONLY")
        cur.execute("SELECT COALESCE(MAX(id),0) AS hi FROM arb_binary_opportunity")
        hi = int(cur.fetchone()["hi"])
        if ceiling is not None:
            hi = min(hi, ceiling)
        cur.execute("""SELECT id,opp_kind,map_key,snapshot_at_ms,total_cost,arb_qty,
                       leg_1_info,leg_2_info,leg_3_info,leg_1_platform,leg_2_platform,leg_3_platform
                       FROM arb_binary_opportunity WHERE id<=%s ORDER BY id""", (hi,))
        rows = cur.fetchall()
        claims, id_rows, outcome_ids = {}, [], set()
        for row in rows:
            legs = []
            for n in range(1, 4):
                raw = row[f"leg_{n}_info"]
                try:
                    info = json.loads(raw or "{}")
                except (ValueError, TypeError):
                    info = {"invalid_json": True}
                if not isinstance(info, dict):
                    info = {"invalid_json": True}
                if not info:
                    continue
                role = next((r for r in ("draw","yes","no") if f"{r}_outcome_id" in info), "unknown")
                value = str(info.get(f"{role}_outcome_id") or "")
                oid = int(value) if value.isdigit() else 0
                outcome_ids.add(oid)
                legs.append({"slot":n, "platform":row[f"leg_{n}_platform"], "oid":oid,
                             "outcome":info.get(f"{role}_outcome_name"), "fixture":info.get(f"{role}_market_name"),
                             "token":info.get(f"{role}_token_id"), "label":info.get(f"{role}_token_name")})
            key = (row["map_key"], row["opp_kind"], json.dumps(legs, sort_keys=True))
            claim = claims.setdefault(key, {"sample":row["id"], "map_key":row["map_key"],
                                            "kind":row["opp_kind"], "legs":legs, "count":0})
            claim["count"] += 1
            id_rows.append([row["id"], claim["sample"], row["total_cost"], row["arb_qty"], row["snapshot_at_ms"]])
        metadata = {}
        if outcome_ids:
            cur.execute("""SELECT o.id, o.platform, o.outcome_type, o.outcome_name, o.market_name,
                           o.config,o.`desc` AS rules,o.slug,o.yes_token_id,o.no_token_id,
                           m.event_start_time,m.map_key,m.name AS parent_name
                           FROM market_outcome o LEFT JOIN market m ON m.id=o.market_row_id
                           WHERE o.id IN %s""", (tuple(outcome_ids),))
            metadata = {r["id"]:r for r in cur.fetchall()}
        cur.execute("""SELECT o.status,COUNT(*) AS n FROM orders o
                       JOIN arb_binary_opportunity a ON a.id=o.arb_opp_id
                       WHERE a.id<=%s AND o.backtest_task_id=0 GROUP BY o.status""",(hi,))
        orders = cur.fetchall()
    folder.mkdir(exist_ok=True,parents=True)
    (folder/"contracts.json").write_text(json.dumps(dict(high_water=hi,orders=orders,
        claims=list(claims.values()),outcomes=metadata),ensure_ascii=False,default=str))
    with (folder/"contract_rows.csv").open("w",newline="") as f:
        writer=csv.writer(f)
        writer.writerow(["id","claim_sample","total_cost","arb_qty","snapshot_at_ms"])
        writer.writerows(id_rows)
    print(f"high_water={hi} rows={len(id_rows)} distinct_leg_sets={len(claims)} orders={orders}")


def team(name):
    text = unicodedata.normalize("NFKD", name.lower().replace("ø", "o"))
    text = "".join(c for c in text if not unicodedata.combining(c))
    # Only aliases observed and reviewed in the fixture metadata. Unmatched clubs go to review.
    aliases = {"1. fsv mainz 05":"fsv mainz", "angers sco":"angers", "sonderjyske fodbold":"sonderjyske",
               "bologna fc 1909":"bologna"}
    text = aliases.get(text.strip(), text.strip())
    words = re.findall(r"[a-z0-9]+", text)
    return " ".join(w for w in words if w not in {"club","de","fc","cf","calcio","cfc","ca","sc","us","sk"})


def fixture(name):
    return tuple(sorted(team(s) for s in re.split(r"\s+vs\.?\s+", name or "", flags=re.I)))


def proposition(title):
    win = re.fullmatch(r"Will (.+) win on [\d-]+\?", title or "")
    if win:
        return team(win[1])
    sx = re.fullmatch(r"(.+) vs Not (.+) \[1X2\]", title or "")
    if sx and sx[1] == sx[2]:
        return team(sx[1])
    if re.fullmatch(r"Will .+ end in a draw\?", title or ""):
        return "draw"
    return None


def classify(claim, outcomes, native):
    legs = claim["legs"]
    meta = [outcomes.get(str(l["oid"]), {}) for l in legs]
    if any(o.get("platform") != l["platform"] or o.get("map_key") != claim["map_key"] or
           o.get("outcome_type") != "moneyline" for l, o in zip(legs, meta)):
        return "metadata_review"
    fixtures = {fixture(l["fixture"]) for l in legs}
    if len(fixtures) != 1 or len(set(next(iter(fixtures)))) != 2:
        return "fixture_review"
    participants = set(next(iter(fixtures)))
    sides = []
    for l, o in zip(legs, meta):
        token = l["token"]
        if not token:
            return "token_review"
        if token == o["yes_token_id"] and token != o["no_token_id"]:
            sides.append("yes")
        elif token == o["no_token_id"] and token != o["yes_token_id"]:
            sides.append("no")
        elif l["platform"] == "SXBET" and o.get("yes_token_id") and token == o["yes_token_id"] + ":no":
            sides.append("no")
        else:
            return "token_review"
    props = [proposition(l["outcome"]) for l in legs]
    if claim["kind"] == "BINARY_YES_NO":
        if sorted(sides) != ["no","yes"]:
            return "direction_review"
        if any(p is None for p in props):
            return {"complementary":"native_complementary","same_team":"same_team"}.get(native, "named_team_review")
        if len(set(props)) != 1 or props[0] not in participants | {"draw"}:
            return "proposition_review"
        if any(l["platform"] == "SXBET" for l in legs):
            return "normal_play_coverage_sx_void_basis"
        if meta[0].get("rules") and meta[0]["rules"] == meta[1].get("rules"):
            return "binary_matching_rule_excerpt"
        return "rule_review"
    side = {"ONE_X2_YES_TRIPLE":"yes","ONE_X2_NO_TRIPLE":"no"}.get(claim["kind"])
    if len(legs) == 3 and set(sides) == {side} and set(props) == participants | {"draw"}:
        return "three_distinct_normal_play_outcomes"
    return "triple_review"


def write_verdicts(folder):
    audit = json.loads((folder/"contracts.json").read_text())
    native_path = folder/"native_labels.json"
    native = {tuple(c.get("tokens", [])): c["status"] for c in json.loads(native_path.read_text())["claims"]} if native_path.exists() else {}
    claims = {c["sample"]:c for c in audit["claims"]}
    verdicts = {k: classify(c, audit["outcomes"], native.get(tuple(l["token"] for l in c["legs"])))
                for k, c in claims.items()}
    counts = Counter()
    signatures = {}
    one_platform = start_conflicts = 0
    with (folder/"contract_rows.csv").open(newline="") as src, (folder/"semantic.csv").open("w",newline="") as dst:
        writer = csv.writer(dst)
        writer.writerow(["id","claim_sample","coverage","one_platform","start_time_conflict"])
        for r in csv.DictReader(src):
            key = int(r["claim_sample"])
            legs = claims[key]["legs"]
            starts = {audit["outcomes"].get(str(l["oid"]),{}).get("event_start_time") for l in legs}
            single = len({l["platform"] for l in legs}) == 1
            writer.writerow([r["id"], key, verdicts[key], single, len(starts) > 1])
            counts[verdicts[key]] += 1
            start_conflicts += len(starts) > 1
            one_platform += single
            tokens = tuple(sorted((l["platform"] or "", str(l["token"] or "")) for l in legs))
            signature = (claims[key]["map_key"], claims[key]["kind"], tokens, r["total_cost"], r["arb_qty"])
            signatures.setdefault(signature, []).append(int(r["snapshot_at_ms"]))
    gaps = []
    for stamps in signatures.values():
        stamps.sort()
        gaps.extend(b - a for a, b in zip(stamps, stamps[1:]))
    summary = dict(high_water=audit["high_water"], coverage=counts, unique_signatures=len(signatures),
                   repeats=len(gaps), repeat_within_1s=sum(g <= 1000 for g in gaps),
                   repeat_within_5s=sum(g <= 5000 for g in gaps), single_venue=one_platform,
                   start_conflicts=start_conflicts, matched_orders_by_status=audit["orders"])
    (folder / "contract_summary.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2))
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", default="conf/local.conf.yaml")
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--max-id", type=int)
    parser.add_argument("--classify-only", action="store_true")
    args = parser.parse_args()
    if not args.classify_only:
        export(args.config, args.output, args.max_id)
    write_verdicts(args.output)
