#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
APP_DIR="${REPOSITORY_DIR}/dist/Brewtifyer.app"
RELEASE_DIR="${REPOSITORY_DIR}/dist/release"
RELEASE_ENV_FILE="${BREWTIFYER_RELEASE_ENV_FILE:-${SCRIPT_DIR}/.env}"

# shellcheck source=version.sh
source "${SCRIPT_DIR}/version.sh"

if [[ -f "${RELEASE_ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${RELEASE_ENV_FILE}"
  set +a
fi

SIGNING_IDENTITY="${SIGNING_IDENTITY:-}"
NOTARY_PROFILE="${NOTARY_PROFILE:-}"
SIGNING_TIMESTAMP_URL="${SIGNING_TIMESTAMP_URL:-}"
NOTARY_TIMEOUT="${NOTARY_TIMEOUT:-30m}"
RELEASE_ARCH="${GOARCH:-$(go env GOARCH)}"
RELEASE_VERSION="${APP_VERSION}"
RELEASE_BUILD_NUMBER="${APP_BUILD_NUMBER}"

if [[ -z "${SIGNING_IDENTITY}" ]]; then
  echo "SIGNING_IDENTITY fehlt (Developer ID Application)." >&2
  exit 1
fi

if [[ -z "${NOTARY_PROFILE}" ]]; then
  echo "NOTARY_PROFILE fehlt (Name eines notarytool-Keychain-Profils)." >&2
  exit 1
fi

if [[ ! "${RELEASE_ARCH}" =~ ^[0-9A-Za-z_-]+$ ]]; then
  echo "Ungültige Architektur: ${RELEASE_ARCH}" >&2
  exit 1
fi

if command -v git >/dev/null 2>&1 && git -C "${REPOSITORY_DIR}" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  EXPECTED_RELEASE_TAG="v${RELEASE_VERSION}"
  RELEASE_TAGS="$(git -C "${REPOSITORY_DIR}" tag --points-at HEAD --list 'v*')"
  if [[ -n "${RELEASE_TAGS}" ]] && ! grep -Fx -- "${EXPECTED_RELEASE_TAG}" <<<"${RELEASE_TAGS}" >/dev/null; then
    echo "Release-Tag am aktuellen Commit stimmt nicht mit VERSION überein." >&2
    echo "Erwartet: ${EXPECTED_RELEASE_TAG}" >&2
    echo "Gefunden: ${RELEASE_TAGS}" >&2
    exit 1
  fi
fi

for command_name in codesign dscacheutil ditto go security spctl xcrun; do
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "Benötigtes Programm fehlt: ${command_name}" >&2
    exit 1
  fi
done

if ! security find-identity -v -p codesigning | grep -F -- "${SIGNING_IDENTITY}" >/dev/null; then
  echo "SIGNING_IDENTITY wurde nicht als gültige Codesignatur-Identität gefunden." >&2
  exit 1
fi

if [[ -z "${SIGNING_TIMESTAMP_URL}" ]]; then
  # codesign bietet keinen Schalter an, um den Apple-Timestamp-Dienst nur über
  # IPv4 aufzurufen. Auf manchen Anschlüssen wird der RFC-3161-Request über IPv6
  # zwar gesendet, die Antwort bleibt jedoch aus. Daher wird der aktuelle
  # A-Record dynamisch aufgelöst und nur für diesen Signaturschritt verwendet.
  timestamp_ipv4="$({
    dscacheutil -q host -a name timestamp.apple.com || true
  } | awk '/ip_address:/ && $2 ~ /^[0-9.]+$/ {print $2; exit}')"

  if [[ -z "${timestamp_ipv4}" ]]; then
    echo "Keine IPv4-Adresse für timestamp.apple.com gefunden." >&2
    exit 1
  fi

  SIGNING_TIMESTAMP_URL="http://${timestamp_ipv4}/ts01"
fi

SUBMISSION_ARCHIVE="${RELEASE_DIR}/Brewtifyer-${RELEASE_VERSION}-notarization.zip"
FINAL_ARCHIVE="${RELEASE_DIR}/Brewtifyer-${RELEASE_VERSION}-macos-${RELEASE_ARCH}.zip"
CHECK_DIR="$(mktemp -d /tmp/brewtifyer-release-check.XXXXXX)"

