#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  build-fbc-catalog-from-source.sh <catalog-root> <catalog-image>

Validates a source-controlled file-based catalog, generates a catalog Dockerfile
in a temporary directory, and builds the catalog image.
EOF
}

if [[ $# -ne 2 ]]; then
  usage
  exit 1
fi

CATALOG_ROOT="$1"
CATALOG_IMAGE="$2"

OPM_BIN="${OPM_BIN:-opm}"

for bin in "$OPM_BIN" podman; do
  command -v "$bin" >/dev/null 2>&1 || {
    echo "$bin is required" >&2
    exit 1
  }
done

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

CATALOG_NAME="$(basename "$CATALOG_ROOT")"
cp -R "$CATALOG_ROOT" "$TMP_DIR/$CATALOG_NAME"

"$OPM_BIN" validate "$TMP_DIR/$CATALOG_NAME"
"$OPM_BIN" generate dockerfile "$TMP_DIR/$CATALOG_NAME"

podman build -f "$TMP_DIR/${CATALOG_NAME}.Dockerfile" -t "$CATALOG_IMAGE" "$TMP_DIR"
