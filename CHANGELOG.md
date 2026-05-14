# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- Initial v7.0 scaffold: orchestrator, plugin registry, plugin DAG scheduler.
- `pkg/api` public Plugin interface.
- `dns` built-in plugin as the canonical reference implementation.
- Five execution profiles: `stealth`, `default`, `aggressive`, `quick`, `compliance`.
- Signature DB at `signatures/` with sensitive-files, ID gov/edu paths, webshell patterns, gambling fingerprints, security-headers checklist — direct port from v6.1 inline lists and validators.
- Multi-stage distroless Dockerfile (target: ≤25 MB image).
- GitHub Actions CI: gofmt, govet, golangci-lint, race-detector test on linux+macos.
- GitHub Actions release: goreleaser with cosign keyless signing, syft SBOM, multi-arch Docker manifest.
- GitHub Actions security: CodeQL, govulncheck, Trivy filesystem scan.
- goreleaser config with cross-compilation (linux/darwin × amd64/arm64), Homebrew tap formula.
- Makefile with `build`, `test`, `lint`, `docker-build`, `release-snapshot` targets.
- Bilingual README (`README.md` EN + `README.id.md` ID).
- Installer script `scripts/install.sh` with SHA-256 + cosign signature verification.

### Notes
- This release is the analysis & redesign milestone. No GA tag yet.
- Reference document: `docs/ANALYSIS_AND_REDESIGN.md` — full reverse engineering audit of v6.1 plus the v7.0 architecture rationale (30 sections, ~2.9k lines).

---

## [v6.1] — 2025-04-28 (legacy Bash)

Preserved for reference. The Bash-era changelog lives in the v6.1 tarball
under `/etc/nightcrawler/nightcrawler.conf` and the inline script header.
