#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
APP_EXECUTABLE="${REPOSITORY_DIR}/dist/Brewtifyer.app/Contents/MacOS/Brewtifyer"

"${SCRIPT_DIR}/build.sh"
exec "${APP_EXECUTABLE}"
