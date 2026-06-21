#!/usr/bin/env bash
# Install Offprint from GitHub Releases:
#   curl -fsSL "https://raw.githubusercontent.com/gabrielassisxyz/offprint/main/install.sh?$(date +%s)" | bash
set -euo pipefail
umask 022

REPO="${OFFPRINT_REPO:-gabrielassisxyz/offprint}"
DEST="${OFFPRINT_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${OFFPRINT_VERSION:-}"
OFFLINE=""
FORCE=0
QUIET=0
TMP=""
LOCK="${TMPDIR:-/tmp}/offprint-install-${UID:-user}.lock"

info() { [[ "$QUIET" == 1 ]] || printf '%s\n' "-> $*"; }
die() { printf 'offprint installer: %s\n' "$*" >&2; exit 1; }
cleanup() { [[ -z "$TMP" ]] || rm -rf "$TMP"; rmdir "$LOCK" 2>/dev/null || true; }
trap cleanup EXIT

usage() {
  cat <<'EOF'
Usage: install.sh [options]
  --version VERSION   Install a specific version (default: latest)
  --dest DIR          Installation directory (default: ~/.local/bin)
  --offline ARCHIVE   Install a previously downloaded release archive
  --force             Reinstall an existing version
  --quiet             Suppress progress output
  -h, --help          Show this help

Environment: OFFPRINT_REPO, OFFPRINT_INSTALL_DIR, OFFPRINT_VERSION, HTTPS_PROXY.
Uninstall: rm "$(command -v offprint)"
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) [[ $# -ge 2 ]] || die "--version requires a value"; VERSION="$2"; shift 2 ;;
    --dest) [[ $# -ge 2 ]] || die "--dest requires a directory"; DEST="$2"; shift 2 ;;
    --offline) [[ $# -ge 2 ]] || die "--offline requires an archive"; OFFLINE="$2"; shift 2 ;;
    --force) FORCE=1; shift ;;
    --quiet) QUIET=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) die "unknown option: $1 (run with --help)" ;;
  esac
done

mkdir "$LOCK" 2>/dev/null || die "another installer is already running ($LOCK)"
TMP="$(mktemp -d)"
mkdir -p "$DEST"
[[ -w "$DEST" ]] || die "installation directory is not writable: $DEST"

case "$(uname -s)" in
  Linux) OS=linux ;;
  Darwin) OS=darwin ;;
  *) die "unsupported operating system: $(uname -s)" ;;
esac
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) die "unsupported architecture: $(uname -m)" ;;
esac

if [[ -n "$OFFLINE" ]]; then
  [[ -f "$OFFLINE" ]] || die "offline archive not found: $OFFLINE"
  ARCHIVE="$OFFLINE"
else
  command -v curl >/dev/null || die "curl is required"
  if [[ -z "$VERSION" ]]; then
    VERSION="$(curl -fsSL --connect-timeout 10 "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p' | head -1)"
    [[ -n "$VERSION" ]] || die "could not determine the latest release"
  fi
  VERSION="${VERSION#v}"
  NAME="offprint_${VERSION}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/$REPO/releases/download/v$VERSION/$NAME"
  ARCHIVE="$TMP/$NAME"
  info "Downloading Offprint $VERSION for $OS/$ARCH"
  curl -fL --connect-timeout 10 --retry 3 "$URL" -o "$ARCHIVE"
  curl -fL --connect-timeout 10 --retry 3 "https://github.com/$REPO/releases/download/v$VERSION/checksums.txt" -o "$TMP/checksums.txt"
  EXPECTED="$(awk -v name="$NAME" '$2 == name {print $1}' "$TMP/checksums.txt")"
  [[ -n "$EXPECTED" ]] || die "release checksum does not list $NAME"
  if command -v sha256sum >/dev/null; then
    ACTUAL="$(sha256sum "$ARCHIVE" | awk '{print $1}')"
  elif command -v shasum >/dev/null; then
    ACTUAL="$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')"
  else
    die "sha256sum or shasum is required to verify the download"
  fi
  [[ "$ACTUAL" == "$EXPECTED" ]] || die "checksum verification failed"
fi

tar -xzf "$ARCHIVE" -C "$TMP"
[[ -x "$TMP/offprint" ]] || die "archive does not contain the offprint binary"
if [[ -x "$DEST/offprint" && "$FORCE" != 1 ]]; then
  CURRENT="$($DEST/offprint version 2>/dev/null || true)"
  [[ "$CURRENT" != *"${VERSION:-}"* ]] || { info "Offprint ${VERSION:-current} is already installed"; exit 0; }
fi
install -m 0755 "$TMP/offprint" "$DEST/offprint"
info "Installed $DEST/offprint"
case ":$PATH:" in
  *:"$DEST":*) ;;
  *) info "Add $DEST to PATH to run offprint from any directory" ;;
esac
