#!/usr/bin/env python3
"""Build the two historical reports from local GoClaw snapshots."""
from __future__ import annotations

import csv
import datetime as dt
import gzip
import json
import re
from collections import Counter, defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ANALYTICS = ROOT / "data/analytics"
OUT = ANALYTICS / "analytics.json"
CRON_FILE = ANALYTICS / "morning-cron-runs.json"
ORDERS_FILE = ANALYTICS / "orders-frequency-report-current-year.csv"
RAW_ORDERS_DIR = ANALYTICS / "raw-orders"
RAW_INVOICES_DIR = ANALYTICS / "raw-invoices"
PRODUCTS_FILE = ANALYTICS / "products-latest.jsonl.gz"
EXCLUDED_SCHEDULE_DATES = {"2026-09-25"}


def load_json(path: Path, fallback):
    try:
        raw = path.read_bytes()
        encoding = "utf-16" if raw.startswith((b"\xff\xfe", b"\xfe\xff")) else "utf-8-sig"
        return json.loads(raw.decode(encoding))
    except (OSError, ValueError, UnicodeError):
        return fallback


def report_date(text: str) -> str | None:
    match = re.search(r"(\d{2}/\d{2}/\d{4})", text or "")
    if not match:
        return None
    day, month, year = match.group(1).split("/")
    return f"{year}-{month}-{day}"


def cron_report():
    rows = load_json(CRON_FILE, [])
    by_date = {}
    invalid = 0
    marker = "📍 Chi nhánh chưa đặt:"
    for row in rows:
        summary = row.get("summary") or ""
        if row.get("status") != "ok" or marker not in summary:
            invalid += 1
            continue
        day = report_date(summary)
        if not day:
            invalid += 1
            continue
        if day in EXCLUDED_SCHEDULE_DATES:
            invalid += 1
            continue
        current = by_date.get(day)
        if current is None or str(row.get("ran_at", "")) > str(current.get("ran_at", "")):
            by_date[day] = row

    missing = Counter()
    daily = []
    for day, row in sorted(by_date.items()):
        text = row.get("summary") or ""
        tail = text[text.find(marker) + len(marker):]
        next_heading = re.search(r"\n\s*\n[^\n]*\n", tail)
        section = tail[: next_heading.start()] if next_heading else tail
        warehouses, total = {}, 0
        for line in section.splitlines():
            line = line.strip().lstrip("•-").strip()
            if not line or ":" not in line:
                continue
            warehouse, names = line.split(":", 1)
            names = names.strip()
            branches = [] if names.lower() in {"không có", "khong co", "-"} else [x.strip() for x in names.split(",") if x.strip()]
            warehouses[warehouse.strip()] = branches
            total += len(branches)
            missing.update(branches)
        daily.append({"date": day, "branches": total, "warehouses": warehouses, "ranAt": row.get("ran_at")})
    return {"source": "GoClaw cron_run_logs / bao-cao-dong-hang-sang", "firstReliableDate": min(by_date) if by_date else None, "lastDate": max(by_date) if by_date else None, "validDays": len(by_date), "ignoredRuns": invalid, "excludedDates": sorted(EXCLUDED_SCHEDULE_DATES), "branchRanking": [{"branch": n, "days": c} for n, c in missing.most_common()], "daily": daily}


def parse_date(value: str) -> dt.date | None:
    try:
        return dt.datetime.fromisoformat(str(value).replace("Z", "+00:00")).date()
    except (ValueError, TypeError):
        return None


def order_records():
    seen_ids = set()
    raw_files = sorted(RAW_ORDERS_DIR.rglob("orders-*.jsonl.gz")) if RAW_ORDERS_DIR.exists() else []
    raw_files = [p for p in raw_files if not p.name.startswith("orders-week-")]
    if raw_files:
        for path in raw_files:
            try:
                with gzip.open(path, "rt", encoding="utf-8") as handle:
                    for line in handle:
                        try:
                            item = json.loads(line).get("data", {})
                        except json.JSONDecodeError:
                            continue
                        order_id = item.get("id")
                        if order_id in seen_ids:
                            continue
                        seen_ids.add(order_id)
                        yield item
            except OSError:
                continue
        return
    if ORDERS_FILE.exists():
        with ORDERS_FILE.open("r", encoding="utf-8-sig", newline="") as handle:
            yield from csv.DictReader(handle)


