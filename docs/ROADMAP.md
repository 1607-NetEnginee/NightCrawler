# NIGHTCRAWLER Roadmap

This is the public roadmap. Items are tentative, dates are not commitments. See `docs/ANALYSIS_AND_REDESIGN.md` §7 and §30 for full rationale.

---

## v7.0 — GA (target: Q3 2026)

**Theme:** Core rewrite + paritas dengan v6.1 + concurrency unlock.

- Single Go binary, modular plugin architecture
- 17 built-in plugins ported from v6.1
- YAML signature database (paths, webshells, gambling, headers)
- 3-layer concurrency (target / plugin DAG / per-plugin probes)
- NDJSON canonical output + HTML + SARIF + TXT
- Bilingual mitigation (id / en)
- Multi-arch Docker image (linux/amd64, linux/arm64)
- Cosign-signed releases, SBOM
- 5 execution profiles
- `nightcrawler config import-v6` migration path

---

## v7.1 — TUI + Distributed (target: ~3 months after GA)

- Bubble Tea TUI dengan live worker monitor, queue, log panel
- gRPC agent mode untuk distributed scanning
- PostgreSQL backend untuk scan history
- Adaptive rate limiter (sudah ada di v7.0 sebagai stub; full implementation v7.1)
- Resume / checkpoint untuk scan yang interrupted
- `age` encrypted output

---

## v7.2 — Web Dashboard

- Read-only operator UI (htmx + templ, no SPA)
- WebSocket live log streaming
- Asset inventory cross-target
- Attack surface graph visualization
- Export PDF via chromedp

---

## v7.3 — Modern Protocols

- HTTP/3 support (quic-go)
- JA3/JA4 fingerprinting (utls)
- Headless browser crawling (chromedp)
- Screenshot engine + visual diff

---

## v7.4 — Notifier ecosystem

- Telegram, Slack, Discord, Mattermost
- Webhook dengan templating
- SIEM-specific exporters (Splunk HEC, Datadog, Sentry)

---

## v7.5 — AI Enrichment (P2)

- Local LLM via Ollama untuk mitigation copywriting
- False-positive triage assistant
- `nightcrawler ask` Q&A mode over scan results
- Privacy-first: local-default, cloud opt-in dengan secret redaction

---

## v7.6 — Plugin Marketplace (P2)

- Remote plugin install via `nightcrawler plugin install ...`
- Plugin signing (ed25519)
- Community signature pack registry
- Plugin sandboxing dengan capability minimization

---

## v8.0 — TBD

Reserved for breaking changes informed by lessons from v7.x. No specifics committed.

---

## How priorities are set

P0 / P1 / P2 buckets are documented in [`ANALYSIS_AND_REDESIGN.md` §7](ANALYSIS_AND_REDESIGN.md#7-prioritas-refactor-p0--p1--p2). Bugs always preempt features. Security fixes preempt everything.

## Want to influence this?

Open a "Feature request" issue. Include the use case, not just the feature. Community traction (👍 reactions, related PRs) reorders items within a release window.
