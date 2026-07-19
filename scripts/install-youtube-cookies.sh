#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/install-youtube-cookies.sh /path/to/cookies.txt [youtube_url]

Installs a fresh Netscape-format YouTube cookies export into /home/toto/yt-dlp,
validates it with yt-dlp inside the TgMusicBot container, and restarts the bot
only when validation succeeds.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" || $# -lt 1 ]]; then
  usage
  exit 0
fi

src=$1
probe_url=${2:-"https://www.youtube.com/watch?v=Bmwkx0tSvKc"}
repo_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cookie_dir=${COOKIE_DIR:-/home/toto/yt-dlp}
proxy_url=${COOKIE_PROXY:-}

if [[ -z "$proxy_url" && -f "$repo_dir/.env" ]]; then
  proxy_url=$(awk -F= '$1 == "PROXY" {sub(/^[ \t]+/, "", $2); sub(/[ \t]+$/, "", $2); print $2; exit}' "$repo_dir/.env")
fi

if [[ ! -f "$src" ]]; then
  echo "Cookie file not found: $src" >&2
  exit 1
fi

if ! grep -qi "Netscape HTTP Cookie File" "$src"; then
  echo "This does not look like a Netscape-format cookies file." >&2
  echo "Export cookies as cookies.txt, not JSON." >&2
  exit 1
fi

if ! awk 'BEGIN{ok=0} !/^#/ && NF >= 7 && ($1 ~ /youtube|google/) {ok=1} END{exit ok ? 0 : 1}' "$src"; then
  echo "No youtube.com/google.com cookies found in the export." >&2
  exit 1
fi

mkdir -p "$cookie_dir"
stamp=$(date -u +%Y%m%d-%H%M%S)
dest="$cookie_dir/cookies-$stamp.txt"
cp "$src" "$dest"
chmod 600 "$dest"

echo "Installed: $dest"
echo "Validating with: $probe_url"

yt_dlp_proxy_args=""
if [[ -n "$proxy_url" ]]; then
  echo "Using proxy: $proxy_url"
  yt_dlp_proxy_args="--proxy '$proxy_url'"
fi

docker run --rm \
  --entrypoint sh \
  -v "$cookie_dir:/cookies" \
  tgmusicbot-go-tgmusic \
  -lc "yt-dlp --cookies /cookies/$(basename "$dest") $yt_dlp_proxy_args --simulate --no-warnings --print title --print id '$probe_url'"

echo "Validation passed. Restarting TgMusicBot..."
docker compose -f "$repo_dir/docker-compose.yml" restart tgmusic
echo "Done."
