#!/bin/bash
# Author: David M. Anderson
# Built with AI assistance (Claude, Anthropic)

set -euo pipefail
umask 022

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONTROL="$ROOT/packaging/control"
VER=$(awk '/^Version:/{print $2}' "$CONTROL")
ARCH=$(awk '/^Architecture:/{print $2}' "$CONTROL")
[ -n "$VER" ] && [ -n "$ARCH" ] || { echo "no Version: or Architecture: in $CONTROL" >&2; exit 1; }

STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
chmod 0755 "$STAGE"

mkdir -p "$STAGE/DEBIAN" "$STAGE/usr/bin" "$STAGE/usr/share/applications" "$STAGE/usr/share/doc/umwand"
cp "$CONTROL" "$STAGE/DEBIAN/control"

(cd "$ROOT" && CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build -trimpath \
    -ldflags "-s -w -X main.version=$VER" -o "$STAGE/usr/bin/umwand" .)
chmod 0755 "$STAGE/usr/bin/umwand"

install -m 0644 "$ROOT/packaging/umwand.desktop" "$STAGE/usr/share/applications/umwand.desktop"
install -D -m 0644 "$ROOT/packaging/icon.svg" "$STAGE/usr/share/icons/hicolor/scalable/apps/umwand.svg"
for size in 16 22 24; do
    install -D -m 0644 "$ROOT/packaging/icon-small.svg" "$STAGE/usr/share/icons/hicolor/${size}x${size}/apps/umwand.svg"
done
install -m 0644 "$ROOT/LICENSE" "$STAGE/usr/share/doc/umwand/copyright"
install -m 0644 "$ROOT/README.md" "$STAGE/usr/share/doc/umwand/README.md"

OUT="/tmp/umwand_${VER}_${ARCH}.deb"
dpkg-deb --build --root-owner-group "$STAGE" "$OUT" >/dev/null
echo "$OUT"
