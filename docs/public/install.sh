#!/usr/bin/env sh
set -eu
(set -o pipefail >/dev/null 2>&1) && set -o pipefail || true

REPO="stackfox-labs/luumen"
VERSION="${LUU_VERSION:-latest}"
INSTALL_DIR="${LUU_INSTALL_DIR:-${HOME}/.local/bin}"
ADD_TO_PATH="${LUU_ADD_TO_PATH:-0}"
DRY_RUN="${LUU_INSTALL_DRY_RUN:-0}"
ALLOW_PRE_RELEASE="${LUU_PRE_RELEASE:-0}"

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'install.sh: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

print_help() {
  cat <<'EOF'
Luumen installer (Linux/macOS)

Usage:
  sh install.sh [--pre-release]

Flags:
  --pre-release  Allow installing pre-release tags.
  -h, --help     Show this help.

Env overrides:
  LUU_VERSION            Version to install (default: latest)
  LUU_INSTALL_DIR        Install directory (default: ~/.local/bin)
  LUU_INSTALL_DRY_RUN    1/true to skip writing files
  LUU_ADD_TO_PATH        1/true to print shell profile hint
  LUU_PRE_RELEASE        1/true to allow pre-release tags
EOF
}

is_truthy() {
  case "$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')" in
    1|true|yes|on) return 0 ;;
    *) return 1 ;;
  esac
}

normalize_arch() {
  machine="$1"
  case "$machine" in
    x86_64|amd64) printf 'amd64' ;;
    arm64|aarch64) printf 'arm64' ;;
    *) fail "unsupported architecture: $machine" ;;
  esac
}

normalize_os() {
  sysname="$1"
  case "$sysname" in
    Linux) printf 'linux' ;;
    Darwin) printf 'darwin' ;;
    *) fail "unsupported operating system: $sysname" ;;
  esac
}

hash_file() {
  path="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$path" | awk '{print tolower($1)}'
    return
  fi

  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$path" | awk '{print tolower($1)}'
    return
  fi

  fail "no SHA-256 tool found (need sha256sum or shasum)"
}

extract_expected_hash() {
  checksum_file="$1"
  artifact_name="$2"

  awk -v name="$artifact_name" '
    {
      if ($0 ~ /^[[:space:]]*[A-Fa-f0-9]{64}[[:space:]]+\*?[^[:space:]]+[[:space:]]*$/) {
        file=$2
        gsub(/^\*/, "", file)
        if (file == name) {
          print tolower($1)
          exit
        }
      }
      if ($0 ~ /^SHA256 \([^)]*\) = [A-Fa-f0-9]{64}[[:space:]]*$/) {
        line=$0
        sub(/^SHA256 \(/, "", line)
        split(line, parts, /\) = /)
        if (parts[1] == name) {
          print tolower(parts[2])
          exit
        }
      }
    }
  ' "$checksum_file"
}

url_encode_version() {
  value="$1"
  case "$value" in
    *[!A-Za-z0-9._-]*)
      fail "invalid version string: $value"
      ;;
    *)
      printf '%s' "$value"
      ;;
  esac
}

extract_tag_name() {
  awk -F '"' '/"tag_name"[[:space:]]*:/ { print $4; exit }'
}

resolve_release_tag() {
  requested="$1"

  if [ "$requested" != "latest" ]; then
    resolved_tag="$(url_encode_version "$requested")"
    if ! is_truthy "$ALLOW_PRE_RELEASE" && printf '%s' "$resolved_tag" | grep -q -- '-'; then
      fail "pre-release version '$resolved_tag' requested but --pre-release was not set"
    fi
    printf '%s' "$resolved_tag"
    return
  fi

  latest_api="https://api.github.com/repos/${REPO}/releases/latest"
  latest_json="$(curl --proto '=https' --tlsv1.2 --silent --show-error --location "$latest_api" 2>/dev/null || true)"
  resolved_tag="$(printf '%s' "$latest_json" | extract_tag_name)"
  if [ -n "$resolved_tag" ]; then
    printf '%s' "$resolved_tag"
    return
  fi

  if ! is_truthy "$ALLOW_PRE_RELEASE"; then
    fail "no stable release found; rerun with --pre-release to allow installing the newest pre-release"
  fi

  log "No stable release found; falling back to newest release including pre-releases."
  list_api="https://api.github.com/repos/${REPO}/releases?per_page=1"
  list_json="$(curl --proto '=https' --tlsv1.2 --fail --show-error --location "$list_api")" || fail "failed to query GitHub releases API"
  resolved_tag="$(printf '%s' "$list_json" | extract_tag_name)"
  [ -n "$resolved_tag" ] || fail "could not resolve latest release tag from GitHub API"
  printf '%s' "$resolved_tag"
}

