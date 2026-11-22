#!/bin/sh
# Cross-compile release binaries + SHA256SUMS into dist/, and stage install.sh.
# Upload afterwards with: gh release create <tag> dist/* --target staging
# Pure Go (CGO disabled); linux/amd64 + linux/arm64 cover PC, servers, Termux.
set -eu
cd "$(dirname "$0")/.."

mkdir -p dist
for TARGET in linux/amd64 linux/arm64; do
  OS="${TARGET%/*}"
  ARCH="${TARGET#*/}"
  OUT="dist/rangkai-${OS}-${ARCH}"
  echo "building $OUT"
  CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" go build -trimpath -ldflags "-s -w" -o "$OUT" ./cmd/rangkai
done

cp install.sh dist/install.sh
(cd dist && sha256sum rangkai-* > SHA256SUMS)
echo "dist ready:"
ls -1 dist
