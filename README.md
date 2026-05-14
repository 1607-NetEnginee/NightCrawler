# NIGHTCRAWLER v7.0

> Offensive Security Framework — *"ignored, but critical"*
>
> by **HnyBadger** / **Cyberoutcast**

[![ci](https://github.com/1607-NetEnginee/NightCrawler/actions/workflows/ci.yml/badge.svg)](https://github.com/1607-NetEnginee/NightCrawler/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/1607-NetEnginee/NightCrawler)](https://github.com/1607-NetEnginee/NightCrawler/releases)
[![codeql](https://github.com/1607-NetEnginee/NightCrawler/actions/workflows/codeql.yml/badge.svg)](https://github.com/1607-NetEnginee/NightCrawler/actions/workflows/codeql.yml)
[![go report](https://goreportcard.com/badge/github.com/1607-NetEnginee/NightCrawler)](https://goreportcard.com/report/github.com/1607-NetEnginee/NightCrawler)
[![license](https://img.shields.io/github/license/1607-NetEnginee/NightCrawler)](LICENSE)

NIGHTCRAWLER is a modular, plugin-driven offensive security framework, written in Go. It is the production-grade successor to the v6.1 Bash framework — a complete re-architecture that preserves five iterations of false-positive engineering, the Indonesian government/education path knowledge, and the bilingual mitigation guidance that defined v6.x, while removing the monolithic, sequential, subprocess-heavy machinery that limited it.

Read the design document at [`docs/ANALYSIS_AND_REDESIGN.md`](docs/ANALYSIS_AND_REDESIGN.md) for the full reverse-engineering audit of v6.1 and the architecture rationale.

> **Indonesian readers:** see [README.id.md](README.id.md).

---

## Highlights

- **Single static binary.** No runtime dependencies. ≤ 30 MB.
- **3-layer concurrency.** Target-level, plugin-DAG-level, and per-plugin probe-level worker pools.
- **17 built-in plugins** covering DNS, TLS, headers, ports, sensitive files, CMS fingerprint, methods, CORS, gambling injection, open redirect, info disclosure, timing, XSS, SQLi, webshell, crt.sh passive recon.
- **Validation layer preserved.** Catch-all detection, tech-aware path filtering, smart IP differential, gambling density check — all five iterations of v6.1 false-positive engineering migrated to data-driven YAML signature packs.
- **NDJSON-first output.** Canonical event stream pipes cleanly into SIEM (Vector, Fluent Bit, Logstash). HTML, SARIF, TXT, and PDF derive from NDJSON.
- **Bilingual mitigations (id / en).** Every finding ships remediation guidance in both languages.
- **Profile-driven.** `stealth`, `default`, `aggressive`, `quick`, `compliance` — switch with a flag.
- **Cloud-native.** Distroless Docker image. K8s Helm chart. Air-gapped deployment supported.
- **Defensive posture.** Non-root by default. TLS verification on by default. Adaptive rate limiting that honors `Retry-After`.

---

## Install

### Single-line installer

```bash
curl -sSL https://raw.githubusercontent.com/1607-NetEnginee/NightCrawler/main/scripts/install.sh | bash
```

The installer verifies the SHA-256 checksum and the Sigstore signature of the release artifact before installing.

### Manual download

Grab the appropriate archive from the [releases page](https://github.com/1607-NetEnginee/NightCrawler/releases) and extract it. Binary is portable; copy it anywhere on your `$PATH`.

### Go install

```bash
go install github.com/1607-NetEnginee/NightCrawler/cmd/nightcrawler@latest
```

### Docker

```bash
docker pull ghcr.io/1607-netengineee/nightcrawler:latest
docker run --rm ghcr.io/1607-netengineee/nightcrawler:latest scan -t example.com
```

### Homebrew (macOS / Linux)

```bash
brew install hnybadger/tap/nightcrawler
```

### Build from source

```bash
git clone https://github.com/1607-NetEnginee/NightCrawler
cd nightcrawler
make build
./bin/nightcrawler --help
```

---

## Quickstart

```bash
# Smoke test
nightcrawler scan -t example.com --profile quick

# Default full scan
nightcrawler scan -t corp.example.com --client "Acme Corp"

# Multi-target from file, stealth profile
nightcrawler scan --targets-file scope.txt --profile stealth -o ./reports

# Pick specific plugins
nightcrawler scan -t example.com --plugins dns,tls,headers,paths

# CI gate: fail build on any HIGH or CRITICAL finding
nightcrawler scan -t $TARGET --profile compliance \
  --format ndjson,sarif --fail-on high -o ./out
```

Detailed CLI reference: [`docs/USAGE.md`](docs/USAGE.md).

---

## What v7.0 is, and is not

**v7.0 is** a production-grade reconnaissance and posture-assessment framework intended for authorized engagements. The defensive bias of v6.1 is preserved: the tool flags compromise indicators (webshells, gambling injection, exposed `.env`) rather than helping attackers exploit them.

**v7.0 is not** an exploit framework, a vulnerability-scanner-of-everything, or a black-box pentest robot. It pairs with — and does not replace — purpose-built tools like Burp Suite, Metasploit, or Nuclei. Where coverage overlaps (e.g. `nikto` integration), the third-party tool is run as an opt-in plugin, not the default.

---

## Architecture at a glance

```
   CLI / TUI
      ↓
   Orchestrator ── Plugin Registry ── Validator Layer ── Signature DB (YAML)
      ↓                                    ↑
   3-layer worker pool ── HTTP / DNS / TLS engines (native, pooled)
      ↓
   Event bus (channel) ── Aggregator ── NDJSON
                                          ↓
                              HTML │ SARIF │ TXT │ PDF │ Webhook │ Syslog
```

Full architecture, plugin DAG semantics, distributed-mode topology: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

---

## Migrating from v6.x

```bash
# 1. Install v7.0 alongside v6.1 (different binary name)
# 2. Translate your existing v6.1 config
nightcrawler config import-v6 /etc/nightcrawler/nightcrawler.conf \
  --output ~/.config/nightcrawler/config.yaml

# 3. Validate
nightcrawler config validate

# 4. Dry-run to confirm scope
nightcrawler scan --targets-file scope.txt --dry-run

# 5. Run for real
nightcrawler scan --targets-file scope.txt
```

Full migration guide: [`docs/MIGRATION_V6_TO_V7.md`](docs/MIGRATION_V6_TO_V7.md).

---

## Plugin development

```go
package myplugin

import (
    "context"

    "github.com/1607-NetEnginee/NightCrawler/internal/plugin"
    "github.com/1607-NetEnginee/NightCrawler/pkg/api"
)

type Plugin struct{}

func init() { plugin.Register(&Plugin{}) }

func (p *Plugin) Manifest() api.Manifest {
    return api.Manifest{
        Name:        "my-plugin",
        Version:     "0.1.0",
        Author:      "You",
        Description: "Does something useful.",
        Category:    api.CategoryRecon,
    }
}

func (p *Plugin) Init(_ context.Context, _ api.Deps) error { return nil }

func (p *Plugin) Run(ctx context.Context, target api.Target, emit api.Emitter) error {
    emit(api.Finding{
        Plugin: "my-plugin",
        Level:  api.SeverityInfo,
        Title:  "Hello from a plugin",
    })
    return nil
}
```

See [`docs/PLUGIN_DEVELOPMENT.md`](docs/PLUGIN_DEVELOPMENT.md) for the full plugin author guide. An end-to-end reference is at [`internal/plugins/dns/`](internal/plugins/dns/).

---

## Security

NIGHTCRAWLER scans the systems you run it against. Run it only against assets you own or have explicit written authorization to test. The authors and maintainers accept no responsibility for misuse.

Report security issues per [`SECURITY.md`](SECURITY.md). Please do not file them as public GitHub issues.

---

## Documentation

| Document | What is in it |
|---|---|
| [`docs/ANALYSIS_AND_REDESIGN.md`](docs/ANALYSIS_AND_REDESIGN.md) | Reverse engineering of v6.1 + v7.0 architecture rationale (2.9k lines) |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Component diagrams, lifecycle, DAG semantics |
| [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md) | Full YAML reference |
| [`docs/USAGE.md`](docs/USAGE.md) | CLI reference, examples, recipes |
| [`docs/PLUGIN_DEVELOPMENT.md`](docs/PLUGIN_DEVELOPMENT.md) | Plugin author guide |
| [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) | Docker, K8s, systemd, air-gapped |
| [`docs/MIGRATION_V6_TO_V7.md`](docs/MIGRATION_V6_TO_V7.md) | v6.x → v7.0 step-by-step |
| [`docs/SECURITY.md`](docs/SECURITY.md) | Threat model, hardening guide |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | What's coming after v7.0 GA |

---

## Contributing

Pull requests are welcome. Read [`CONTRIBUTING.md`](CONTRIBUTING.md) first — in particular the commit-message convention and the plugin proposal process.

---

## License

Apache 2.0. See [`LICENSE`](LICENSE).

---

*Built with care by HnyBadger / Cyberoutcast. If NIGHTCRAWLER helped you, drop a star — it's free, and it helps the project's visibility.*