append_path_hint() {
  if printf ':%s:' "$PATH" | grep -F ":$INSTALL_DIR:" >/dev/null 2>&1; then
    return
  fi

  if is_truthy "$ADD_TO_PATH"; then
    log "Install directory is not in PATH."
    log "Add this line to your shell profile:"
    log "  export PATH=\"$INSTALL_DIR:\$PATH\""
  else
    log "Install directory is not in PATH."
    log "Run this to use luu now:"
    log "  export PATH=\"$INSTALL_DIR:\$PATH\""
  fi
}

need_cmd curl
need_cmd tar
need_cmd awk
need_cmd uname
need_cmd mktemp
need_cmd find

while [ "$#" -gt 0 ]; do
  case "$1" in
    --pre-release)
      ALLOW_PRE_RELEASE=1
      ;;
    -h|--help)
      print_help
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
  shift
done

OS="$(normalize_os "$(uname -s)")"
ARCH="$(normalize_arch "$(uname -m)")"
ARTIFACT="luu-${OS}-${ARCH}.tar.gz"
CHECKSUMS_FILE="checksums.txt"

RESOLVED_VERSION="$(resolve_release_tag "$VERSION")"
BASE_URL="https://github.com/${REPO}/releases/download/${RESOLVED_VERSION}"

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/luu-install.XXXXXX")"
ARCHIVE_PATH="$TMP_DIR/$ARTIFACT"
CHECKSUMS_PATH="$TMP_DIR/$CHECKSUMS_FILE"
EXTRACT_DIR="$TMP_DIR/extract"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

log "Preparing Luumen install..."
log "  Repository: https://github.com/$REPO"
log "  Version:    $VERSION"
log "  Pre-rel:    $ALLOW_PRE_RELEASE"
if [ "$VERSION" != "$RESOLVED_VERSION" ]; then
  log "  Resolved:   $RESOLVED_VERSION"
fi
log "  Platform:   $OS/$ARCH"
log "  Install to: $INSTALL_DIR"
log "Downloading release metadata..."

curl --proto '=https' --tlsv1.2 --fail --show-error --location \
  "$BASE_URL/$CHECKSUMS_FILE" \
  --output "$CHECKSUMS_PATH"

curl --proto '=https' --tlsv1.2 --fail --show-error --location \
  "$BASE_URL/$ARTIFACT" \
  --output "$ARCHIVE_PATH"

EXPECTED_HASH="$(extract_expected_hash "$CHECKSUMS_PATH" "$ARTIFACT")"
[ -n "$EXPECTED_HASH" ] || fail "no checksum found for $ARTIFACT in $CHECKSUMS_FILE"

ACTUAL_HASH="$(hash_file "$ARCHIVE_PATH")"
[ "$EXPECTED_HASH" = "$ACTUAL_HASH" ] || fail "checksum verification failed for $ARTIFACT"

mkdir -p "$EXTRACT_DIR"
tar -xzf "$ARCHIVE_PATH" -C "$EXTRACT_DIR"

BIN_PATH="$(find "$EXTRACT_DIR" -type f -name luu | head -n 1 || true)"
[ -n "$BIN_PATH" ] || fail "could not find luu binary in extracted archive"

if is_truthy "$DRY_RUN"; then
  log "Dry run successful."
  log "Would install: $BIN_PATH -> $INSTALL_DIR/luu"
  exit 0
fi

mkdir -p "$INSTALL_DIR"
TMP_BIN="$INSTALL_DIR/.luu.new.$$"
cp "$BIN_PATH" "$TMP_BIN"
chmod 755 "$TMP_BIN"
mv -f "$TMP_BIN" "$INSTALL_DIR/luu"

log "Installed luu to $INSTALL_DIR/luu"
append_path_hint
log "Run 'luu --help' to verify installation."
