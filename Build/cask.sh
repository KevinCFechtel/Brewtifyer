#!/usr/bin/env bash
set -euo pipefail

# Generates the Homebrew cask for the current release archive.
#
# The cask itself lives in a separate tap repository (a GitHub repository named
# homebrew-tap). This script only produces the file so that version and
# checksum can never drift from the artifact that Build/release.sh produced.
#
# Usage:
#   ./Build/release.sh          # produce and verify the release archive
#   ./Build/cask.sh             # print the cask
#   ./Build/cask.sh Casks/brewtifyer.rb   # or write it somewhere

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
RELEASE_DIR="${REPOSITORY_DIR}/dist/release"
OUTPUT_PATH="${1:-}"

# shellcheck source=version.sh
source "${SCRIPT_DIR}/version.sh"

ARCHIVE="${RELEASE_DIR}/Brewtifyer-${APP_VERSION}-macos-universal.zip"

if [[ ! -f "${ARCHIVE}" ]]; then
  echo "Release archive is missing: ${ARCHIVE}" >&2
  echo "Run ./Build/release.sh first." >&2
  exit 1
fi

ARCHIVE_SHA256="$(shasum -a 256 "${ARCHIVE}" | awk '{print $1}')"
if [[ -z "${ARCHIVE_SHA256}" ]]; then
  echo "Checksum for ${ARCHIVE} could not be computed." >&2
  exit 1
fi

# macOS 13 Ventura matches LSMinimumSystemVersion in Build/Info.plist.
CASK_CONTENTS="$(
  cat <<CASK
cask "brewtifyer" do
  version "${APP_VERSION}"
  sha256 "${ARCHIVE_SHA256}"

  url "https://github.com/KevinCFechtel/Brewtifyer/releases/download/v#{version}/Brewtifyer-#{version}-macos-universal.zip",
      verified: "github.com/KevinCFechtel/Brewtifyer/"
  name "Brewtifyer"
  desc "Menu bar app for Homebrew formula and cask updates"
  homepage "https://github.com/KevinCFechtel/Brewtifyer"

  livecheck do
    url :url
    strategy :github_latest
  end

  depends_on macos: ">= :ventura"

  app "Brewtifyer.app"

  zap trash: [
    "~/Library/Application Support/Brewtifyer",
    "~/Library/Logs/Brewtifyer",
  ]
end
CASK
)"

if [[ -n "${OUTPUT_PATH}" ]]; then
  printf '%s\n' "${CASK_CONTENTS}" >"${OUTPUT_PATH}"
  echo "Cask for ${APP_VERSION} written: ${OUTPUT_PATH}"
  echo "SHA-256: ${ARCHIVE_SHA256}"
else
  printf '%s\n' "${CASK_CONTENTS}"
fi
