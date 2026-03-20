#!/usr/bin/env bash

set -euo pipefail

REPO="devopshouse/tmswitch"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "error: required command not found: $1" >&2
    exit 1
  }
}

need_cmd curl
need_cmd tar
need_cmd uname

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    echo "error: unsupported operating system: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64|amd64) arch="x86_64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "error: unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

latest_tag="$(
  curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | sed -n 's/.*"tag_name": "\([^"]*\)".*/\1/p' \
  | head -n 1
)"

if [[ -z "$latest_tag" ]]; then
  echo "error: could not determine latest release tag" >&2
  exit 1
fi

version="${latest_tag#v}"
archive="tmswitch_${version}_${os}_${arch}.tar.gz"
download_url="https://github.com/$REPO/releases/download/${latest_tag}/${archive}"

echo "Downloading $download_url"
curl -fsSL "$download_url" -o "$tmpdir/$archive"
tar -xzf "$tmpdir/$archive" -C "$tmpdir"

if [[ ! -f "$tmpdir/tmswitch" ]]; then
  echo "error: extracted archive does not contain tmswitch binary" >&2
  exit 1
fi

chmod +x "$tmpdir/tmswitch"

if [[ -w "$INSTALL_DIR" ]]; then
  mv "$tmpdir/tmswitch" "$INSTALL_DIR/tmswitch"
else
  sudo mv "$tmpdir/tmswitch" "$INSTALL_DIR/tmswitch"
fi

echo "Installed tmswitch ${latest_tag} to $INSTALL_DIR/tmswitch"
