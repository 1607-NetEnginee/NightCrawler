#!/usr/bin/env bash
# ════════════════════════════════════════════════════════════════════
#  NIGHTCRAWLER v7.0 — one-line installer
#  Author: HnyBadger / Cyberoutcast
#
#  Usage:
#    curl -sSL https://raw.githubusercontent.com/HnyBadger/nightcrawler/main/scripts/install.sh | bash
#
#  Env vars:
#    NC_VERSION   — pin a specific release (default: latest)
#    NC_INSTALL_DIR — installation directory (default: /usr/local/bin)
#    NC_SKIP_VERIFY — set to 1 to skip checksum + signature checks (NOT recommended)
# ════════════════════════════════════════════════════════════════════
set -euo pipefail

NC_REPO="HnyBadger/nightcrawler"
NC_VERSION="${NC_VERSION:-latest}"
NC_INSTALL_DIR="${NC_INSTALL_DIR:-/usr/local/bin}"
NC_SKIP_VERIFY="${NC_SKIP_VERIFY:-0}"

# ── styling ────────────────────────────────────────────────────────
if [ -t 1 ]; then
    BOLD=$'\e[1m'; DIM=$'\e[2m'; RESET=$'\e[0m'
    GREEN=$'\e[32m'; RED=$'\e[31m'; YELLOW=$'\e[33m'; BLUE=$'\e[34m'
else
    BOLD=""; DIM=""; RESET=""; GREEN=""; RED=""; YELLOW=""; BLUE=""
fi

say()  { printf "%s● %s%s\n" "$BLUE" "$1" "$RESET"; }
ok()   { printf "%s✓ %s%s\n" "$GREEN" "$1" "$RESET"; }
warn() { printf "%s! %s%s\n" "$YELLOW" "$1" "$RESET"; }
die()  { printf "%s✗ %s%s\n" "$RED" "$1" "$RESET" >&2; exit 1; }

# ── platform detection ─────────────────────────────────────────────
detect_platform() {
    local os arch
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)   arch=amd64 ;;
        aarch64|arm64)  arch=arm64 ;;
        *)              die "unsupported architecture: $arch" ;;
    esac
    case "$os" in
        linux|darwin)   : ;;
        *)              die "unsupported OS: $os" ;;
    esac
    echo "${os}_${arch}"
}

# ── resolve version ────────────────────────────────────────────────
resolve_version() {
    if [ "$NC_VERSION" = "latest" ]; then
        curl -fsSL "https://api.github.com/repos/${NC_REPO}/releases/latest" \
            | grep -E '"tag_name"' | head -1 | cut -d'"' -f4
    else
        echo "$NC_VERSION"
    fi
}

# ── download and verify ────────────────────────────────────────────
main() {
    say "Detecting platform"
    local platform; platform=$(detect_platform)
    ok "Platform: $platform"

    say "Resolving release version"
    local version; version=$(resolve_version)
    [ -n "$version" ] || die "could not resolve release version"
    ok "Version: $version"

    local stripped="${version#v}"
    local archive="nightcrawler_${stripped}_${platform}.tar.gz"
    local base_url="https://github.com/${NC_REPO}/releases/download/${version}"

    local tmpdir; tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT
    cd "$tmpdir"

    say "Downloading ${archive}"
    curl -fsSL -o "$archive" "${base_url}/${archive}" \
        || die "failed to download $archive"
    ok "Downloaded archive"

    if [ "$NC_SKIP_VERIFY" != "1" ]; then
        say "Downloading checksums"
        curl -fsSL -o checksums.txt "${base_url}/checksums.txt" \
            || die "failed to download checksums.txt"

        say "Verifying SHA-256"
        if command -v sha256sum >/dev/null 2>&1; then
            grep "$archive" checksums.txt | sha256sum -c - >/dev/null \
                || die "checksum verification failed"
        elif command -v shasum >/dev/null 2>&1; then
            grep "$archive" checksums.txt | shasum -a 256 -c - >/dev/null \
                || die "checksum verification failed"
        else
            warn "no sha256sum/shasum binary found, skipping checksum verification"
        fi
        ok "Checksum OK"

        if command -v cosign >/dev/null 2>&1; then
            say "Verifying Sigstore signature"
            curl -fsSL -o checksums.txt.sig "${base_url}/checksums.txt.sig" \
                || warn "no signature available"
            if [ -f checksums.txt.sig ]; then
                cosign verify-blob \
                    --certificate-identity-regexp '.*' \
                    --certificate-oidc-issuer https://token.actions.githubusercontent.com \
                    --signature checksums.txt.sig \
                    checksums.txt >/dev/null 2>&1 \
                    && ok "Signature OK" \
                    || warn "signature verification failed (continuing anyway; pin a version + verify manually if this matters)"
            fi
        else
            warn "cosign not installed — skipping signature verification (see https://github.com/sigstore/cosign)"
        fi
    else
        warn "NC_SKIP_VERIFY=1 — skipping all integrity checks"
    fi

    say "Extracting"
    tar -xzf "$archive"
    [ -f nightcrawler ] || die "binary not found in archive"

    say "Installing to ${NC_INSTALL_DIR}"
    if [ -w "$NC_INSTALL_DIR" ]; then
        install -m 0755 nightcrawler "${NC_INSTALL_DIR}/nightcrawler"
    else
        sudo install -m 0755 nightcrawler "${NC_INSTALL_DIR}/nightcrawler"
    fi

    ok "Installed"
    echo
    "${NC_INSTALL_DIR}/nightcrawler" version || true
    echo
    printf "%s\n" "${BOLD}Next:${RESET}"
    printf "  ${DIM}# Generate a default config${RESET}\n"
    printf "  nightcrawler config init\n\n"
    printf "  ${DIM}# Smoke test against a target you own${RESET}\n"
    printf "  nightcrawler scan -t example.com --profile quick\n\n"
    printf "  ${DIM}# Read the docs${RESET}\n"
    printf "  https://github.com/${NC_REPO}\n\n"
}

main "$@"
