#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  update-fbc-catalog-source.sh <channel> <bundle-image> [catalog-root] [default-channel]

Updates a source-controlled file-based catalog by:
  - rendering the immutable bundle image
  - writing/updating the package metadata
  - adding the bundle object if it does not already exist
  - promoting the bundle into the requested channel
EOF
}

if [[ $# -lt 2 || $# -gt 4 ]]; then
  usage
  exit 1
fi

CHANNEL="$1"
BUNDLE_IMAGE="$2"
CATALOG_ROOT="${3:-catalog}"
DEFAULT_CHANNEL="${4:-candidate}"

for bin in opm yq; do
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

RENDER_FILE="$TMP_DIR/render.yaml"
BUNDLE_FILE_TMP="$TMP_DIR/bundle.yaml"

opm render "$BUNDLE_IMAGE" -o yaml > "$RENDER_FILE"
yq eval 'select(.schema == "olm.bundle")' "$RENDER_FILE" > "$BUNDLE_FILE_TMP"

PACKAGE_NAME="$(yq -r '.package' "$BUNDLE_FILE_TMP")"
BUNDLE_NAME="$(yq -r '.name' "$BUNDLE_FILE_TMP")"

if [[ -z "$PACKAGE_NAME" || "$PACKAGE_NAME" == "null" || -z "$BUNDLE_NAME" || "$BUNDLE_NAME" == "null" ]]; then
  echo "failed to determine package or bundle name from $BUNDLE_IMAGE" >&2
  exit 1
fi

PACKAGE_DIR="$CATALOG_ROOT/$PACKAGE_NAME"
CHANNELS_DIR="$PACKAGE_DIR/channels"
BUNDLES_DIR="$PACKAGE_DIR/bundles"
PACKAGE_FILE="$PACKAGE_DIR/package.yaml"
CHANNEL_FILE="$CHANNELS_DIR/$CHANNEL.yaml"
BUNDLE_FILE="$BUNDLES_DIR/$BUNDLE_NAME.yaml"

mkdir -p "$CHANNELS_DIR" "$BUNDLES_DIR"

PACKAGE_DEFAULT_CHANNEL="$DEFAULT_CHANNEL"
if [[ "$CHANNEL" == "stable" ]]; then
  PACKAGE_DEFAULT_CHANNEL="stable"
elif [[ -f "$PACKAGE_FILE" ]]; then
  CURRENT_DEFAULT_CHANNEL="$(yq -r '.defaultChannel // ""' "$PACKAGE_FILE")"
  if [[ -n "$CURRENT_DEFAULT_CHANNEL" && "$CURRENT_DEFAULT_CHANNEL" != "null" ]]; then
    PACKAGE_DEFAULT_CHANNEL="$CURRENT_DEFAULT_CHANNEL"
  fi
fi

if [[ -f "$PACKAGE_FILE" ]]; then
  PACKAGE_NAME="$PACKAGE_NAME" DEFAULT_CHANNEL="$PACKAGE_DEFAULT_CHANNEL" yq -i '
    .schema = "olm.package" |
    .name = strenv(PACKAGE_NAME) |
    .defaultChannel = strenv(DEFAULT_CHANNEL)
  ' "$PACKAGE_FILE"
else
  cat > "$PACKAGE_FILE" <<EOF
schema: olm.package
name: $PACKAGE_NAME
defaultChannel: $PACKAGE_DEFAULT_CHANNEL
EOF
fi

cp "$BUNDLE_FILE_TMP" "$BUNDLE_FILE"

if [[ -f "$CHANNEL_FILE" ]]; then
  PACKAGE_NAME="$PACKAGE_NAME" CHANNEL="$CHANNEL" yq -i '
    .schema = "olm.channel" |
    .package = strenv(PACKAGE_NAME) |
    .name = strenv(CHANNEL) |
    .entries = (.entries // [])
  ' "$CHANNEL_FILE"
else
  cat > "$CHANNEL_FILE" <<EOF
schema: olm.channel
package: $PACKAGE_NAME
name: $CHANNEL
entries: []
EOF
fi

if ! BUNDLE_NAME="$BUNDLE_NAME" yq -e '.entries[]? | select(.name == strenv(BUNDLE_NAME))' "$CHANNEL_FILE" >/dev/null 2>&1; then
  PREVIOUS_HEAD="$(yq -r '.entries[0].name // ""' "$CHANNEL_FILE")"
  if [[ -n "$PREVIOUS_HEAD" ]]; then
    BUNDLE_NAME="$BUNDLE_NAME" PREVIOUS_HEAD="$PREVIOUS_HEAD" yq -i '
      .entries = [{"name": strenv(BUNDLE_NAME), "replaces": strenv(PREVIOUS_HEAD)}] + (.entries // [])
    ' "$CHANNEL_FILE"
  else
    BUNDLE_NAME="$BUNDLE_NAME" yq -i '
      .entries = [{"name": strenv(BUNDLE_NAME)}]
    ' "$CHANNEL_FILE"
  fi
fi

echo "Updated catalog source for package=$PACKAGE_NAME channel=$CHANNEL bundle=$BUNDLE_NAME"