cleanup() {
  rm -rf -- "${CHECK_DIR}"
}
trap cleanup EXIT

verify_bundle_version() {
  local bundle_path="$1"
  local bundle_info_plist="${bundle_path}/Contents/Info.plist"
  local actual_version
  local actual_build_number

  actual_version="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "${bundle_info_plist}")"
  actual_build_number="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleVersion' "${bundle_info_plist}")"
  if [[ "${actual_version}" != "${RELEASE_VERSION}" || "${actual_build_number}" != "${RELEASE_BUILD_NUMBER}" ]]; then
    echo "Versionsdaten stimmen nicht mit VERSION und BUILD_NUMBER überein: ${bundle_path}" >&2
    echo "Erwartet: ${RELEASE_VERSION} (${RELEASE_BUILD_NUMBER})" >&2
    echo "Gefunden: ${actual_version} (${actual_build_number})" >&2
    exit 1
  fi
}

echo "1/8 Brewtifyer-App bauen"
GOARCH="${RELEASE_ARCH}" "${SCRIPT_DIR}/build.sh"
verify_bundle_version "${APP_DIR}"

echo "2/8 Mit Developer ID und Hardened Runtime signieren"
codesign \
  --force \
  --options runtime \
  --timestamp="${SIGNING_TIMESTAMP_URL}" \
  --sign "${SIGNING_IDENTITY}" \
  "${APP_DIR}"

codesign --verify --deep --strict --verbose=4 "${APP_DIR}"

signature_details="$(codesign --display --verbose=4 "${APP_DIR}" 2>&1)"
if ! grep -Eq '^Timestamp=' <<<"${signature_details}"; then
  echo "Die Developer-ID-Signatur enthält keinen sicheren Zeitstempel." >&2
  echo "Prüfe DNS, Firewall und den Zugriff auf Apples Timestamp-Dienst." >&2
  exit 1
fi

mkdir -p "${RELEASE_DIR}"

echo "3/8 Archiv zur Notarisierung erstellen"
rm -f -- "${SUBMISSION_ARCHIVE}"
COPYFILE_DISABLE=1 ditto \
  -c -k \
  --keepParent \
  --norsrc \
  --noextattr \
  "${APP_DIR}" \
  "${SUBMISSION_ARCHIVE}"

echo "4/8 Bei Apple einreichen und Ergebnis abwarten"
xcrun notarytool submit \
  "${SUBMISSION_ARCHIVE}" \
  --keychain-profile "${NOTARY_PROFILE}" \
  --wait \
  --timeout "${NOTARY_TIMEOUT}"

echo "5/8 Notarisierungsticket an die App heften"
xcrun stapler staple "${APP_DIR}"
xcrun stapler validate "${APP_DIR}"

echo "6/8 Signatur und Gatekeeper-Freigabe prüfen"
codesign --verify --deep --strict --verbose=4 "${APP_DIR}"
spctl --assess --type execute --verbose=4 "${APP_DIR}"

echo "7/8 Sauberes Release-ZIP ohne AppleDouble-Dateien erstellen"
rm -f -- "${FINAL_ARCHIVE}"
COPYFILE_DISABLE=1 ditto \
  -c -k \
  --keepParent \
  --norsrc \
  --noextattr \
  "${APP_DIR}" \
  "${FINAL_ARCHIVE}"

echo "8/8 Release-ZIP erneut extrahieren und vollständig prüfen"
ditto -x -k "${FINAL_ARCHIVE}" "${CHECK_DIR}"
verify_bundle_version "${CHECK_DIR}/Brewtifyer.app"
xcrun stapler validate "${CHECK_DIR}/Brewtifyer.app"
codesign --verify --deep --strict --verbose=4 "${CHECK_DIR}/Brewtifyer.app"
spctl --assess --type execute --verbose=4 "${CHECK_DIR}/Brewtifyer.app"

echo "Release ${RELEASE_VERSION} (Build ${RELEASE_BUILD_NUMBER}) erstellt: ${FINAL_ARCHIVE}"
