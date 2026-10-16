#!/usr/bin/env python3
"""Zero-dependency local server for the portable Yagami morning dashboard."""
from __future__ import annotations

import datetime as dt
import json
import subprocess
import sys
import threading
import shutil
from http import HTTPStatus
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

ROOT = Path(__file__).resolve().parent
WEB_DIR = ROOT / "web"
DATA_DIR = ROOT / "data/reports"
ANALYTICS_FILE = ROOT / "data/analytics/analytics.json"
CAPTURE_DIR = ROOT / "data/screenshots"
SETTINGS_PATH = ROOT / "settings.json"
TZ = dt.timezone(dt.timedelta(hours=7))
SYNC_LOCK = threading.Lock()
SYNC_STATE = {"running": False, "lastStartedAt": None, "lastFinishedAt": None, "lastResult": None}


def load_settings() -> dict:
    defaults = {"port": 8765, "bindHost": "127.0.0.1", "autoSync": True, "dailyTime": "07:30"}
    if SETTINGS_PATH.exists():
        try:
            defaults.update(json.loads(SETTINGS_PATH.read_text(encoding="utf-8")))
        except json.JSONDecodeError:
            pass
    return defaults


def save_settings(value: dict) -> dict:
    current = load_settings()
    current.update({key: value[key] for key in ("autoSync", "dailyTime") if key in value})
    if "lanEnabled" in value:
        current["bindHost"] = "0.0.0.0" if value["lanEnabled"] else "127.0.0.1"
    if not isinstance(current["autoSync"], bool):
        current["autoSync"] = bool(current["autoSync"])
    if not isinstance(current["dailyTime"], str) or len(current["dailyTime"]) != 5:
        current["dailyTime"] = "07:30"
    SETTINGS_PATH.write_text(json.dumps(current, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return current


def chrome_path() -> str | None:
    candidates = [
        shutil.which("chrome"),
        r"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
        r"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
        str(Path.home() / "AppData/Local/Google/Chrome/Application/chrome.exe"),
    ]
    return next((candidate for candidate in candidates if candidate and Path(candidate).exists()), None)


def capture_dashboard(port: int) -> dict:
    chrome = chrome_path()
    if not chrome:
        return {"ok": False, "error": "Chưa tìm thấy Google Chrome để tạo ảnh."}
    CAPTURE_DIR.mkdir(parents=True, exist_ok=True)
    target = CAPTURE_DIR / f"bao-cao-sang-{dt.datetime.now(TZ).strftime('%Y%m%d-%H%M%S')}.png"
    try:
        subprocess.run([chrome, "--headless", "--disable-gpu", "--hide-scrollbars", "--window-size=1440,2600", f"--screenshot={target}", f"http://127.0.0.1:{port}/"], check=True, text=True, capture_output=True, timeout=45)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired) as err:
        return {"ok": False, "error": f"Không tạo được ảnh: {err}"}
    if not target.exists():
        return {"ok": False, "error": "Chrome không trả về tệp ảnh."}
    return {"ok": True, "filename": target.name, "url": f"/_captures/{target.name}"}


def read_json(path: Path, fallback):
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (FileNotFoundError, json.JSONDecodeError):
        return fallback


def latest_report():
    return read_json(DATA_DIR / "latest.json", None)


def analytics_report():
    return read_json(ANALYTICS_FILE, {"schedule": {}, "stoppedProducts": {}, "generatedAt": None})


def report_history():
    rows = []
    for path in sorted(DATA_DIR.glob("morning-report-*.json"), reverse=True)[:60]:
        item = read_json(path, None)
        if item:
            rows.append({"file": path.name, "reportDate": item.get("reportDate"), "generatedAt": item.get("generatedAt"), "summary": item.get("summary", {})})
    return rows


def start_sync():
    if not SYNC_LOCK.acquire(blocking=False):
        return False
    SYNC_STATE.update(running=True, lastStartedAt=dt.datetime.now(TZ).isoformat(), lastResult=None)

    def worker():
        try:
            result = subprocess.run([sys.executable, str(ROOT / "scripts/run_sync.py")], cwd=ROOT, text=True, capture_output=True, timeout=600)
            raw = (result.stdout or result.stderr or "").strip()
            try:
                payload = json.loads(raw.splitlines()[-1]) if raw else {"ok": False, "error": "Không có phản hồi từ tác vụ đồng bộ"}
            except json.JSONDecodeError:
                payload = {"ok": result.returncode == 0, "detail": raw[-3000:]}
            if result.returncode != 0:
                payload["ok"] = False
                payload.setdefault("error", result.stderr[-2000:] or raw[-2000:])
            SYNC_STATE["lastResult"] = payload
        except subprocess.TimeoutExpired:
            SYNC_STATE["lastResult"] = {"ok": False, "error": "Đồng bộ quá 10 phút nên đã dừng"}
        finally:
            SYNC_STATE.update(running=False, lastFinishedAt=dt.datetime.now(TZ).isoformat())
            SYNC_LOCK.release()

    threading.Thread(target=worker, daemon=True).start()
    return True


class Handler(SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(WEB_DIR), **kwargs)

    def log_message(self, fmt, *args):
        print(f"[{dt.datetime.now(TZ).strftime('%H:%M:%S')}] {fmt % args}")

    def json_response(self, payload, status=HTTPStatus.OK):
        encoded = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(encoded)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(encoded)

    def do_GET(self):
        if self.path == "/api/dashboard":
            self.json_response({"report": latest_report(), "settings": load_settings(), "sync": SYNC_STATE})
            return
        if self.path == "/api/history":
            self.json_response({"items": report_history()})
            return
        if self.path == "/api/analytics":
            self.json_response(analytics_report())
            return
        if self.path == "/api/sync-status":
            self.json_response(SYNC_STATE)
            return
        if self.path.startswith("/_captures/"):
            name = Path(self.path).name
            candidate = CAPTURE_DIR / name
            if candidate.suffix.lower() != ".png" or not candidate.is_file():
                self.send_error(HTTPStatus.NOT_FOUND)
                return
            body = candidate.read_bytes()
            self.send_response(HTTPStatus.OK)
            self.send_header("Content-Type", "image/png")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        if self.path == "/":
            self.path = "/index.html"
        super().do_GET()

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(length) if length else b"{}"
        try:
            body = json.loads(raw.decode("utf-8"))
        except json.JSONDecodeError:
            self.json_response({"ok": False, "error": "JSON không hợp lệ"}, HTTPStatus.BAD_REQUEST)
            return
        if self.path == "/api/sync":
            started = start_sync()
            self.json_response({"ok": started, "message": "Đang đồng bộ KiotViet" if started else "Đồng bộ đang chạy"}, HTTPStatus.ACCEPTED if started else HTTPStatus.CONFLICT)
            return
        if self.path == "/api/settings":
            self.json_response({"ok": True, "settings": save_settings(body)})
            return
        if self.path == "/api/capture":
            self.json_response(capture_dashboard(self.server.server_address[1]))
            return
        self.json_response({"ok": False, "error": "Không tìm thấy endpoint"}, HTTPStatus.NOT_FOUND)


def scheduler():
    last_run_date = ""
    while True:
        now = dt.datetime.now(TZ)
        settings = load_settings()
        due = now.strftime("%H:%M") == settings.get("dailyTime") and now.date().isoformat() != last_run_date
        if settings.get("autoSync") and due and start_sync():
            last_run_date = now.date().isoformat()
        threading.Event().wait(20)


if __name__ == "__main__":
    settings = load_settings()
    threading.Thread(target=scheduler, daemon=True).start()
    port = int(settings.get("port", 8765))
    bind_host = settings.get("bindHost", "127.0.0.1")
    print(f"Yagami Morning Dashboard: http://{bind_host}:{port}")
    ThreadingHTTPServer((bind_host, port), Handler).serve_forever()
