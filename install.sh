#!/bin/sh
# Rangkai installer: fetch a release binary, verify its SHA256, install to a bin
# directory. POSIX sh; needs only curl, sha256sum, and standard coreutils.
#
#   curl -fsSL https://github.com/arigustisj/rangkai/releases/latest/download/install.sh | sh
#
# Overrides: RANGKAI_VERSION (default latest), RANGKAI_BIN_DIR (default
# $HOME/.local/bin), RANGKAI_BASE_URL (for testing against a local mirror).
set -eu

REPO="arigustisj/rangkai"
VERSION="${RANGKAI_VERSION:-latest}"
BIN_DIR="${RANGKAI_BIN_DIR:-$HOME/.local/bin}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64 | amd64) ARCH=amd64 ;;
  aarch64 | arm64) ARCH=arm64 ;;
  *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;;
esac
case "$OS" in
  linux | darwin) ;;
  *) echo "unsupported OS: $OS" >&2; exit 1 ;;
esac

NAME="rangkai-${OS}-${ARCH}"
if [ -n "${RANGKAI_BASE_URL:-}" ]; then
  BASE="$RANGKAI_BASE_URL"
elif [ "$VERSION" = "latest" ]; then
  BASE="https://github.com/$REPO/releases/latest/download"
else
  BASE="https://github.com/$REPO/releases/download/$VERSION"
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT INT TERM

echo "Downloading $NAME ..."
curl -fsSL "$BASE/$NAME" -o "$TMP/$NAME"
curl -fsSL "$BASE/SHA256SUMS" -o "$TMP/SHA256SUMS"

echo "Verifying checksum ..."
LINE="$(grep "  $NAME\$" "$TMP/SHA256SUMS" || true)"
if [ -z "$LINE" ]; then
  echo "no checksum entry for $NAME" >&2
  exit 1
fi
printf '%s\n' "$LINE" | (cd "$TMP" && sha256sum -c -)

chmod +x "$TMP/$NAME"
mkdir -p "$BIN_DIR"
mv "$TMP/$NAME" "$BIN_DIR/rangkai"
echo "Installed: $BIN_DIR/rangkai"
"$BIN_DIR/rangkai" --version

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *) echo "Note: add $BIN_DIR to your PATH" ;;
esac
