# Contributing to NIGHTCRAWLER

First — thank you. The project benefits enormously from outside eyes, especially on:

- New plugins (recon, content checks, integrations)
- Signature pack contributions (especially for Indonesian gov/edu/private-sector targets)
- Documentation improvements and translations
- Bug reports with reliable reproduction steps
- Performance improvements with benchmarks

Please read this short guide before opening a PR — it saves time on both sides.

---

## Before you start

1. **Open an issue first** for anything larger than a typo fix. Tagging the maintainers and getting feedback on direction prevents wasted work.
2. **Search existing issues and PRs** — your idea may already be in flight.
3. **Be respectful.** See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

---

## Development setup

```bash
git clone https://github.com/HnyBadger/nightcrawler
cd nightcrawler

# install golangci-lint (one-time)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# build, test, lint
make build
make test
make lint
```

You need Go 1.22+ and a recent `golangci-lint`. Optional: `goreleaser`, `cosign`, `syft`, `docker buildx`.

---

## Commit message convention

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(plugin/paths): port v6.1 validation rules for .env detection

Fixes #123
```

Allowed types: `feat`, `fix`, `perf`, `refactor`, `test`, `docs`, `chore`, `ci`, `sec`.

Use `!` after the type for breaking changes, e.g. `feat(api)!: rename Emitter to Emit`.

The release changelog is generated from commit subjects, so a good subject line matters.

---

## Pull request checklist

Before requesting review:

- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] New code has tests (target: ≥75% on `internal/core`, `internal/validator`, `internal/plugins/*`)
- [ ] Public API changes are documented in `pkg/api/*.go` godoc and in the relevant `docs/` file
- [ ] CHANGELOG entry added under `## [Unreleased]`
- [ ] Commit messages follow the convention above
- [ ] For breaking changes: an [ADR](docs/adr/) is included

PRs against `main` require:

- Green CI
- One approving review from a maintainer (CODEOWNERS auto-assigns)
- All review comments resolved or explicitly deferred

---

## Plugin contributions

A new plugin needs:

1. **An issue first** with a "Plugin proposal" template. We discuss scope, manifest fields, validation rules, and dependencies before code.
2. **An implementation** under `internal/plugins/<name>/`:
   - `plugin.go` — the `api.Plugin` implementation
   - `plugin.yaml` — the manifest (for `nightcrawler plugin info`)
   - signature data under `signatures/<category>/<name>.yaml` if applicable
3. **A blank import** added to `cmd/nightcrawler/builtins.go`
4. **Tests:**
   - Unit tests covering the validation logic
   - Integration test using a recorded HTTP fixture under `test/fixtures/<plugin>/`
5. **Documentation:**
   - `docs/PLUGINS.md` table entry
   - Examples in the plugin's package godoc

Use `internal/plugins/dns/` as the reference implementation.

---

## Signature pack contributions

This is the easiest way to contribute and arguably the most valuable. To add a sensitive path / webshell pattern / gambling keyword:

1. Open a PR adding entries to the appropriate file under `signatures/`.
2. **Provide validators.** Bare path probes that just check for `200 OK` are rejected — they produce false positives. Use `contains`, `any_contains`, or `matches_regex` to fingerprint the actual content.
3. **Provide a mitigation** in both `id` and `en` if you can; English-only is acceptable for now and we'll translate.
4. **Note the source** — was this seen in the wild on a real engagement? CVE-backed? Reference helps reviewers.

Signature PRs do not require Go knowledge.

---

## Code style

- `gofmt`-formatted (enforced by CI)
- `goimports` group order: stdlib → external → `github.com/HnyBadger/nightcrawler/...`
- Errors wrapped with `fmt.Errorf("context: %w", err)` — no naked propagation
- Exported identifiers documented with godoc-style comments
- Avoid `init()` for anything other than plugin registration
- No global mutable state (use the orchestrator's `ScanContext`)
- Channel-based concurrency preferred over mutexes for new code; mutexes acceptable inside hot caches

---

## Questions?

- General discussion: GitHub Discussions
- Quick Q&A: `@HnyBadger` on the issue thread
- Security: see [SECURITY.md](SECURITY.md)

Thanks again — and welcome.
