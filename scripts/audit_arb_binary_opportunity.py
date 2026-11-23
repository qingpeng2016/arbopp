"""Export an opportunity-by-opportunity audit, including archived quotes before and after the signal."""
import argparse
import csv
import json
from collections import Counter, defaultdict
from decimal import Decimal as D
from pathlib import Path

from arb_audit_timeline import timeline
from arb_audit_db import (
    BINARY, NO3, YES3, connect, fee_rate, leg_role, load_market_data, load_records, number, query, top,
)

TOLERANCE = D("0.00000001")
ONE = D(1)


def venue_cost(platform, price):
    # Scenario for standard takers with no discount. Normalize each leg to one unit of net payout.
    if platform == "PREDICT":
        return price / (1 - D(".02") * min(price, 1 - price) / price)
    if platform == "SXBET":
        return price / (1 - D(".01") * (1 - price))
    if platform == "POLYMARKET":
        return price + D(".05") * price * (1 - price)
    return None


def net_quote(platform, price, size, rate):
    if platform == "PREDICT":
        ratio = 1 - rate * min(price, 1 - price) / price
        return price / ratio, size * ratio
    if platform == "SXBET":
        ratio = 1 - rate * (1 - price)
        return price / ratio, size / price * ratio
    return price + rate * price * (1 - price), size


def price_verdict(cost, threshold, payout):
    if cost < threshold:
        return "passes_threshold"
    return "positive_below_required_margin" if cost < payout else "no_positive_margin"


