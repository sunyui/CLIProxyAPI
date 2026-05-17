#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
INSTALL_DIR="${CLIPROXYAPI_INSTALL_DIR:-$HOME/.local/share/cliproxyapi-team}"
CONFIG_FILE="${CLIPROXYAPI_CONFIG:-$HOME/.cli-proxy-api/config.yaml}"
PLIST_FILE="${CLIPROXYAPI_PLIST:-$HOME/Library/LaunchAgents/com.sunyan.cliproxyapi.plist}"
LABEL="${CLIPROXYAPI_LABEL:-com.sunyan.cliproxyapi}"
DOMAIN="gui/$(id -u)"

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    echo "unsupported OS: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

archive="$DIST_DIR/cli-proxy-api_team-dist_${os}_${arch}.tar.gz"
if [[ ! -f "$archive" ]]; then
  echo "missing dist archive: $archive" >&2
  exit 1
fi
if [[ ! -f "$CONFIG_FILE" ]]; then
  echo "missing config file: $CONFIG_FILE" >&2
  exit 1
fi
if [[ "$os" == "darwin" && ! -f "$PLIST_FILE" ]]; then
  echo "missing launchd plist: $PLIST_FILE" >&2
  exit 1
fi

(cd "$DIST_DIR" && shasum -a 256 -c checksums.txt >/dev/null)

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT

tar -xzf "$archive" -C "$tmpdir"

if [[ "$os" == "darwin" ]]; then
  launchctl bootout "$DOMAIN" "$PLIST_FILE" >/dev/null 2>&1 || true
fi

install -d "$INSTALL_DIR"
install -m 0755 "$tmpdir/cli-proxy-api" "$INSTALL_DIR/cli-proxy-api"
install -m 0644 "$tmpdir/config.example.yaml" "$INSTALL_DIR/config.example.yaml"
install -m 0644 "$tmpdir/installation.md" "$INSTALL_DIR/installation.md"
install -m 0644 "$tmpdir/team-config.yaml" "$INSTALL_DIR/team-config.yaml"

if [[ "$os" == "darwin" ]]; then
  launchctl bootstrap "$DOMAIN" "$PLIST_FILE" >/dev/null 2>&1 || true
  launchctl kickstart -k "$DOMAIN/$LABEL"
  launchctl print "$DOMAIN/$LABEL" | grep -E 'state =|program =|pid =|last exit code =' || true
fi

echo "deployed $archive to $INSTALL_DIR"
echo "config remains $CONFIG_FILE"
