#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
SOURCE_PNG="${REPOSITORY_DIR}/assets/BrewtifyerIcon.png"
OUTPUT_ICNS="${SCRIPT_DIR}/AppIcon.icns"
TEMP_DIR="$(mktemp -d /tmp/brewtifyer-icon.XXXXXX)"
ICONSET_DIR="${TEMP_DIR}/Brewtifyer.iconset"

cleanup() {
  rm -rf -- "${TEMP_DIR}"
}
trap cleanup EXIT

mkdir -p "${ICONSET_DIR}"

sizes=(16 32 32 64 128 256 256 512 512 1024)
names=(
  icon_16x16.png
  icon_16x16@2x.png
  icon_32x32.png
  icon_32x32@2x.png
  icon_128x128.png
  icon_128x128@2x.png
  icon_256x256.png
  icon_256x256@2x.png
  icon_512x512.png
  icon_512x512@2x.png
)

for index in "${!sizes[@]}"; do
  size="${sizes[${index}]}"
  sips -z "${size}" "${size}" "${SOURCE_PNG}" --out "${ICONSET_DIR}/${names[${index}]}" >/dev/null
done

cd "${REPOSITORY_DIR}"
go run ./tools/icnspack "${ICONSET_DIR}" "${OUTPUT_ICNS}"

echo "App-Icon erstellt: ${OUTPUT_ICNS}"