def invoice_records():
    seen_ids = set()
    for path in sorted(RAW_INVOICES_DIR.rglob("invoices-*.jsonl.gz")):
        try:
            with gzip.open(path, "rt", encoding="utf-8") as handle:
                for line in handle:
                    try:
                        item = json.loads(line).get("data", {})
                    except json.JSONDecodeError:
                        continue
                    invoice_id = item.get("id")
                    if invoice_id in seen_ids:
                        continue
                    seen_ids.add(invoice_id)
                    date = parse_date(item.get("purchaseDate", ""))
                    code = str(item.get("code") or "").upper()
                    status = str(item.get("statusValue") or "").strip().upper()
                    if date and date.year == dt.date.today().year and status == "HOÀN THÀNH" and (code.startswith("HD") or code.startswith("HĐ")):
                        yield item
        except OSError:
            continue


def product_aliases():
    """Map retired product codes to a sole active replacement with the same name."""
    groups = defaultdict(list)
    if not PRODUCTS_FILE.exists():
        return {}, [], set()
    try:
        with gzip.open(PRODUCTS_FILE, "rt", encoding="utf-8") as handle:
            for line in handle:
                try:
                    item = json.loads(line).get("data", {})
                except json.JSONDecodeError:
                    continue
                name = str(item.get("name") or "").strip().casefold()
                code = str(item.get("code") or "").strip()
                if name and code:
                    groups[name].append(item)
    except OSError:
        return {}, [], set()
    aliases, evidence, ignored = {}, [], set()
    for items in groups.values():
        active = [item for item in items if item.get("isActive") is True]
        retired = [item for item in items if item.get("isActive") is False]
        retired = [item for item in retired if not str(item.get("code") or "").strip().endswith("-")]
        if len(active) == 1 and retired:
            replacement = active[0]
            canonical = f"{replacement.get('code')} · {replacement.get('name')}"
            for item in items:
                aliases[str(item.get("code"))] = canonical
            evidence.append({"canonical": canonical, "retiredCodes": [item.get("code") for item in retired], "activeCode": replacement.get("code"), "name": replacement.get("name")})
        elif not active and retired:
            ignored.update(str(item.get("code")) for item in retired)
    return aliases, evidence, ignored


