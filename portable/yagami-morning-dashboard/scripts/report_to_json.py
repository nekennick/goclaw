#!/usr/bin/env python3
"""Turn the canonical Zalo morning report into portable dashboard data."""
from __future__ import annotations

import argparse
import datetime as dt
import json
import re
from pathlib import Path

TZ = dt.timezone(dt.timedelta(hours=7))
WAREHOUSES = ["Tổng kho YAGAMI", "KHO YAGAMI BÌNH DƯƠNG"]
SKU_CODES = ["SML", "SMN", "SLTL", "MYG"]


def number(value: str) -> float:
    value = value.strip().replace(".", "").replace(",", ".")
    return float(value) if value else 0


def whole(value: str) -> int | float:
    parsed = number(value)
    return int(parsed) if parsed.is_integer() else parsed


def money(value: str) -> int:
    return int(number(value))


def match(text: str, pattern: str, default=None, flags=0):
    found = re.search(pattern, text, flags)
    return found.groups() if found else default


def parse(path: Path) -> dict:
    text = path.read_text(encoding="utf-8")
    lines = [line.rstrip() for line in text.splitlines()]
    now = dt.datetime.now(TZ)
    date_match = match(text, r"(?:Thứ .*?), (\d{2}/\d{2}/\d{4})")
    report_date = date_match[0] if date_match else now.strftime("%d/%m/%Y")
    schedule = match(text, r"• Lịch .*?: (\d+) chi nhánh", ("0",))
    placed = match(text, r"• Chi nhánh đã đặt: (\d+) chi nhánh / (\d+) phiếu / ([\d.]+)đ", ("0", "0", "0"))
    missing = match(text, r"• Chi nhánh chưa đặt: (\d+) chi nhánh", ("0",))
    external = match(text, r"• Gửi xe ngoài hôm nay: (\d+) phiếu / ([\d.]+)đ", ("0", "0"))
    off_schedule = match(text, r"• Đơn ngoài lịch khác: (\d+) phiếu / ([\d.]+)đ", ("0", "0"))
    orders_time = match(text, r"Mốc dữ liệu đơn hàng: .*? đến (\d{2}:\d{2})")
    inventory_time = match(text, r"Dữ liệu tồn kho .*? lúc (\d{2}:\d{2})")
    warning = match(text, r"• Cảnh báo chính: (.+)", ("Chưa có cảnh báo",))

    warehouse_rows = {name: {"name": name, "branches": 0, "orders": 0, "value": 0, "branchNames": []} for name in WAREHOUSES}
    missing_by_warehouse = {name: [] for name in WAREHOUSES}
    external_by_warehouse = {name: {"branches": 0, "orders": 0, "value": 0, "branchNames": []} for name in WAREHOUSES}
    off_schedule_by_warehouse = {name: {"orders": 0, "value": 0} for name in WAREHOUSES}
    inventory = {name: {} for name in WAREHOUSES}

    section = ""
    current_warehouse = ""
    for line in lines:
        if line == "🚚 Đóng hàng theo lịch": section = "scheduled"; continue
        if line == "📍 Chi nhánh chưa đặt:": section = "missing"; continue
        if line == "🚛 Gửi xe ngoài hôm nay:": section = "external"; continue
        if line == "🧾 Đơn ngoài lịch khác:": section = "off_schedule"; continue
        if line == "📦 4 mã siết tồn — đơn theo lịch + gửi xe ngoài": section = "inventory"; continue
        if line == "⚠️ Cần theo dõi:": section = "alerts"; continue
        if line.startswith("🏭 "):
            candidate = line[2:].rstrip(":")
            current_warehouse = candidate if candidate in WAREHOUSES else ""
            continue
        if not line.startswith("• "):
            continue
        content = line[2:]
        if section == "scheduled" and current_warehouse:
            row = match(content, r"(\d+) chi nhánh / (\d+) phiếu / ([\d.]+)đ")
            if row:
                warehouse_rows[current_warehouse].update(branches=int(row[0]), orders=int(row[1]), value=money(row[2]))
            elif content != "không có":
                warehouse_rows[current_warehouse]["branchNames"] = [item.strip() for item in content.split(",") if item.strip()]
        elif section == "missing":
            row = match(content, r"(.+?): (.+)")
            if row and row[0] in missing_by_warehouse and row[1] != "không có":
                missing_by_warehouse[row[0]] = [item.strip() for item in row[1].split(",") if item.strip()]
        elif section == "external":
            row = match(content, r"(.+?): (\d+) chi nhánh / (\d+) phiếu / ([\d.]+)đ — (.+)")
            if row and row[0] in external_by_warehouse:
                external_by_warehouse[row[0]] = {"branches": int(row[1]), "orders": int(row[2]), "value": money(row[3]), "branchNames": [] if row[4] == "không có" else [x.strip() for x in row[4].split(",") if x.strip()]}
        elif section == "off_schedule":
            row = match(content, r"(.+?): (\d+) phiếu / ([\d.]+)đ")
            if row and row[0] in off_schedule_by_warehouse:
                off_schedule_by_warehouse[row[0]] = {"orders": int(row[1]), "value": money(row[2])}
        elif section == "inventory" and current_warehouse:
            row = match(content, r"(SML|SMN|SLTL|MYG): cần (-?[\d.,]+) \| tồn (-?[\d.,]+) \| giữ (-?[\d.,]+) \| khả dụng (-?[\d.,]+)(.*)")
            if row:
                need, on_hand, reserved, available = map(whole, row[1:5])
                inventory[current_warehouse][row[0]] = {"code": row[0], "need": need, "onHand": on_hand, "reserved": reserved, "available": available, "shortage": max(0, need - available), "warning": "thiếu" in row[5].lower()}

    alerts = []
    capture_alerts = False
    for line in lines:
        if line == "⚠️ Cần theo dõi:": capture_alerts = True; continue
        if capture_alerts and line.startswith("• "): alerts.append(line[2:])

    return {
        "schemaVersion": 1,
        "generatedAt": now.isoformat(),
        "reportDate": report_date,
        "reportTitle": "Báo cáo sáng hôm nay",
        "source": {"type": "KiotViet", "rawReport": str(path.as_posix())},
        "sync": {"ordersThrough": orders_time[0] if orders_time else None, "inventorySyncedAt": inventory_time[0] if inventory_time else None, "isCatchUp": "Báo cáo bù" in text},
        "summary": {"scheduledBranches": int(schedule[0]), "placedBranches": int(placed[0]), "scheduledOrders": int(placed[1]), "scheduledValue": money(placed[2]), "missingBranches": int(missing[0]), "externalOrders": int(external[0]), "externalValue": money(external[1]), "offScheduleOrders": int(off_schedule[0]), "offScheduleValue": money(off_schedule[1]), "mainWarning": warning[0]},
        "warehouses": list(warehouse_rows.values()),
        "missingByWarehouse": missing_by_warehouse,
        "externalByWarehouse": external_by_warehouse,
        "offScheduleByWarehouse": off_schedule_by_warehouse,
        "inventory": inventory,
        "alerts": alerts,
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("source", type=Path)
    parser.add_argument("--out-dir", type=Path, default=Path("data/reports"))
    args = parser.parse_args()
    report = parse(args.source)
    args.out_dir.mkdir(parents=True, exist_ok=True)
    stamp = dt.datetime.now(TZ).strftime("%Y-%m-%dT%H%M%S")
    dated = args.out_dir / f"morning-report-{stamp}.json"
    payload = json.dumps(report, ensure_ascii=False, indent=2) + "\n"
    dated.write_text(payload, encoding="utf-8")
    (args.out_dir / "latest.json").write_text(payload, encoding="utf-8")
    print(json.dumps({"ok": True, "report": str(dated), "latest": str(args.out_dir / 'latest.json')}, ensure_ascii=False))


if __name__ == "__main__":
    main()
