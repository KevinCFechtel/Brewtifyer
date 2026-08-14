#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
SOURCE_SVG="${REPOSITORY_DIR}/internal/tray/menu_bar_icon.svg"
OUTPUT_PNG="${REPOSITORY_DIR}/internal/tray/menu_bar_icon.png"
TEMP_DIR="$(mktemp -d /tmp/brewtifyer-menu-bar-icon.XXXXXX)"

cleanup() {
  rm -rf -- "${TEMP_DIR}"
}
trap cleanup EXIT

qlmanage -t -s 1024 -o "${TEMP_DIR}" "${SOURCE_SVG}" >/dev/null
sips -z 32 32 \
  "${TEMP_DIR}/menu_bar_icon.svg.png" \
  --out "${TEMP_DIR}/menu_bar_icon.png" >/dev/null

cd "${REPOSITORY_DIR}"
go run ./tools/templatemask "${TEMP_DIR}/menu_bar_icon.png" "${OUTPUT_PNG}"

echo "Menüleisten-Icon erstellt: ${OUTPUT_PNG}"
