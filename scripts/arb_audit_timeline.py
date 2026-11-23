"""Compare each purchased token with adjacent WS observations. Later quotes are diagnostic only."""
from bisect import bisect_right
from collections import defaultdict
from decimal import Decimal
import json

from arb_audit_db import query


def ask(levels):
    try:
        levels = json.loads(levels, parse_float=Decimal)
        for level in levels:
            price, size = map(lambda value: Decimal(str(value)), level[:2])
            if price.is_finite() and size.is_finite() and 0 < price < 1 and size > 0:
                return price, size
    except (ValueError, TypeError, ArithmeticError):
        pass
    return None, None


def timeline(conn, groups, window_ms=5000, complement_bid=False):
    """Group by (platform, token, platform outcome ID, map key). Entries are (ms, id, slot)."""
    intervals = []
    for key, events in groups.items():
        spans = []
        for stamp in sorted({event[0] for event in events}):
            lo, hi = stamp - window_ms, stamp + window_ms
            if spans and lo <= spans[-1][1]:
                spans[-1][1] = hi
            else:
                spans.append([lo, hi])
        intervals.extend((key, lo, hi) for lo, hi in spans)

    archive = defaultdict(list)
    for start in range(0, len(intervals), 24):
        batch = intervals[start:start + 24]
        clauses, args = [], []
        for index, ((platform, token, market, fixture), lo, hi) in enumerate(batch):
            column = "order_book_bids" if complement_bid else "order_book_asks"
            clauses.append(
                f"SELECT %s AS bucket,id,snapshot_at_ms,{column} AS levels FROM market_order_book_ws "
                "WHERE platform=%s AND token_id=%s AND snapshot_at_ms BETWEEN %s AND %s "
                "AND platform_market_outcome_id=%s AND map_key=%s"
            )
            args.extend((index, platform, token, lo, hi, market, fixture))
        for row in query(conn, " UNION ALL ".join(clauses), args):
            key = batch[row["bucket"]][0]
            price, size = ask(row["levels"])
            if complement_bid and price is not None:
                price = 1 - price
            archive[key].append((row["snapshot_at_ms"], row["id"], price, size))

    result = {}
    for key, events in groups.items():
        observations = sorted(archive[key], key=lambda o: (o[0], o[1]))
        times = [o[0] for o in observations]
        for stamp, record_id, slot in events:
            right = bisect_right(times, stamp)
            adjacent = []
            for index in (right - 1, right):
                if 0 <= index < len(times) and abs(times[index] - stamp) <= window_ms:
                    time_ms, ws_id, price, size = observations[index]
                    adjacent.append((ws_id, time_ms - stamp, price, size))
                else:
                    adjacent.append((None,) * 4)
            result[record_id, slot] = adjacent
    return result
