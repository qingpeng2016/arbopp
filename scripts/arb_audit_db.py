"""Read-only MySQL access and compact book parsing shared by the opportunity audits."""
from decimal import Decimal as D, InvalidOperation
import json
from pathlib import Path
import subprocess
import zlib

import pymysql

ZERO = D(0)
BINARY = "BINARY_YES_NO"
YES3 = "ONE_X2_YES_TRIPLE"
NO3 = "ONE_X2_NO_TRIPLE"
DEFAULT_FEES = {"POLYMARKET": D(".05"), "KALSHI": D(".07"), "PREDICT": D(".02"), "SXBET": D(".02")}


def connect(config):
    try:
        from yaml import safe_load
    except ImportError:
        # macOS system Ruby includes Psych; the repo's Python env may only include PyMySQL.
        program = 'require "yaml"; require "json"; print JSON.generate(YAML.safe_load(File.read(ARGV[0])).fetch("db"))'
        db = json.loads(subprocess.check_output(["ruby", "-e", program, str(config)]))
    else:
        db = safe_load(config.read_text())["db"]
    host, sep, port = str(db["host"]).rpartition(":")
    return pymysql.connect(host=host if sep else port, port=int(port) if sep else 3306,
                           user=db["user"], password=db["password"], database=db["db_name"],
                           charset="utf8mb4", connect_timeout=12, read_timeout=180,
                           cursorclass=pymysql.cursors.DictCursor)


def query(conn, sql, args=()):
    with conn.cursor() as cur:
        cur.execute(sql, args)
        return cur.fetchall()


def number(x):
    try:
        value = D(str(x))
        return value if value.is_finite() else None
    except (TypeError, ValueError, InvalidOperation):
        return None


def compact_leg(info_raw, book_raw):
    try:
        info = json.loads(info_raw or "{}")
        book = json.loads(zlib.decompress(book_raw[4:]) if book_raw else "{}",
                          parse_float=D)
        if not isinstance(info, dict) or not isinstance(book, dict):
            raise ValueError
        books = []
        for b in book.get("books") or []:
            # Full payloads are compressed on the wire. Only top-of-book evidence is retained.
            books.append(dict(role=b.get("role"), book_key=b.get("book_key"),
                              ask=(b.get("asks") or [None])[0], bid=(b.get("bids") or [None])[0]))
        return dict(info=info, platform=book.get("platform"), market_id=book.get("platform_market_outcome_id"),
                    books=books, quote_version=book.get("quote_version"), fee_rate=book.get("fee_rate"),
                    quote_at_ms=book.get("quote_at_ms"), net_cost=book.get("net_cost"), net_qty=book.get("net_qty"))
    except (TypeError, ValueError, AttributeError, zlib.error):
        return dict(info={}, books=[], parse_error=True)


def load_records(conn, max_id):
    cols = ["id", "opp_kind", "map_key", "category", "sub_category", "total_cost", "profit_space",
            "arb_qty", "arb_profit", "threshold_key", "threshold_value", "snapshot_at_ms"]
    for i in (1, 2, 3):
        cols += [f"leg_{i}_info", f"COMPRESS(leg_{i}_order_books) AS b{i}",
                 f"leg_{i}_platform", f"leg_{i}_best_ask"]
    records = []
    last_id = 0
    while last_id < max_id:
        batch = query(conn, "SELECT " + ",".join(cols) + " FROM arb_binary_opportunity "
                      "WHERE id>%s AND id<=%s ORDER BY id LIMIT 256", (last_id, max_id))
        if not batch:
            break
        for row in batch:
            row["legs"] = [compact_leg(row.pop(f"leg_{i}_info"), row.pop(f"b{i}")) for i in (1, 2, 3)]
            records.append(row)
        last_id = batch[-1]["id"]
    return records


def load_market_data(conn, rows):
    ids = set()
    for r in rows:
        for leg in r["legs"]:
            for k in ("yes_outcome_id", "no_outcome_id", "draw_outcome_id"):
                val = str(leg["info"].get(k, ""))
                if val.isdigit():
                    ids.add(int(val))
    metadata = {}
    ids = sorted(ids)
    for start in range(0, len(ids), 500):
        batch = ids[start:start+500]
        records = query(conn, "SELECT id,category,sub_category,platform,platform_market_outcome_id,"
                        "yes_token_id,no_token_id,market_row_id, map_key,outcome_type FROM market_outcome "
                        "WHERE id IN %s", (batch,))
        metadata.update({r["id"]: r for r in records})
    fee_rows = query(conn, "SELECT platform,category,sub_category,taker_fee_coef,updated_at "
                     "FROM config_strategy_fee WHERE status=1 ORDER BY id")
    return metadata, fee_rows


def fee_rate(fees, platform, category, sub):
    for c, s in ((category, sub), (category, ""), ("", "")):
        if (platform, c, s) in fees:
            return fees[platform, c, s]
    if platform in ("SXBET", "KALSHI"):
        return DEFAULT_FEES[platform]
    return D(".05") if category.lower() in ("sports", "") else ZERO


def leg_role(info):
    roles = [r for r in ("yes", "no", "draw") if info.get(f"{r}_outcome_id")]
    return roles[0] if len(roles) == 1 else None


def top(book, key):
    level = book.get(key)
    if isinstance(level, dict):
        return number(level.get("price")), number(level.get("size"))
    if isinstance(level, list) and len(level) >= 2:
        return number(level[0]), number(level[1])
    return None, None
