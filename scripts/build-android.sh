#!/usr/bin/env bash
# Cross-compiles cmd/bot for Android (arm64), suitable for running under
# Termux on a phone/tablet. Android 12 devices are effectively all arm64, and
# it's the only android/* target Go can link without the NDK/cgo, so that's
# the only one built here.
set -euo pipefail
cd "$(dirname "$0")/.."

OUT="${1:-dist/base-bot-android-arm64}"
mkdir -p "$(dirname "$OUT")"

GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$OUT" ./cmd/bot

echo "Built $OUT"
file "$OUT" 2>/dev/null || true
