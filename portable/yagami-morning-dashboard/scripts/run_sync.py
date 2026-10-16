#!/usr/bin/env python3
"""Run a live KiotViet morning refresh and export its portable dashboard JSON."""
from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ENV_FILE = ROOT / "config/.env.kiotviet"


def main() -> int:
    if not ENV_FILE.exists():
        print(json.dumps({"ok": False, "error": "Thiếu config/.env.kiotviet. Hãy copy file template và điền credential KiotViet."}, ensure_ascii=False))
        return 2
    report_script = ROOT / "scripts/kv_morning_report.py"
    result = subprocess.run([sys.executable, str(report_script)], cwd=ROOT, text=True, capture_output=True)
    report_path = ROOT / "outputs/morning-report.txt"
    if result.returncode != 0 or not report_path.exists():
        print(json.dumps({"ok": False, "error": result.stderr[-2000:] or result.stdout[-2000:] or "Không tạo được báo cáo sáng"}, ensure_ascii=False))
        return result.returncode or 1
    export = subprocess.run([sys.executable, str(ROOT / "scripts/report_to_json.py"), str(report_path), "--out-dir", str(ROOT / "data/reports")], cwd=ROOT, text=True, capture_output=True)
    if export.returncode != 0:
        print(json.dumps({"ok": False, "error": export.stderr[-2000:] or export.stdout[-2000:]}, ensure_ascii=False))
        return export.returncode
    print(json.dumps({"ok": True, "message": "Đồng bộ KiotViet và cập nhật dashboard thành công", "detail": export.stdout.strip()}, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
