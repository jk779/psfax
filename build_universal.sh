#!/usr/bin/env bash
# build_universal.sh — build macOS universal2 binary for psfax
# usage:
#   ./build_universal.sh              # build only
#   SIGN=1 ./build_universal.sh       # build + ad-hoc codesign
#   OUT=psfax ./build_universal.sh    # change output name (default: psfax)

set -euo pipefail

OUT="${OUT:-psfax}"
APP="${OUT}"
TMP_ARM="${OUT}-arm64"
TMP_AMD="${OUT}-amd64"

# minimal go sanity check
command -v go >/dev/null || { echo "go not found"; exit 1; }
command -v lipo >/dev/null || { echo "lipo not found (Xcode CLT)"; exit 1; }

# optional version stamp (purely cosmetic)
STAMP="$(date -u +%Y%m%d%H%M%S)"
LDFLAGS="-s -w -X main.buildStamp=${STAMP}"

echo "==> building arm64…"
GOOS=darwin GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o "${TMP_ARM}"

echo "==> building amd64…"
GOOS=darwin GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o "${TMP_AMD}"

echo "==> lipo create → ${APP}"
lipo -create -output "${APP}" "${TMP_ARM}" "${TMP_AMD}"

# clean up temps
rm -f "${TMP_ARM}" "${TMP_AMD}"

# optional ad-hoc sign (for local use)
if [[ "${SIGN:-0}" == "1" ]]; then
  if command -v codesign >/dev/null; then
    echo "==> ad-hoc signing…"
    codesign --force --sign - --timestamp --options=runtime "${APP}"
    codesign --verify --verbose "${APP}" || { echo "codesign verify failed"; exit 1; }
  else
    echo "codesign not found; skipping sign"
  fi
fi

echo "==> done"
file "${APP}"
shasum -a 256 "${APP}" || true