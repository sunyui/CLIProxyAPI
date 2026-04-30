#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
WORK_DIR="$DIST_DIR/package-work"
VERSION="${VERSION:-team-dist}"
GOROOT_VALUE="${CLIPROXYAPI_GOROOT:-/usr/local/Cellar/go/1.26.2/libexec}"
GOPROXY_VALUE="${CLIPROXYAPI_GOPROXY:-https://goproxy.cn,direct}"

payload_files=(
  "config.example.yaml"
  "docs/installation.md:installation.md"
  "docs/team-config.yaml:team-config.yaml"
)

targets=(
  "darwin arm64 cli-proxy-api cli-proxy-api_team-dist_darwin_arm64.tar.gz"
  "darwin amd64 cli-proxy-api cli-proxy-api_team-dist_darwin_amd64.tar.gz"
  "linux amd64 cli-proxy-api cli-proxy-api_team-dist_linux_amd64.tar.gz"
  "windows amd64 cli-proxy-api.exe cli-proxy-api_team-dist_windows_amd64.zip"
)

rm -rf "$DIST_DIR"
mkdir -p "$WORK_DIR"

for target in "${targets[@]}"; do
  read -r goos goarch binary archive <<<"$target"
  package_dir="$WORK_DIR/cli-proxy-api_${goos}_${goarch}"
  mkdir -p "$package_dir"

  GOROOT="$GOROOT_VALUE" \
  GOPROXY="$GOPROXY_VALUE" \
  CGO_ENABLED=0 \
  GOOS="$goos" \
  GOARCH="$goarch" \
    go build -ldflags "-s -w -X 'main.Version=$VERSION'" -o "$package_dir/$binary" ./cmd/server

  for entry in "${payload_files[@]}"; do
    src="${entry%%:*}"
    dst="${entry#*:}"
    if [[ "$src" == "$dst" ]]; then
      dst="$(basename "$src")"
    fi
    cp "$ROOT_DIR/$src" "$package_dir/$dst"
  done

  if [[ "$archive" == *.zip ]]; then
    python3 - <<PY
from pathlib import Path
import zipfile
base = Path(r"$package_dir")
with zipfile.ZipFile(Path(r"$DIST_DIR") / "$archive", "w", compression=zipfile.ZIP_DEFLATED) as zf:
    for name in ["$binary", "config.example.yaml", "installation.md", "team-config.yaml"]:
        zf.write(base / name, name)
PY
  else
    COPYFILE_DISABLE=1 tar -C "$package_dir" -czf "$DIST_DIR/$archive" "$binary" config.example.yaml installation.md team-config.yaml
  fi
done

(
  cd "$DIST_DIR"
  shasum -a 256 cli-proxy-api_team-dist_* > checksums.txt
)

rm -rf "$WORK_DIR"

python3 - <<'PY'
from pathlib import Path
import tarfile, zipfile
allowed_unix = {'cli-proxy-api', 'config.example.yaml', 'installation.md', 'team-config.yaml'}
allowed_win = {'cli-proxy-api.exe', 'config.example.yaml', 'installation.md', 'team-config.yaml'}
for archive in sorted(Path('dist').glob('*.tar.gz')):
    with tarfile.open(archive) as tf:
        names = {n.lstrip('./') for n in tf.getnames() if n.lstrip('./')}
    if names != allowed_unix:
        raise SystemExit(f'{archive}: unexpected files {sorted(names)}')
for archive in sorted(Path('dist').glob('*.zip')):
    with zipfile.ZipFile(archive) as zf:
        names = {n for n in zf.namelist() if n and not n.endswith('/')}
    if names != allowed_win:
        raise SystemExit(f'{archive}: unexpected files {sorted(names)}')
print('team dist packages built successfully')
PY
