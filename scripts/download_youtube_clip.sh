#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <youtube-url> [output-prefix]" >&2
  exit 1
fi

url="$1"
prefix="${2:-download_$(date +%s)}"
output_path="${prefix}.mp4"

cleanup() {
  rm -f "${prefix}".*.mp4 "${prefix}".*.part "${prefix}".*.tmp 2>/dev/null || true
}

run_ytdlp() {
  local label="$1"
  shift
  echo "==> trying: ${label}" >&2
  local ytdlp_bin=()
  if command -v yt-dlp >/dev/null 2>&1; then
    ytdlp_bin=(yt-dlp)
  elif command -v python3 >/dev/null 2>&1; then
    ytdlp_bin=(python3 -m yt_dlp)
  elif command -v python >/dev/null 2>&1; then
    ytdlp_bin=(python -m yt_dlp)
  else
    echo "yt-dlp executable and python module are unavailable" >&2
    return 1
  fi

  if "${ytdlp_bin[@]}" \
    --ignore-config \
    --proxy "" \
    --force-ipv4 \
    --no-check-certificates \
    --no-playlist \
    -f "bestvideo[height<=1080][ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best" \
    -o "${prefix}.%(ext)s" \
    "$@" \
    "${url}"; then
    local found
    found="$(ls -1 "${prefix}".* 2>/dev/null | head -n1 || true)"
    if [[ -z "${found}" ]]; then
      echo "download succeeded but output file not found" >&2
      return 1
    fi
    mv -f "${found}" "${output_path}"
    echo "${output_path}"
    return 0
  fi
  return 1
}

trap cleanup EXIT

strategies=(
  "plain::"
  "browser_headers::--add-header Referer:https://www.youtube.com/ --add-header Origin:https://www.youtube.com --user-agent Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"
  "web_clients::--extractor-args youtube:player_client=web,web_creator,mweb,tv,android"
  "cookies_from_browser::--cookies-from-browser ${YTDLP_COOKIES_FROM_BROWSER:-chrome:Default}"
  "impersonate::--impersonate ${YTDLP_IMPERSONATE:-chrome:windows-10}"
  "js_runtime::--js-runtimes ${YTDLP_JS_RUNTIME:-node}"
)

for strategy in "${strategies[@]}"; do
  IFS="::" read -r label args <<<"${strategy}"
  if [[ "${label}" == "cookies_from_browser" && -z "${YTDLP_COOKIES_FROM_BROWSER:-chrome:Default}" ]]; then
    continue
  fi
  if [[ -z "${args}" ]]; then
    if run_ytdlp "${label}"; then
      exit 0
    fi
  else
    # shellcheck disable=SC2206
    extra_args=(${args})
    if run_ytdlp "${label}" "${extra_args[@]}"; then
      exit 0
    fi
  fi
done

echo "all strategies failed for: ${url}" >&2
exit 1