def product_report():
    using_invoices = RAW_INVOICES_DIR.exists()
    if not using_invoices and not ORDERS_FILE.exists() and not RAW_ORDERS_DIR.exists():
        return {"available": False, "reason": "Chưa có dữ liệu đơn hàng"}
    rows, latest = 0, None
    customer_products = defaultdict(Counter)
    latest_orders = {}
    code_aliases, alias_evidence, ignored_codes = product_aliases()
    orders = list(invoice_records() if using_invoices else order_records())
    customer_names = defaultdict(Counter)
    customer_latest = {}
    for row in orders:
        customer_id = row.get("customerId")
        customer_name = (row.get("customerName") or row.get("customerCode") or "").strip()
        day = parse_date(row.get("purchaseDate", ""))
        if customer_id and customer_name:
            customer_names[str(customer_id)][customer_name] += 1
            if day and (str(customer_id) not in customer_latest or day >= customer_latest[str(customer_id)][0]):
                customer_latest[str(customer_id)] = (day, customer_name)
    aliases = [{"customerId": cid, "canonical": customer_latest[cid][1], "aliases": sorted(counts), "orders": sum(counts.values())} for cid, counts in customer_names.items() if len(counts) > 1]
    canonical_by_id = {cid: value[1] for cid, value in customer_latest.items()}
    for row in orders:
        day = parse_date(row.get("purchaseDate", ""))
        if not day:
            continue
        latest = max(latest, day) if latest else day
        # Đây là khách hàng/chi nhánh đặt hàng, không phải kho xử lý đơn.
        customer_id = str(row.get("customerId") or "")
        customer = canonical_by_id.get(customer_id) or (row.get("customerName") or row.get("customerCode") or "").strip() or "Không rõ khách hàng"
        try:
            details = row.get("invoiceDetails" if using_invoices else "orderDetails") or []
            if isinstance(details, str):
                details = json.loads(details)
        except json.JSONDecodeError:
            continue
            for item in details:
                code, name = str(item.get("productCode") or "").strip(), str(item.get("productName") or "").strip()
                if code or name:
                    key = (customer, f"{code} · {name}" if code else name)
                    customer_products[key][day] += 1
                    previous = latest_orders.get(key)
                    if previous is None or day > previous["date"]:
                        latest_orders[key] = {"date": day, "invoice": str(row.get("code") or row.get("id") or "")}
        for item in details:
            code = str(item.get("productCode") or "").strip()
            name = str(item.get("productName") or "").strip()
            if "{DEL}" in code.upper():
                continue
            if code.endswith("-") or code in ignored_codes:
                continue
            if code in code_aliases:
                canonical_code, canonical_name = code_aliases[code].split(" · ", 1)
                code, name = canonical_code, canonical_name
            if code or name:
                key = (customer, f"{code} · {name}" if code else name)
                customer_products[key][day] += 1
                previous = latest_orders.get(key)
                if previous is None or day > previous["date"]:
                    latest_orders[key] = {"date": day, "invoice": str(row.get("code") or row.get("id") or "")}
        rows += 1
    if not latest:
        return {"available": False, "reason": "Không có đơn hàng hợp lệ"}
    recent_start = latest - dt.timedelta(days=29)
    # Use the whole current year as the historical baseline, then compare it
    # with the last 30 days.
    baseline_start = dt.date(latest.year, 1, 1)
    stopped = []
    for (customer, product), dates in customer_products.items():
        baseline = sum(v for day, v in dates.items() if baseline_start <= day < recent_start)
        recent = sum(v for day, v in dates.items() if recent_start <= day <= latest)
        if baseline >= 2 and recent == 0:
            last = latest_orders[(customer, product)]
            stopped.append({"branch": customer, "product": product, "baselineOrders": baseline, "recentOrders": recent, "latestInvoice": last["invoice"], "latestPurchaseDate": str(last["date"]), "daysSincePurchase": (latest - last["date"]).days})
    stopped.sort(key=lambda x: (-x["baselineOrders"], x["branch"], x["product"]))
    all_days = [day for dates in customer_products.values() for day in dates]
    branch_options = sorted({item["branch"] for item in stopped})
    return {"available": True, "source": "customerId/customerName từ KiotViet orders", "rows": rows, "dataFrom": str(min(all_days)), "dataTo": str(latest), "baseline": {"from": str(baseline_start), "to": str(recent_start - dt.timedelta(days=1))}, "recent": {"from": str(recent_start), "to": str(latest)}, "stoppedCount": len(stopped), "branchOptions": branch_options, "customerAliases": aliases, "items": stopped}


def main():
    ANALYTICS.mkdir(parents=True, exist_ok=True)
    payload = {"generatedAt": dt.datetime.now(dt.timezone.utc).isoformat(), "schedule": cron_report(), "stoppedProducts": product_report()}
    if RAW_INVOICES_DIR.exists() and payload["stoppedProducts"].get("available"):
        payload["stoppedProducts"]["source"] = "customerId/customerName từ hóa đơn Hoàn thành KiotViet"
    _, aliases, ignored = product_aliases()
    payload["stoppedProducts"]["productAliases"] = aliases
    payload["stoppedProducts"]["ignoredInactiveCodes"] = sorted(ignored)
    payload["stoppedProducts"]["deletedCodeRule"] = "Loại mã chứa {DEL}"
    payload["stoppedProducts"]["trailingDashRule"] = "Loại mã kết thúc bằng -"
    OUT.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"ok": True, "output": str(OUT), "scheduleDays": payload["schedule"]["validDays"], "stoppedProducts": payload["stoppedProducts"].get("stoppedCount", 0), "dataTo": payload["stoppedProducts"].get("dataTo")}, ensure_ascii=False))


if __name__ == "__main__":
    main()
