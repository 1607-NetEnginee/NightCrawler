# Arsitektur NIGHTCRAWLER v7.0

> Dokumen ini menjelaskan arsitektur internal dan desain sistem NIGHTCRAWLER v7.0.

---

## Overview

NIGHTCRAWLER v7.0 dibangun ulang dari nol dalam Go dengan fokus pada:

- **Concurrency 3 lapis** — target-level, plugin DAG-level, per-plugin probe-level
- **Plugin DAG** — dependency antar plugin dikelola sebagai Directed Acyclic Graph
- **Single binary** — tidak ada dependency runtime, ≤ 30 MB
- **NDJSON canonical stream** — semua output diderive dari event stream tunggal

---

## Komponen Utama

```
nightcrawler/
├── cmd/nightcrawler/     # Entry point, CLI setup
├── internal/
│   ├── cli/              # Subcommand: scan, plugin, config, version
│   ├── core/             # Orchestrator, DAG engine, ScanRequest/Result
│   └── plugins/          # Implementasi 17 plugin bawaan
├── pkg/api/              # Public API untuk plugin eksternal
├── configs/              # Contoh konfigurasi
└── signatures/           # Database signature (paths, patterns)
```

---

## Alur Eksekusi Scan

```
CLI (scan command)
  └─► ScanRequest.Validate()
        └─► Orchestrator.Run()
              ├─► Load config & profile
              ├─► Resolve plugin DAG
              ├─► Worker pool per target
              │     └─► Worker pool per plugin
              │           └─► Probe pool per plugin
              └─► Report writer (NDJSON → HTML/SARIF/TXT)
```

---

## Plugin DAG

Plugin dijalankan berdasarkan dependency graph, bukan urutan linear:

- `dns` → tidak ada dependency
- `tls` → tidak ada dependency
- `cms` → depends on `dns`, `headers`
- `paths` → depends on `cms` (untuk filter berbasis tech profile)
- `xss`, `sqli` → depends on `paths`

Artinya plugin yang tidak saling bergantung dijalankan **paralel**, plugin yang bergantung menunggu hasilnya.

---

## Format Output NDJSON

Setiap event ditulis sebagai satu baris JSON:

```json
{"schema":"nightcrawler.io/v1/finding","scan_id":"s-xxx","ts":"...","plugin":"headers","severity":"high","title":"Missing HSTS","target":"example.com","evidence":"...","remediation_id":"hsts-001"}
```

Format HTML, SARIF, dan TXT semua diderive dari stream NDJSON ini.

---

*Dokumentasi lengkap arsitektur ada di [`docs/ANALYSIS_AND_REDESIGN.md`](ANALYSIS_AND_REDESIGN.md).*

*Dokumentasi ini bagian dari [NightCrawler v7.0](https://github.com/1607-NetEnginee/NightCrawler) oleh 1607-NetEnginee / Cyberoutcast.*
