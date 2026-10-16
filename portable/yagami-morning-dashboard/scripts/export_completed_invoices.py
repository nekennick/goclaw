#!/usr/bin/env python3
"""Export completed KiotViet invoices with product details, without third-party packages."""
from __future__ import annotations

import csv
import datetime as dt
import json
import os
import time
import urllib.parse
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ENV_FILE = Path(os.environ.get("KV_ENV_FILE", ROOT / "config/.env.kiotviet"))
OUT = Path(os.environ.get("KV_INVOICE_OUT", ROOT / "data/analytics/invoices-completed-2026.csv"))
TZ = dt.timezone(dt.timedelta(hours=7))


def env():
    values = {}
    for line in ENV_FILE.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if line and not line.startswith("#") and "=" in line:
            key, value = line.split("=", 1)
            values[key.strip()] = value.strip().strip('"').strip("'")
    return values


def get_json(request, timeout=90):
    with urllib.request.urlopen(request, timeout=timeout) as response:
        return json.loads(response.read().decode("utf-8"))


def token(config):
    body = urllib.parse.urlencode({"scopes": config.get("KIOTVIET_SCOPE", "PublicApi.Access"), "grant_type": "client_credentials", "client_id": config["KIOTVIET_CLIENT_ID"], "client_secret": config["KIOTVIET_CLIENT_SECRET"]}).encode()
    request = urllib.request.Request(config.get("KIOTVIET_AUTH_URL", "https://id.kiotviet.vn/connect/token"), data=body, headers={"Content-Type": "application/x-www-form-urlencoded"}, method="POST")
    return get_json(request, 30)["access_token"]


def fetch_page(config, access_token, params):
    url = config.get("KIOTVIET_API_URL", "https://public.kiotapi.com").rstrip("/") + "/invoices?" + urllib.parse.urlencode(params)
    request = urllib.request.Request(url, headers={"Authorization": "Bearer " + access_token, "Retailer": config["KIOTVIET_RETAILER"]})
    return get_json(request)


def parse_date(value):
    try:
        return dt.datetime.fromisoformat(str(value)[:19])
    except (ValueError, TypeError):
        return None


def complete(invoice):
    status = str(invoice.get("statusValue") or "").strip().upper()
    code = str(invoice.get("code") or "").upper()
    return status == "HOÀN THÀNH" and (code.startswith("HD") or code.startswith("HĐ"))


def main():
    config = env()
    now = dt.datetime.now(TZ).replace(tzinfo=None)
    start = dt.datetime(now.year, 1, 1)
    access_token = token(config)
    invoices, current = [], 0
    while True:
        payload = fetch_page(config, access_token, {"pageSize": 100, "currentItem": current, "orderBy": "purchaseDate", "orderDirection": "Desc", "fromPurchaseDate": start.strftime("%Y-%m-%dT%H:%M:%S"), "toPurchaseDate": now.strftime("%Y-%m-%dT%H:%M:%S")})
        batch = payload.get("data", []) if isinstance(payload, dict) else []
        if not batch:
            break
        invoices.extend(batch)
        current += len(batch)
        if len(batch) < 100 or current >= int(payload.get("total", current)):
            break
        time.sleep(0.05)
    complete_invoices = [x for x in invoices if (date := parse_date(x.get("purchaseDate"))) and start <= date <= now and complete(x)]
    OUT.parent.mkdir(parents=True, exist_ok=True)
    fields = ["invoiceId", "invoiceCode", "customerId", "customerCode", "customerName", "purchaseDate", "status", "statusValue", "orderCode", "invoiceDetails"]
    with OUT.open("w", encoding="utf-8-sig", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=fields)
        writer.writeheader()
        for invoice in complete_invoices:
            writer.writerow({"invoiceId": invoice.get("id"), "invoiceCode": invoice.get("code"), "customerId": invoice.get("customerId"), "customerCode": invoice.get("customerCode"), "customerName": invoice.get("customerName"), "purchaseDate": invoice.get("purchaseDate"), "status": invoice.get("status"), "statusValue": invoice.get("statusValue"), "orderCode": invoice.get("orderCode"), "invoiceDetails": json.dumps(invoice.get("invoiceDetails") or [], ensure_ascii=False)})
    print(json.dumps({"ok": True, "downloaded": len(invoices), "completed": len(complete_invoices), "from": start.date().isoformat(), "to": now.date().isoformat(), "output": str(OUT)}, ensure_ascii=False))


if __name__ == "__main__":
    main()
