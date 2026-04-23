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

OPEN_CONTEXTS = []


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


def headless_enabled() -> bool:
    value = os.environ.get("BROWSER_LAUNCH_HEADLESS", "").strip().lower()
    if value in {"1", "true", "yes", "on"}:
        return True
    if value in {"0", "false", "no", "off"}:
        return False
    return not os.environ.get("DISPLAY") and not os.environ.get("WAYLAND_DISPLAY")


def cleanup_stale_profile_locks(profile_path: str) -> None:
    profile_dir = Path(profile_path)
    lock_path = profile_dir / "SingletonLock"
    if not lock_path.is_symlink():
        return

    try:
        target = os.readlink(lock_path)
    except OSError:
        return

    pid_text = target.rsplit("-", 1)[-1]
    if not pid_text.isdigit():
        return
    if Path(f"/proc/{pid_text}").exists():
        return

    for name in ("SingletonLock", "SingletonCookie", "SingletonSocket"):
        try:
            (profile_dir / name).unlink()
        except FileNotFoundError:
            pass


def launch_with_playwright(profile_path: str, target_url: str) -> bool:
    try:
        from playwright.sync_api import sync_playwright
    except Exception:
        return False

    args = [
        "--no-first-run",
        "--no-default-browser-check",
        "--disable-dev-shm-usage",
    ]
    if os.environ.get("BROWSER_LAUNCH_NO_SANDBOX", "").strip().lower() in {"1", "true", "yes", "on"}:
        args.append("--no-sandbox")
    if os.environ.get("BROWSER_LAUNCH_IGNORE_CERT_ERRORS", "").strip().lower() in {"1", "true", "yes", "on"}:
        args.extend(["--ignore-certificate-errors", "--allow-running-insecure-content"])
    if os.environ.get("WAYLAND_DISPLAY") and not os.environ.get("DISPLAY"):
        args.extend(["--ozone-platform=wayland", "--enable-features=UseOzonePlatform"])

    playwright = sync_playwright().start()
    context = playwright.chromium.launch_persistent_context(
        user_data_dir=profile_path,
        headless=headless_enabled(),
        args=args,
    )
    page = context.pages[0] if context.pages else context.new_page()
    page.goto(target_url, wait_until="domcontentloaded")
    OPEN_CONTEXTS.append((playwright, context))
    print(f"[browser-launcher] launching playwright chromium at {target_url}", flush=True)
    return True


def launch_request(request_file: Path) -> None:
    with request_file.open("r", encoding="utf-8") as fh:
        request = json.load(fh)

    profile_path = request["profile_path"]
    target_url = request["target_url"]

    os.makedirs(profile_path, exist_ok=True)
    cleanup_stale_profile_locks(profile_path)

    try:
        if launch_with_playwright(profile_path, target_url):
            return
    except Exception as exc:
        if "profile appears to be in use" in str(exc).lower():
            cleanup_stale_profile_locks(profile_path)
            if launch_with_playwright(profile_path, target_url):
                return
        raise

    browser = resolve_browser()
    args = [
        browser,
        f"--user-data-dir={profile_path}",
        "--no-first-run",
        "--no-default-browser-check",
        "--disable-dev-shm-usage",
        target_url,
    ]
    if os.environ.get("BROWSER_LAUNCH_NO_SANDBOX", "").strip().lower() in {"1", "true", "yes", "on"}:
        args.insert(5, "--no-sandbox")
    if os.environ.get("BROWSER_LAUNCH_IGNORE_CERT_ERRORS", "").strip().lower() in {"1", "true", "yes", "on"}:
        args.insert(5 if "--no-sandbox" not in args else 6, "--ignore-certificate-errors")
        args.insert(5 if "--no-sandbox" not in args else 6, "--allow-running-insecure-content")
    if headless_enabled():
        args.insert(5 if "--no-sandbox" not in args else 6, "--headless=new")
    else:
        args.insert(5 if "--no-sandbox" not in args else 6, "--new-window")

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