def inspect_record(record, metadata, rates, quotes, order_amount, synthetic):
    kind = record["opp_kind"]
    payout = D(2) if kind == NO3 else ONE
    count = 2 if kind == BINARY else 3
    result = {key: record[key] for key in ("id", "map_key", "opp_kind", "snapshot_at_ms", "threshold_value",
                                           "total_cost", "arb_qty", "profit_space", "arb_profit")}
    flags = set()
    threshold_keys = {BINARY: "binary_yes_no_max_cost", YES3: "1x2_yes_triple_max_cost", NO3: "1x2_no_triple_max_cost"}
    if record["threshold_key"] != threshold_keys.get(kind):
        flags.add("threshold_key")
    extended_costs, extended_sizes = [], []
    prior_prices, prior_sizes, book_sizes, book_costs, picked_costs, actual_costs = [], [], [], [], [], []
    prior_match, post_match = [], []
    leg_rows = []
    if kind not in (NO3, YES3, BINARY):
        flags.add("unknown_kind")
    for slot, leg in enumerate(record["legs"], 1):
        scalar = record[f"leg_{slot}_best_ask"]
        platform = record[f"leg_{slot}_platform"]
        if not leg.get("info") and scalar is None and not leg.get("books") and not platform and not leg.get("parse_error"):
            continue
        role = leg_role(leg["info"])
        info = leg["info"]
        oid = str(info.get(f"{role}_outcome_id", ""))
        meta = metadata.get(oid, {})
        token = info.get(f"{role}_token_id")
        if leg.get("parse_error") or not role or not token or not meta:
            flags.add("missing_identity")
        if meta.get("platform") != platform or meta.get("map_key") != record["map_key"] or platform != leg.get("platform"):
            flags.add("identity_conflict")
        buy_yes = kind == YES3 or (kind == BINARY and role == "yes")
        selected = [b for b in leg["books"] if b.get("role") in
                    (("YES_BOOK", "YES_TOKEN") if buy_yes else ("NO_BOOK", "NO_TOKEN"))]
        price, size = top(selected[0], "ask") if len(selected) == 1 else (None, None)
        rate = fee_rate(rates, platform or "", meta.get("category") or "", meta.get("sub_category") or "")
        revised = leg.get("quote_version") == 2
        if revised:
            stored_rate = number(leg.get("fee_rate"))
            if stored_rate is not None and 0 <= stored_rate < 1:
                rate = stored_rate
            else:
                flags.add("quote_fee_rate")
        prior, post = quotes.get((record["id"], slot), ((None,) * 4,) * 2)
        derived = synthetic.get((record["id"], slot), ((None,) * 4,) * 2)[0]
        available = prior if prior[2] is not None else derived
        if available[2] is not None:
            px, depth = available[2], available[3]
            cost, qty = net_quote(platform, px, depth, rate) if revised else (px + rate * px * (1 - px), depth)
            extended_costs.append(cost)
            extended_sizes.append(qty)
        leg_rows.append(dict(id=record["id"], slot=slot, platform=platform, outcome_id=oid, token=token,
                             best_ask_column=scalar, book_price=price, book_size=size, ws_id=prior[0],
                             quote_version=leg.get("quote_version"),
                             ws_delta_ms=prior[1], ws_price=prior[2], ws_size=prior[3],
                             next_ws_id=post[0], next_delta_ms=post[1], next_price=post[2], next_size=post[3],
                             derived_ws_id=derived[0], derived_delta_ms=derived[1],
                             derived_no_ask=derived[2], derived_size=derived[3]))
        if not (price is not None and 0 < price < 1 and size and size > 0 and scalar and 0 < scalar < 1 + rate):
            flags.add("invalid_book_or_scalar")
            continue
        if revised:
            net_cost, net_qty = net_quote(platform, price, size, rate)
            at = number(leg.get("quote_at_ms"))
            if at is None or not 0 <= record["snapshot_at_ms"] - at <= 5000:
                flags.add("quote_age")
            for stored, value in ((leg.get("net_cost"), net_cost), (leg.get("net_qty"), net_qty)):
                stored = number(stored)
                if stored is None or abs(stored - value) > max(TOLERANCE, abs(value) * D("1e-14")):
                    flags.add("quote_net_fields")
        else:
            net_cost, net_qty = price + rate * price * (1 - price), size
        book_sizes.append(net_qty)
        book_costs.append(net_cost)
        expected_scalar = price if revised or kind == BINARY else book_costs[-1]
        if abs(expected_scalar - scalar) > TOLERANCE:
            flags.add("book_price")
        # Triple scalars already include the configured fee. Invert p + r*p*(1-p) before scenario pricing.
        picked = scalar if revised or kind == BINARY else 2 * scalar / (1 + rate + ((1 + rate)**2 - 4 * rate * scalar).sqrt())
        picked_costs.append(net_quote(platform, picked, size, rate)[0] if revised else picked + rate * picked * (1 - picked))
        scenario = venue_cost(platform, picked)
        if scenario is not None:
            actual_costs.append(scenario)
        if prior[2] is not None:
            px, depth = prior[2], prior[3]
            cost, qty = net_quote(platform, px, depth, rate) if revised else (px + rate * px * (1 - px), depth)
            prior_prices.append(cost)
            prior_sizes.append(qty)
        prior_match.append(prior[2] is not None and abs(prior[2] - price) <= TOLERANCE)
        post_match.append(post[2] is not None and abs(post[2] - price) <= TOLERANCE)
    if len(leg_rows) != count or len(book_costs) != count:
        flags.add("leg_count_or_unreadable")
    if abs(record["profit_space"] - (payout - record["total_cost"])) > TOLERANCE:
        flags.add("profit_space")
    qty = record["arb_qty"]
    if abs(record["arb_profit"] - qty * record["profit_space"]) > (abs(qty) + abs(record["profit_space"]) + 1) * TOLERANCE / 2:
        flags.add("arb_profit")
    if record["total_cost"] >= record["threshold_value"] or qty <= 0:
        flags.add("stored_threshold_or_qty")
    if len(book_costs) == count:
        result.update(book_cost=sum(book_costs), book_qty=min(book_sizes))
        if abs(sum(book_costs) - record["total_cost"]) > count * TOLERANCE:
            flags.add("book_cost")
        if abs(min(book_sizes) - qty) > TOLERANCE:
            flags.add("book_qty")
        if abs(sum(picked_costs) - record["total_cost"]) > count * TOLERANCE:
            flags.add("scalar_cost")
        divisor = sum(picked_costs) if any(leg.get("quote_version") == 2 for leg in record["legs"]) else sum(
            record[f"leg_{leg['slot']}_best_ask"] for leg in leg_rows)
        if divisor > 0:
            result["configured_target_qty"] = order_amount / divisor
    if len(actual_costs) == count:
        result["standard_fee_scenario_cost"] = sum(actual_costs)
    if len(prior_prices) == count and len(leg_rows) == count:
        ws_cost = sum(prior_prices)
        result.update(ws_cost=ws_cost, ws_qty=min(prior_sizes), ws_depth_covers_qty=min(prior_sizes) >= qty - TOLERANCE)
        result["ws_result"] = price_verdict(ws_cost, record["threshold_value"], payout)
    else:
        result["ws_result"] = "insufficient_quotes"
    if len(extended_costs) == count and len(leg_rows) == count:
        extended = sum(extended_costs)
        result.update(ws_including_derived_cost=extended, ws_including_derived_qty=min(extended_sizes))
        result["ws_including_derived_result"] = price_verdict(extended, record["threshold_value"], payout)
    else:
        result["ws_including_derived_result"] = "insufficient_quotes"
    result.update(ws_all_prices_match=len(prior_match) == count and all(prior_match),
                  later_all_prices_match=len(post_match) == count and all(post_match),
                  record_issues="|".join(sorted(flags)))
    return result, leg_rows


