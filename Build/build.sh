#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
APP_DIR="${REPOSITORY_DIR}/dist/Brewtifyer.app"
CONTENTS_DIR="${APP_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"
INFO_PLIST="${CONTENTS_DIR}/Info.plist"
BUILDINFO_PACKAGE="github.com/KevinCFechtel/Brewtifyer/internal/buildinfo"

# macOS 13 is the floor of the Go toolchain pinned in go.mod. Keep this in sync
# with LSMinimumSystemVersion in Build/Info.plist.
DEPLOYMENT_TARGET="13.0"

# Releases ship a universal binary so that Intel Macs are covered too.
# Set BREWTIFYER_ARCHS=arm64 for a faster single-architecture development build.
read -r -a TARGET_ARCHS <<<"${BREWTIFYER_ARCHS:-arm64 amd64}"

# shellcheck source=version.sh
source "${SCRIPT_DIR}/version.sh"

LDFLAGS=(
  -s -w
  "-X=${BUILDINFO_PACKAGE}.Version=${APP_VERSION}"
  "-X=${BUILDINFO_PACKAGE}.Build=${APP_BUILD_NUMBER}"
  "-X=${BUILDINFO_PACKAGE}.Commit=${APP_COMMIT}"
)

if [[ ${#TARGET_ARCHS[@]} -eq 0 ]]; then
  echo "BREWTIFYER_ARCHS is empty; expected at least one of arm64 or amd64." >&2
  exit 1
fi

if [[ "${APP_DIR}" != "${REPOSITORY_DIR}/dist/Brewtifyer.app" ]]; then
  echo "Unexpected app path: ${APP_DIR}" >&2
  exit 1
fi

rm -rf -- "${APP_DIR}"
mkdir -p "${MACOS_DIR}" "${RESOURCES_DIR}"
install -m 0644 "${SCRIPT_DIR}/Info.plist" "${INFO_PLIST}"
install -m 0644 "${SCRIPT_DIR}/AppIcon.icns" "${RESOURCES_DIR}/AppIcon.icns"
install -m 0644 "${SCRIPT_DIR}/Assets.car" "${RESOURCES_DIR}/Assets.car"

/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString ${APP_VERSION}" "${INFO_PLIST}"
/usr/libexec/PlistBuddy -c "Set :CFBundleVersion ${APP_BUILD_NUMBER}" "${INFO_PLIST}"

BUILD_DIR="$(mktemp -d /tmp/brewtifyer-build.XXXXXX)"
cleanup() {
  rm -rf -- "${BUILD_DIR}"
}
trap cleanup EXIT

cd "${REPOSITORY_DIR}"

# clang_arch translates a Go architecture into the clang -arch value that cgo
# needs when cross-compiling the Objective-C bridges.
clang_arch() {
  case "$1" in
    arm64) printf 'arm64' ;;
    amd64) printf 'x86_64' ;;
    *)
      echo "Unsupported architecture: $1 (expected arm64 or amd64)" >&2
      return 1
      ;;
  esac
}

SLICES=()
for arch in "${TARGET_ARCHS[@]}"; do
  clang_target="$(clang_arch "${arch}")"
  slice="${BUILD_DIR}/Brewtifyer-${arch}"
  echo "Building ${arch} slice"
  MACOSX_DEPLOYMENT_TARGET="${DEPLOYMENT_TARGET}" \
    CC="clang -arch ${clang_target}" \
    CXX="clang++ -arch ${clang_target}" \
    CGO_CFLAGS="${CGO_CFLAGS:-} -mmacosx-version-min=${DEPLOYMENT_TARGET}" \
    CGO_LDFLAGS="${CGO_LDFLAGS:-} -mmacosx-version-min=${DEPLOYMENT_TARGET}" \
    CGO_ENABLED=1 GOOS=darwin GOARCH="${arch}" \
    go build -buildvcs=false -o "${slice}" -ldflags "${LDFLAGS[*]}" ./cmd/brewtifyer
  SLICES+=("${slice}")
done

if [[ ${#SLICES[@]} -eq 1 ]]; then
  install -m 0755 "${SLICES[0]}" "${MACOS_DIR}/Brewtifyer"
else
  lipo -create -output "${MACOS_DIR}/Brewtifyer" "${SLICES[@]}"
  chmod 0755 "${MACOS_DIR}/Brewtifyer"
fi

# Ad-hoc signature for local development. Releases are re-signed with a
# Developer ID by Build/release.sh. --deep is deliberately not used: Apple
# discourages it for signing, and the bundle has no nested code anyway.
if command -v codesign >/dev/null 2>&1; then
  codesign --force --sign - "${APP_DIR}"
fi

BUILT_VERSION="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "${INFO_PLIST}")"
BUILT_NUMBER="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleVersion' "${INFO_PLIST}")"
if [[ "${BUILT_VERSION}" != "${APP_VERSION}" || "${BUILT_NUMBER}" != "${APP_BUILD_NUMBER}" ]]; then
  echo "Version metadata in the app bundle does not match VERSION and BUILD_NUMBER." >&2
  exit 1
fi

BUILT_ARCHS="$(lipo -archs "${MACOS_DIR}/Brewtifyer")"
echo "Brewtifyer ${APP_VERSION} (build ${APP_BUILD_NUMBER}) created: ${APP_DIR}"
echo "Architectures: ${BUILT_ARCHS}"
