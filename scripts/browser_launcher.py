#!/usr/bin/env python3
import argparse
import json
import os
import shutil
import signal
import subprocess
import sys
import time
from pathlib import Path


def resolve_browser() -> str:
    candidates = [
        os.environ.get("CHROMIUM_BINARY", "").strip(),
        os.environ.get("GOOGLE_CHROME_BIN", "").strip(),
        os.environ.get("CHROME_BIN", "").strip(),
        "chromium",
        "chromium-browser",
        "google-chrome",
        "google-chrome-stable",
    ]
    for candidate in candidates:
        if not candidate:
            continue
        path = shutil.which(candidate)
        if path:
            return path
    raise RuntimeError("no chromium binary found on host")


def launch_request(request_file: Path) -> None:
    with request_file.open("r", encoding="utf-8") as fh:
        request = json.load(fh)

    browser = resolve_browser()
    profile_path = request["profile_path"]
    target_url = request["target_url"]

    os.makedirs(profile_path, exist_ok=True)

    args = [
        browser,
        f"--user-data-dir={profile_path}",
        "--no-first-run",
        "--no-default-browser-check",
        "--disable-dev-shm-usage",
        "--new-window",
        target_url,
    ]
    if os.environ.get("BROWSER_LAUNCH_NO_SANDBOX", "").strip().lower() in {"1", "true", "yes", "on"}:
        args.insert(5, "--no-sandbox")
    if os.environ.get("BROWSER_LAUNCH_IGNORE_CERT_ERRORS", "").strip().lower() in {"1", "true", "yes", "on"}:
        args.insert(5 if "--no-sandbox" not in args else 6, "--ignore-certificate-errors")
        args.insert(5 if "--no-sandbox" not in args else 6, "--allow-running-insecure-content")

    print(f"[browser-launcher] launching: {' '.join(args)}", flush=True)
    subprocess.Popen(args, start_new_session=True)


def watch(request_dir: Path, interval: float) -> None:
    request_dir.mkdir(parents=True, exist_ok=True)
    processed_dir = request_dir / "_processed"
    failed_dir = request_dir / "_failed"
    processed_dir.mkdir(exist_ok=True)
    failed_dir.mkdir(exist_ok=True)

    print(f"[browser-launcher] watching {request_dir}", flush=True)

    stop = False

    def handle_signal(signum, _frame):
        nonlocal stop
        stop = True

    signal.signal(signal.SIGINT, handle_signal)
    signal.signal(signal.SIGTERM, handle_signal)

    while not stop:
        files = sorted(request_dir.glob("*.json"), key=lambda path: path.stat().st_mtime)
        for request_file in files:
            try:
                launch_request(request_file)
                request_file.rename(processed_dir / request_file.name)
            except Exception as exc:
                print(f"[browser-launcher] failed {request_file.name}: {exc}", file=sys.stderr, flush=True)
                try:
                    request_file.rename(failed_dir / request_file.name)
                except Exception:
                    pass
        time.sleep(interval)


def main() -> int:
    parser = argparse.ArgumentParser(description="Host-side Chromium launcher for profile login")
    parser.add_argument("command", choices=["watch", "launch"], help="run mode")
    parser.add_argument("--dir", default=os.environ.get("BROWSER_LAUNCH_REQUEST_DIR", ".runtime/browser-launch-requests"))
    parser.add_argument("--interval", type=float, default=2.0)
    parser.add_argument("--request-file", help="launch a single request file")
    args = parser.parse_args()

    request_dir = Path(args.dir).expanduser().resolve()
    if args.command == "watch":
        watch(request_dir, args.interval)
        return 0

    if not args.request_file:
        parser.error("--request-file is required for launch")
    launch_request(Path(args.request_file).expanduser().resolve())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