def write_csv(path, rows):
    fields = dict.fromkeys(key for row in rows for key in row)
    with path.open("w", newline="", encoding="utf-8") as out:
        writer = csv.DictWriter(out, fieldnames=fields, lineterminator="\n")
        writer.writeheader()
        writer.writerows(rows)


def run(config, output, max_id, window_ms):
    with connect(config) as conn:
        query(conn, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ")
        query(conn, "START TRANSACTION WITH CONSISTENT SNAPSHOT, READ ONLY")
        high = int(query(conn, "SELECT COALESCE(MAX(id),0) AS n FROM arb_binary_opportunity")[0]["n"])
        records = load_records(conn, high if max_id is None else min(high, max_id))
        md, fee_rows = load_market_data(conn, records)
        params = {r["config_key"]: r["config_value"] for r in query(conn,
                  "SELECT config_key,config_value FROM config_strategy_threshold WHERE status=1 "
                  "AND config_key IN ('binary_yes_no_order_amount','binary_yes_no_level_factor')")}
        order_amount = params.get("binary_yes_no_order_amount", D(100))
        if order_amount <= 0:
            order_amount = D(100)
        metadata = {str(k): v for k, v in md.items()}
        rates = {(r["platform"], r["category"] or "", r["sub_category"] or ""): number(r["taker_fee_coef"])
                 for r in fee_rows}
        groups = defaultdict(list)
        derived_groups = defaultdict(list)
        for r in records:
            for slot, leg in enumerate(r["legs"], 1):
                info = leg["info"]
                role = leg_role(info)
                meta = metadata.get(str(info.get(f"{role}_outcome_id")), {})
                token = info.get(f"{role}_token_id")
                if meta and token and r[f"leg_{slot}_platform"] == meta["platform"] and meta["map_key"] == r["map_key"]:
                    key = (meta["platform"], token, meta["platform_market_outcome_id"], r["map_key"])
                    entry = (r["snapshot_at_ms"], r["id"], slot)
                    groups[key].append(entry)
                    if (meta["platform"] == "SXBET" and not meta.get("no_token_id") and
                            token == meta["platform_market_outcome_id"] + ":no" and meta.get("yes_token_id")):
                        base = (meta["platform"], meta["yes_token_id"], meta["platform_market_outcome_id"], r["map_key"])
                        derived_groups[base].append(entry)
        quotes = timeline(conn, groups, window_ms)
        synthetic = timeline(conn, derived_groups, window_ms, complement_bid=True)
    results, legs = [], []
    for record in records:
        result, evidence = inspect_record(record, metadata, rates, quotes, order_amount, synthetic)
        results.append(result)
        legs.extend(evidence)
    output.mkdir(exist_ok=True, parents=True)
    write_csv(output / "evidence.csv", results)
    write_csv(output / "leg_evidence.csv", legs)
    summary = dict(records=len(results), legs=len(legs), window_ms=window_ms,
                   max_id=records[-1]["id"] if records else None, current_params=params,
                   record_issues=Counter(row["record_issues"] or "consistent" for row in results),
                   direct_ws=Counter(row["ws_result"] for row in results),
                   including_derived_ws=Counter(row["ws_including_derived_result"] for row in results))
    (output / "evidence_summary.json").write_text(json.dumps(summary, indent=2, default=str) + "\n")
    print(f"{len(results)} records, {len(legs)} legs, window {window_ms} ms, max ID {records[-1]['id'] if records else 0}")
    print(f"Current order_amount {order_amount}, level_factor {params.get('binary_yes_no_level_factor', 2)}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", type=Path, default=Path("conf/local.conf.yaml"))
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--max-id", type=int)
    parser.add_argument("--window-ms", type=int, default=5000)
    args = parser.parse_args()
    if args.window_ms < 0:
        parser.error("--window-ms must be nonnegative")
    run(args.config, args.output, args.max_id, args.window_ms)
