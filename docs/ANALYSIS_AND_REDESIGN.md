# NIGHTCRAWLER v7.0 — Analisa Mendalam v6.1 & Cetak Biru Redesign

> **Author:** 1607-NetEnginee  
> **Project:** NIGHTCRAWLER v7.0 (Cyberoutcast)  
> **Dokumen:** Reverse Engineering Report + Migrasi Bash → Go + Arsitektur Enterprise

---

## Executive Summary

NIGHTCRAWLER v6.1 adalah Bash script monolitik 3.247 baris yang berfungsi sebagai **defensive security orchestrator** — membungkus `nmap`, `nikto`, `gobuster`, `sqlmap`, `wpscan`, `whatweb`, `sslscan`, `masscan`, `crt.sh`, `dig`, `whois` dalam satu CLI interaktif berbahasa Indonesia. Tool ini terbukti efektif untuk audit pelanggan satu-per-satu, tapi sudah mentok di tiga tembok: **tidak bisa diskalakan**, **tidak bisa dipelihara secara modular**, dan **tidak siap cloud-native**.

Dokumen ini adalah dua hal sekaligus: (a) **reverse engineering audit** terhadap v6.1, dan (b) **cetak biru implementasi v7.0** sebagai framework Golang modular, plugin-based, terdistribusi, dan production-grade.

---

# PART I — REVERSE ENGINEERING

## §1. Ringkasan Reverse Engineering v6.1

**Distribusi:** `nightcrawler-v6.1.tar.gz` (44 KB) berisi 5 file: `nightcrawler` (binary utama — *sebenarnya* bash script 142 KB), `nightcrawler.conf`, `install.sh`, `uninstall.sh`, `README.md`.

**Komponen utama yang teridentifikasi:**

| Layer | Komponen | Implementasi v6.1 |
|---|---|---|
| **Entry & control** | `main()`, `show_menu()`, `input_setup()` | Loop interaktif read-eval, mode `--auto` untuk CI |
| **Evasion engine** | `stealth_curl()`, `UA_POOL`, `REFERER_POOL`, `DELAY_JITTER` | Random UA (12 entries), random Referer, jitter delay, header impersonation Chrome-style (Sec-Fetch-*) |
| **UI engine** | `banner()`, `spinner()`, `progress_bar()`, `matrix_rain()`, `glitch_text()`, `typing_effect()` | ANSI escape codes manual, ASCII spider neofetch-style |
| **Validation layer** | `detect_catchall()`, `is_genuine_response()`, `validate_path_content()`, `detect_tech_profile()`, `is_path_relevant()`, `is_suspicious_ip_diff()`, `count_gambling_density()` | Mitigasi false-positive hasil 5 iterasi (v4.0–v5.1) — *aset paling berharga di codebase* |
| **Scan modules (17)** | `scan_dns`, `scan_waf`, `scan_ports`, `scan_ssl`, `scan_headers`, `scan_dirs`, `scan_webshell`, `scan_cms`, `scan_gambling`, `scan_nikto`, `scan_sqli`, `scan_xss`, `scan_methods`, `scan_cors`, `scan_open_redirect` (v6.1), `scan_info_disclosure` (v6.1), `scan_response_time` (v6.1) | Setiap modul = function bash yang shell-out ke tool eksternal + parse output dengan grep/awk/sed |
| **Recon enrichment** | `scan_subdomain_passive` (crt.sh JSON via jq), `scan_dsstore_parse`, `scan_subdomain_deep`, `scan_subdomain_fingerprint` | Brute-force list 80 subdomain + passive crt.sh dengan `MAX_SUBDOMAINS_DEEP=50` cap |
| **Reporting** | `generate_txt_report()`, `generate_html_report()`, `generate_attack_surface_report()` | TXT plain + HTML 700+ baris (inline JetBrains Mono, gauge SVG, tabs, filter buttons, collapsibles, print mode) |
| **Persistence layer** | `add_finding()`, `FINDINGS_DB[]`, `RISK_CRITICAL/HIGH/MEDIUM/LOW` counters | Array string `LEVEL\|DESC\|MITIGATION`, agregasi via counter integer |
| **Storage** | `REPORT_DIR=$HOME/nightcrawler-reports/<domain>_<timestamp>/` | File-per-domain dengan struktur sub-folder `subdomains/<sub>/...` |

**Filosofi kode yang teramati:**
- Defensive-first: `set -o pipefail`, trap INT/TERM yang menyelamatkan partial report.
- Bilingual: komentar Bahasa Indonesia, label CLI Bahasa Indonesia, mitigasi dalam Bahasa Indonesia.
- Iteratif: changelog 5 versi (v4.0 → v6.1), tiap rilis fokus mitigasi false positive — bukan tambah fitur.
- Indonesian-context: wordlist berisi `sipd`, `simpeg`, `sister`, `siakad`, `sakip`, `lakip`, `presensi`, `baak`, `pmb`, `akademik` — jelas dioptimasi untuk audit instansi pendidikan & pemerintahan Indonesia.

---

## §2. Analisa Kelemahan (Audit Negatif)

Diurut berdasarkan severity terhadap maintainability + scalability:

### 2.1 — Arsitektur Monolitik
- **Penyebab:** Semua logic di 1 file 3.247 baris. Tidak ada separation of concerns. UI, parsing, networking, persistence, orchestration semua bercampur.
- **Dampak:** Setiap perubahan kecil = risiko regresi tinggi. Hanya satu orang (1607-NetEnginee sendiri) yang bisa contribute tanpa bingung.
- **Bukti dari code:** Function `scan_dirs()` (line 1493) sekaligus melakukan: catch-all detection, tech-profile detection, path iteration, HTTP request, content validation, finding registration, file logging, gobuster shell-out. 6 responsibility dalam 1 function.

### 2.2 — Bash sebagai Bahasa Implementasi
- **Penyebab:** Bash tidak punya struct/typed data, error handling lemah (`set -e` tidak menangkap subshell, pipe failures), dan tidak ada native concurrency selain `&` + `wait`.
- **Dampak:** Bug halus seperti `RISK_SCORE=0` di v5.1 (line 2762 changelog: *"generate_attack_surface_report risk score always 0 — subshell bug"*) — counter di-increment di dalam subshell pipe, perubahan hilang setelah subshell exit. Bug ini bertahan lebih dari satu rilis sebelum di-fix di v6.0.
- **Bukti:** Pattern `local x=$(echo "$y" | grep ... | tail -1)` muncul 200+ kali. Tidak ada type checking, semua string manipulation.

### 2.3 — Tidak Ada Concurrency
- **Penyebab:** Loop `for domain in "${TARGETS[@]}"; do run_full_scan "$domain"; done` di line 3200 — fully sequential.
- **Dampak:** Scan 10 domain dengan 17 modul masing-masing = 170 fase berurutan. Wall-clock time = sum of all module time. Pada VPS modern dengan 8 core, 7 core menganggur.
- **Estimasi:** Full scan 1 domain rata-rata 8-15 menit (nikto + gobuster dominan). Untuk 10 domain ≈ 2 jam. Versi concurrent bisa < 20 menit.

### 2.4 — Subprocess Hell
- **Penyebab:** Setiap fungsi memanggil `curl`, `dig`, `grep`, `awk`, `sed`, `jq` berkali-kali. Estimasi `scan_dirs()` saja: ~70 path × 2-3 `curl` = 140-210 subprocess fork per scan_dirs satu domain.
- **Dampak:** Overhead fork-exec di Linux ≈ 1-5ms per call. Untuk full scan: 5.000-10.000 subprocess → 5-50 detik *hanya untuk fork overhead* sebelum hitung waktu network.

### 2.5 — Output Format Tidak Machine-Friendly
- **Penyebab:** Output utama = TXT plain + HTML statis. Tidak ada JSON, NDJSON, atau format struktural lain.
- **Dampak:** Integrasi dengan SIEM (Splunk, ELK, Wazuh) memerlukan custom parser. Tidak bisa diumpan ke pipeline (post-process, deduplication, enrichment). Tidak bisa di-diff antar scan untuk detect drift.
- **Bukti:** `FINDINGS_DB+=("${level}|${desc}|${mitigation}")` — pipe-delimited string, akan break jika description atau mitigation mengandung `|`.

### 2.6 — Konfigurasi Sangat Terbatas
- **Penyebab:** `nightcrawler.conf` hanya 11 variabel. Banyak hal hard-coded: wordlist subdomain (line 1285-1291: 80 entries), sensitive paths list (line 1505-1528: 70 entries), webshell signatures (line 1616-1623: 23 entries), gambling keywords (line 1752-1757: 18 entries).
- **Dampak:** Tidak bisa adaptasi tanpa edit source. Tidak bisa ada community-contributed signature pack.

### 2.7 — Tidak Ada Plugin System
- **Penyebab:** Setiap "modul" adalah function bash. Tidak ada interface, tidak ada loader, tidak ada manifest.
- **Dampak:** User harus fork repo untuk tambah scanner baru. Tidak ada marketplace, tidak ada versioning per-module.

### 2.8 — Logging Tidak Terstruktur
- **Penyebab:** `log()` = `echo | tee`, semua level (info, warn, fail, ok) campur dalam satu file flat.
- **Dampak:** Tidak bisa filter by level di shell pipeline. Tidak bisa kirim ke central log collector (Loki, Vector, Fluent Bit) tanpa parsing regex rapuh. Tidak ada correlation ID antar request.

### 2.9 — Tidak Ada Resume / Idempotency
- **Penyebab:** Jika scan crash di tengah, harus mulai dari awal. Tidak ada checkpoint.
- **Dampak:** Untuk target dengan 100+ subdomain, gangguan jaringan 30 detik = restart dari nol.

### 2.10 — Validasi Path Type Hard-coded
- **Penyebab:** `validate_path_content()` (line 447) menggunakan `case` Bash untuk match path → expected content. Setiap path baru = edit Bash.
- **Dampak:** Aset paling cerdas di v6.1 (5 iterasi validasi false-positive) terkunci di Bash dan tidak bisa di-reuse oleh tool lain.

### 2.11 — Dependency Detection Lemah
- **Penyebab:** `install_deps()` menjalankan `apt-get install` tanpa version pinning, dan tergantung distro Debian/Ubuntu.
- **Dampak:** Tidak portable ke RHEL/Alpine. Tidak ada vendoring. Tidak ada hash verification.

### 2.12 — Risiko Race Condition
- **Penyebab:** `BASELINE_HASH` dan `CATCHALL_CACHE` adalah associative array global — aman selama sequential, tapi akan corrupt jika di-parallelize naif.
- **Dampak:** Memblokir jalan upgrade ke concurrent.

### 2.13 — Banner ASCII Tidak Konsisten Width
- **Penyebab:** `banner()` (line 264) padding string manual 46 char. Width tidak adaptif terhadap terminal.
- **Dampak:** Di terminal <80 col rusak. Di terminal lebar terlihat kosong.

### 2.14 — HTML Report Static
- **Penyebab:** HTML report generated dengan heredoc + variable interpolation. Tidak ada template engine, tidak ada client-side data binding.
- **Dampak:** Update look-and-feel = edit 700+ baris bash yang berisi CSS+JS+HTML. Tidak bisa export ke PDF tanpa puppeteer eksternal.

### 2.15 — Tidak Ada Test Suite
- **Penyebab:** Zero test file, zero CI, zero linter config.
- **Dampak:** Tidak ada safety net untuk refactor. Risiko regresi tinggi (terbukti dari 5 iterasi changelog yang banyak isinya `[FIX]`).

---

## §3. Analisa Kekuatan (Audit Positif)

Hal-hal di v6.1 yang **harus dipertahankan** dan dipindahkan ke v7.0:

### 3.1 — Validation Layer Sangat Matang
`validate_path_content()` (line 447) adalah hasil 5 iterasi engineering. Mengetahui bahwa:
- `.git/HEAD` harus mengandung `ref:`
- `.env` harus mengandung `KEY=VALUE` atau `APP_KEY/DB_HOST`
- `phpinfo.php` harus mengandung `PHP Version`
- `laravel.log` harus ada timestamp `[YYYY-MM-DD`
- `actuator/env` harus ada `"activeProfiles"`

Ini adalah **domain knowledge berharga** yang membedakan tool ini dari scanner generic. Wajib direplikasi ke v7.0 sebagai data-driven YAML signature database.

### 3.2 — Catch-all / Soft-404 Detection
`detect_catchall()` (line 406) request random slug, simpan baseline hash, lalu bandingkan setiap response. Mitigasi false-positive ini sering hilang di scanner Bash umum. **Wajib pertahankan.**

### 3.3 — Tech-aware Path Filtering
`is_path_relevant()` (line 548) tidak scan `wp-config.php` di non-WordPress site. Menghemat request dan mengurangi noise. **Wajib pertahankan.**

### 3.4 — Smart IP Differential
`is_suspicious_ip_diff()` (line 579) skip `ns1.`, `mail.`, dan IP range Google/Cloudflare/AWS yang dikenali sebagai mail relay/CDN. Tidak flag mail.domain.com → 74.125.x.x sebagai anomaly. **Wajib pertahankan.**

### 3.5 — Gambling Density Check
`count_gambling_density()` (line 615) strip tag `<nav>`/`<footer>`/`<header>` dan baru hitung density. Menghindari false-positive dari breadcrumb "Slots > Detail". **Wajib pertahankan** — relevan untuk konteks Indonesia.

### 3.6 — Indonesian Government & Education Knowledge
Wordlist v6.1 punya paths: `sipd`, `simpeg`, `sister`, `siakad`, `sakip`, `lakip`, `presensi`, `baak`, `pmb`, `akademik`, `dosen`, `mahasiswa`, `lms`, `elearning`. Wordlist gov/edu Indonesia **sulit ditemukan di tool internasional**. Aset diferensiasi.

### 3.7 — Sub-domain Priority Queue
`scan_subdomain_deep()` (line 2647) memprioritaskan subdomain `dev`, `staging`, `admin`, `vpn`, `git`, `jenkins`, dst sebelum yang regular. Cap `MAX_SUBDOMAINS_DEEP=50` mencegah runaway. **Wajib pertahankan.**

### 3.8 — Defensive Posture by Default
- Wajib `--auto "Client Name"` di mode otomatis (line 3192) — memaksa atribusi.
- `add_finding "MEDIUM" "WAF detected"` (line 1337) — flag WAF *sebagai pembatas scan*, bukan target bypass. Konsisten dengan posisi defensive.
- Trap INT/TERM yang flush partial report (line 199) — operator selalu dapat hasil parsial.

### 3.9 — Compact Distribution
44 KB tarball. Zero runtime dependency (selain coreutils & tools standard). Bisa drop ke airgapped VM. **Filosofi wajib pertahankan**: v7.0 single binary Go juga ≤ 30 MB.

### 3.10 — Interactive HTML Report (Konsep)
Tabs, filter, collapsible, gauge SVG, print stylesheet. Eksekusi messy (inline di bash heredoc), tapi konsepnya solid dan disukai operator non-teknis. **Konsep wajib pertahankan**, implementasi diganti template engine.

### 3.11 — Bilingual UX
Operator Indonesia mendapatkan mitigasi dalam Bahasa Indonesia. v7.0 wajib menyimpan struktur i18n agar fitur ini tetap ada.

### 3.12 — Stealth Engine Design Sensible
`stealth_curl()` mengkombinasikan random UA + random Referer + jitter delay + browser headers (Sec-Fetch-*) dengan masuk akal. Tidak overdesigned. **Pertahankan paradigmanya** (config-driven user-agent pool, jitter window).

---

## §4. Workflow Mapping (Eksekusi v6.1)

Diagram alur eksekusi `--auto`:

```
                       ┌────────────────────┐
                       │ main()             │
                       │ - parse args       │
                       │ - root check       │
                       │ - mkdir OUTPUT_DIR │
                       └─────────┬──────────┘
                                 │
                                 ▼
                       ┌────────────────────┐
                       │ install_deps()     │
                       │ apt-get install... │
                       └─────────┬──────────┘
                                 │
            ┌────────────────────┼─── for each target ───┐
            │                    ▼                       │
            │       ┌────────────────────┐               │
            │       │ run_full_scan(d)   │               │
            │       └─────────┬──────────┘               │
            │                 ▼                          │
            │   scan_dns ─→ scan_waf ─→ scan_ports       │ sequential
            │   ─→ scan_ssl ─→ scan_headers ─→ scan_dirs │ no parallelism
            │   ─→ scan_webshell ─→ scan_cms             │ inter-domain
            │   ─→ scan_gambling ─→ scan_nikto           │ or inter-module
            │   ─→ scan_sqli ─→ scan_xss                 │
            │   ─→ scan_methods ─→ scan_cors             │
            │   ─→ scan_open_redirect ─→ scan_info_disc  │ (v6.1)
            │   ─→ scan_response_time                    │
            │   ─→ generate_attack_surface_report        │
            └────────────────────┬───────────────────────┘
                                 ▼
                       ┌────────────────────┐
                       │ generate_txt_report│
                       │ generate_html_rpt  │
                       └────────────────────┘
```

**Per-module data flow:** Setiap `scan_X(domain)` ⇒ tulis file ke `$REPORT_DIR/$domain/<artifact>.txt` + panggil `add_finding()` yang update counter global. Tidak ada return value, tidak ada error propagation.

**Coupling rendah ke fungsi orchestrator**, tapi **coupling tinggi ke filesystem layout dan global state**.

---

## §5. Bottleneck Utama

Diurut by impact pada wall-clock time:

| # | Bottleneck | Lokasi | Impact | Solusi v7.0 |
|---|---|---|---|---|
| 1 | Sequential inter-module | `run_full_scan()` line 3121 | 17× sequential = 8-15 min/domain | Worker pool dengan dependency graph; modul independen jalan paralel |
| 2 | Sequential inter-domain | line 3200 | 10× domain sequential | DOP-level parallelism dengan concurrency cap |
| 3 | Subprocess fork overhead | semua scan modules | 5K-10K fork per scan | Go native `net/http` client; reuse connections |
| 4 | DNS resolver sequential | line 1296 brute-force loop | 80 dig calls × 100ms = 8s | `miekg/dns` library, parallel resolver pool |
| 5 | sslscan blocking | line 1397 | 5-30s per host | Native TLS dengan `crypto/tls`, parallel cipher probe |
| 6 | gobuster blocking | line 1583 | 5-10 min per host | Built-in fuzzer dengan worker pool |
| 7 | nikto blocking 5 min | line 1828 | Hard cap 5 min | Kept as optional plugin; tidak default di full-scan v7.0 |
| 8 | sqlmap dengan timeout 10s/payload | line 1868 | 30-300s | Kept as optional plugin; native fast probe untuk smoke test |
| 9 | crt.sh JSON parsing dengan jq | line 2402 | 5-30s tergantung response size | Native JSON parsing |
| 10 | `tr -d '\0' \| md5sum \| cut` chain | banyak tempat | Subprocess × 3 per call | Hash di memori |

---

## §6. Risiko Keamanan (Hardening Internal)

Risiko yang melekat pada **scanner itu sendiri**, bukan pada target:

### 6.1 — Wajib Berjalan sebagai Root
- **Sebab:** `apt-get install` di `install_deps()`, port scan privileged via nmap SYN scan.
- **Risiko:** Eskalasi privilege jika scanner kompromised. Tools (jq, curl) dari apt yang tidak diverifikasi.
- **Mitigasi v7.0:** Run as non-root. Port scan dengan TCP connect (tidak butuh root). Wrap nmap dalam capability container (`cap_net_raw`) bukan setuid root.

### 6.2 — Output Path Bisa Di-traverse via INSTANSI_NAME
- **Sebab:** Line 680: `REPORT_DIR="$OUTPUT_DIR/${INSTANSI_NAME// /_}_${primary_target}..."` — nama instansi hanya di-replace spasi.
- **Risiko:** Operator memasukkan `INSTANSI_NAME="../../../tmp/evil"` → path traversal. Karena scanner run sebagai root, bisa overwrite file system.
- **Mitigasi v7.0:** Whitelist regex `[a-zA-Z0-9_-]+`, atau hash-based folder name.

### 6.3 — Curl tanpa Cert Pinning
- **Sebab:** `stealth_curl` tidak verify CA store, tidak ada `--cacert` opt-in.
- **Risiko:** MITM dari attacker on-path bisa inject payload yang memicu false finding atau memanipulasi mitigasi text.
- **Mitigasi v7.0:** TLS verify default ON, ada flag `--insecure` eksplisit dengan warning.

### 6.4 — WPScan dengan `--disable-tls-checks`
- **Sebab:** Line 1695: `wpscan ... --disable-tls-checks`.
- **Risiko:** Memberi precedent buruk; meningkatkan attack surface scanner.
- **Mitigasi v7.0:** Hapus flag ini. Jika cert invalid, log warning, jangan disable verification.

### 6.5 — Storage Bukan Encrypted
- **Sebab:** `$REPORT_DIR` mode 750, plain text.
- **Risiko:** Pada VPS shared atau backup tape, finding (kadang berisi credential yang ditemukan di .env publik) dapat tersebar.
- **Mitigasi v7.0:** Opsi encrypt-at-rest dengan `age` atau AES-GCM, key dari env var atau Vault.

### 6.6 — Webshell Path Probe Bisa Trigger Honeypot/IDS
- **Sebab:** Path seperti `/c99.php`, `/r57.php`, `/wso.php` adalah signature payload terkenal. Banyak IDS (Suricata, mod_security) auto-block IP yang probe path ini.
- **Risiko:** Scanner IP di-blacklist mid-scan. Worse: dilaporkan ke abuse@isp.
- **Mitigasi v7.0:** Klasifikasi modul (`stealth` vs `noisy`), beri warning sebelum jalankan modul noisy, hormati `robots.txt` opsional.

### 6.7 — Tidak Ada Rate Limit Adaptif
- **Sebab:** Jitter delay tetap 0.3+rand(0.5) s, tidak adaptasi terhadap response code.
- **Risiko:** Jika target return 429/503, scanner tetap pukul → bisa DDoS tidak sengaja.
- **Mitigasi v7.0:** Adaptive rate limiter dengan exponential backoff pada 429/503/Retry-After header.

### 6.8 — sed/awk dengan Input dari Target
- **Sebab:** Line 615: `sed 's/<nav[^>]*>.*<\/nav>//gI'` pada konten HTML dari target.
- **Risiko:** Resource exhaustion via malicious response (sed/grep dengan ReDoS pattern atau extreme input).
- **Mitigasi v7.0:** Body size cap (`io.LimitReader`), regex dengan timeout (`regexp/syntax` dengan complexity check).

### 6.9 — Eksekusi Tool Eksternal Tanpa Path Pinning
- **Sebab:** `curl`, `dig` dipanggil dari `$PATH`. Jika `$PATH` dikompromikan, eksekusi arbitrary.
- **Mitigasi v7.0:** Native implementation, tidak shell-out. Jika harus shell-out, gunakan absolute path.

### 6.10 — Telegram Bot Token via Plain CLI
- **Sebab:** `monitor_realtime()` line 1984 prompt token, simpan di memory.
- **Risiko:** Tampil di shell history, ps output, atau core dump.
- **Mitigasi v7.0:** Baca dari env var atau secret store; tidak pernah dari argv.

---

## §7. Prioritas Refactor (P0 / P1 / P2)

Menggunakan kerangka **impact × effort**, item refactor dipetakan ke 3 tier:

### P0 — Wajib di v7.0 GA (tanpa ini, tidak release)

| Item | Impact | Effort | Justifikasi |
|---|---|---|---|
| Migrasi core ke Go | Sangat tinggi | Tinggi | Membuka jalan untuk semua P1/P2 |
| Plugin interface + manifest | Sangat tinggi | Sedang | Mengganti monolit menjadi modular |
| Worker pool + concurrency | Sangat tinggi | Sedang | 5-10× speedup wall-clock |
| Structured logging (slog/zerolog) | Tinggi | Rendah | Wajib untuk integrasi enterprise |
| YAML config + signature DB eksternal | Tinggi | Sedang | Memindahkan domain knowledge dari hardcoded ke data |
| JSON output (NDJSON event stream) | Tinggi | Rendah | Integrasi SIEM/pipeline |
| Native HTTP client dengan connection pooling | Tinggi | Rendah | Menghapus ribuan subprocess fork |
| Single binary distribution | Tinggi | Rendah | Konsisten dengan filosofi compact 44 KB |
| Validation layer di-port persis | Sangat tinggi | Sedang | Aset diferensiasi v6.1 |
| Indonesian wordlist + signature pack | Sangat tinggi | Rendah | Aset diferensiasi v6.1 |
| Trap signal → flush partial report | Sedang | Rendah | UX yang sudah baik di v6.1 |
| Non-root operation | Tinggi | Sedang | Hardening internal |
| Path traversal sanitization | Tinggi | Rendah | Hardening internal |

### P1 — Target v7.1 (3-6 bulan setelah GA)

| Item | Impact | Effort |
|---|---|---|
| BubbleTea TUI | Tinggi | Tinggi |
| Web dashboard (read-only) | Tinggi | Tinggi |
| HTML report dengan Go template engine | Sedang | Rendah |
| Adaptive rate limiter (429-aware) | Sedang | Sedang |
| Resume / checkpoint system | Sedang | Sedang |
| Distributed worker (gRPC agent) | Tinggi | Sangat tinggi |
| Encrypted output (`age`) | Sedang | Rendah |
| Telegram/Slack/Discord notifier | Sedang | Rendah |

### P2 — Target v7.5+ (vision items)

| Item | Impact | Effort |
|---|---|---|
| Plugin marketplace | Tinggi | Sangat tinggi |
| AI risk scoring + LLM mitigation generator | Tinggi | Tinggi |
| RBAC + multi-tenant dashboard | Tinggi | Sangat tinggi |
| Threat intel correlation (OSINT enrichment) | Tinggi | Tinggi |
| Headless browser crawling (chromedp) | Tinggi | Tinggi |
| Screenshot engine + visual diff | Sedang | Sedang |
| Plugin signing & supply chain | Tinggi | Tinggi |

**Prinsip prioritasi:** P0 fokus pada *paritas fungsional + performance unlock*. Tidak ada fitur baru di v7.0 GA yang tidak ada di v6.1 — semua "fitur powerful" dari brief Anda (HTTP/3, AI scoring, marketplace) masuk roadmap P1/P2 supaya v7.0 GA bisa di-ship dalam 8-12 minggu, bukan 12 bulan. Overengineering adalah aturan utama yang harus dihindari (sesuai brief).

---

## §8. Strategi Migrasi Bash → Go

### 8.1 — Strategi Top-Level: "Skeleton First, Modules in Waves"

Bukan rewrite big-bang. Tiga gelombang:

**Wave A (minggu 1-3):** Bangun kerangka kosong yang berjalan:
- Cobra root command, flag parsing, config loader (Viper)
- Logger (slog), event bus, plugin loader interface
- Worker pool primitive
- Output writer (NDJSON + TXT)
- ONE end-to-end skeleton module: `dns` (paling sederhana, tidak butuh body parsing)

Kriteria done: `nightcrawler scan --target example.com --module dns -o out.json` menghasilkan NDJSON valid.

**Wave B (minggu 4-8):** Port modul 1-by-1 dengan urutan kompleksitas naik:
1. `dns` — simple DNS lookup + brute subdomain
2. `tls` — TLS handshake, cert info, cipher enumeration
3. `headers` — HTTP HEAD/GET, parse headers, score security headers
4. `ports` — TCP connect scan (default), opsi spawn nmap
5. `paths` — sensitive file scanner (port validation layer v6.1)
6. `webshell` — path probe + signature match
7. `cms` — fingerprint via headers + body patterns
8. `methods` — OPTIONS + per-method probe
9. `cors` — header probe dengan origin permutations
10. `gambling` — keyword density check (port v6.1 logic)
11. `redirect` — open redirect dengan canary domain
12. `disclosure` — stack trace patterns
13. `timing` — baseline + probe time-based
14. `xss` — reflected payload probe
15. `sqli` — fast error-based probe (sqlmap = plugin opsional)
16. `nikto` — wrapper plugin opsional
17. `crtsh` — passive recon via crt.sh API

Setiap modul harus lulus:
- Unit test untuk validation logic
- Integration test terhadap fixtures (recorded HTTP responses)
- Golden file test untuk output struktur

**Wave C (minggu 9-12):** Hardening + UX
- Trap signal, partial report
- TUI (Bubble Tea)
- HTML report dengan template
- Docker, CI/CD, release pipeline
- Documentation, README, contributing guide

### 8.2 — Aturan Translasi Bash → Go

| Pola Bash v6.1 | Padanan Go v7.0 |
|---|---|
| `curl -s -A "$ua" "$url"` | `http.Client` global dengan custom `RoundTripper` yang inject UA pool + jitter |
| `dig +short A "$d"` | `github.com/miekg/dns` |
| `grep -oP "regex"` | `regexp.MustCompile(...).FindAllStringSubmatch(...)` |
| `awk '{print $3}'` | `strings.Fields(line)[2]` atau `bufio.Scanner` |
| `sed 's/x/y/g'` | `strings.ReplaceAll` atau `regexp.ReplaceAllString` |
| `md5sum` | `crypto/md5` (in-memory) |
| `jq '.[]'` | `encoding/json` decoder dengan struct tag |
| `for path in "${PATHS[@]}"; do ...` | `errgroup.Group` + worker pool |
| `printf` formatting | `fmt.Sprintf` |
| `RISK_CRITICAL=$((+1))` global counter | Channel-based event aggregator |
| `cat > file <<EOF` heredoc HTML | `html/template` dengan `embed.FS` |
| `tput civis/cnorm` | Bubble Tea cursor management |
| `trap cleanup INT TERM` | `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` |
| `apt-get install` | Tidak ada — single static binary |

### 8.3 — Mapping Validation Layer ke Go

Aset terpenting di v6.1 (§3.1-3.5) di-translate menjadi data, bukan kode:

`internal/signatures/paths.yaml`:
```yaml
- path: "**/.git/HEAD"
  validators:
    - contains: "ref:"
- path: "**/.env*"
  validators:
    - matches_regex: '^[A-Z_]+=.+'
    - any_contains: ["APP_KEY", "DB_HOST", "DB_PASSWORD", "DB_NAME"]
- path: "**/phpinfo.php"
  validators:
    - any_contains: ["PHP Version", "phpinfo()", "php_uname"]
- path: "**/laravel.log"
  validators:
    - matches_regex: '\[\d{4}-\d{2}-\d{2}'
    - any_contains: ["local.ERROR", "local.INFO", "production.ERROR", "APP_ENV"]
```

Engine validator generic membaca YAML dan apply ke setiap response. Menambahkan path baru = tambah YAML entry, tidak edit Go code. Community contribution lewat PR ke signature pack.

### 8.4 — Output Compatibility Mode

v7.0 menulis NDJSON sebagai output kanonik, tapi punya mode `--legacy-format=v6` yang menghasilkan TXT/HTML mirip v6.1 untuk operator yang sudah punya workflow lama. Periode deprecation 6 bulan, kemudian dihapus di v7.5.

### 8.5 — Migrasi Konfigurasi

Script kecil `nightcrawler config import-v6 /etc/nightcrawler/nightcrawler.conf` membaca format Bash, men-translate ke YAML v7.0, dan menyimpan ke `~/.config/nightcrawler/config.yaml`. Operator existing tidak perlu setup ulang.

---

## §9. Arsitektur NIGHTCRAWLER v7.0

### 9.1 — High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         NIGHTCRAWLER v7.0 — TOPOLOGY                         │
└─────────────────────────────────────────────────────────────────────────────┘

           ┌───────────────────────────────────────────────┐
           │              EDGE / OPERATOR                  │
           │   ┌─────────────┐    ┌─────────────────────┐  │
           │   │  CLI (Cobra) │    │  TUI (Bubble Tea)  │  │
           │   └──────┬──────┘    └──────────┬──────────┘  │
           │          └──────────┬───────────┘             │
           └─────────────────────┼─────────────────────────┘
                                 │  invoke: scan, report, plugin, agent
                                 ▼
           ┌────────────────────────────────────────────────────┐
           │                  CORE ORCHESTRATOR                  │
           │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │
           │  │  Config  │  │  Plugin  │  │   Event Bus      │  │
           │  │  Loader  │  │  Loader  │  │  (channels)      │  │
           │  └──────────┘  └──────────┘  └──────────────────┘  │
           │  ┌──────────────────────────────────────────────┐  │
           │  │    Scheduler (DAG + Worker Pool)             │  │
           │  └──────────────────────────────────────────────┘  │
           │  ┌──────────────────────────────────────────────┐  │
           │  │  HTTP Engine (pooled, stealth, adaptive RL)  │  │
           │  └──────────────────────────────────────────────┘  │
           │  ┌──────────────────────────────────────────────┐  │
           │  │  DNS Engine | TLS Engine | Port Engine       │  │
           │  └──────────────────────────────────────────────┘  │
           └─────────────────────┬──────────────────────────────┘
                                 │
        ┌────────────────────────┼────────────────────────┐
        ▼                        ▼                        ▼
  ┌───────────┐          ┌───────────┐          ┌──────────────┐
  │ Plugins   │          │ Validator │          │  Findings    │
  │ (modular  │  ──────▶ │ Layer     │  ──────▶ │  Aggregator  │
  │ scanners) │          │ (YAML DB) │          │  (channels)  │
  └───────────┘          └───────────┘          └──────┬───────┘
                                                       │
                                                       ▼
                              ┌─────────────────────────────────────┐
                              │         OUTPUT LAYER                 │
                              │   NDJSON │ TXT │ HTML │ SARIF │ PDF  │
                              │   ↓                                  │
                              │   Sinks: file, S3, webhook, syslog,  │
                              │          Telegram, Slack, Discord    │
                              └─────────────────────────────────────┘

                              ┌─────────────────────────────────────┐
                              │   OPTIONAL: DISTRIBUTED MODE         │
                              │                                      │
                              │   Controller ←── gRPC ──→ Agent[N]   │
                              │      ↑                       ↓       │
                              │      └─────── results ──────┘        │
                              └─────────────────────────────────────┘

                              ┌─────────────────────────────────────┐
                              │   OPTIONAL: WEB DASHBOARD (v7.1+)    │
                              │   - WebSocket live log               │
                              │   - Scan history, asset inventory    │
                              │   - SOC-style attack surface view    │
                              └─────────────────────────────────────┘
```

### 9.2 — Komponen Inti

**Edge Layer:**
- **CLI (Cobra):** root command + subcommand: `scan`, `report`, `plugin`, `agent`, `config`, `serve`.
- **TUI (Bubble Tea):** interactive mode dengan live worker monitor, queue, log panel.

**Core Orchestrator:**
- **Config Loader:** Viper, hierarchy `$XDG_CONFIG_HOME/nightcrawler/config.yaml` → env var → CLI flag.
- **Plugin Loader:** loader berbasis Go plugin manifest YAML; in-process plugins via `init()` registration; remote plugins via gRPC sidecar (v7.1+).
- **Event Bus:** unbuffered channels untuk findings, status updates, log records.
- **Scheduler:** DAG dependency-aware (mis. `paths` butuh `tech_profile` dari `cms`); worker pool dengan concurrency cap per target dan global.
- **HTTP Engine:** singleton `http.Client` dengan `Transport` custom: connection pool, jitter middleware, UA rotation, adaptive backoff pada 429/503, body size limit.
- **DNS/TLS/Port Engine:** native implementation, shared resource (mis. DNS resolver cache).

**Plugins:** Tiap scanner adalah package Go yang implementasi `Plugin` interface (lihat §12). Modul v6.1 di-port satu-per-satu sebagai plugin built-in.

**Validator Layer:** Engine yang load YAML signature DB dan apply ke response untuk validasi content + filter false positive.

**Findings Aggregator:** Goroutine yang collect events dari event bus, deduplikasi (hash dari `level+resource+rule`), calculate risk score, write ke output.

**Output Layer:** Multi-writer. NDJSON adalah format kanonik, semua format lain di-derive dari sana (template-based).

### 9.3 — Lifecycle Sebuah Scan

```
1. CLI parse → ScanConfig{targets, modules, output_dir, concurrency, profile}
2. Config loader merge defaults + file + env + flags
3. Plugin loader resolve module list → []Plugin
4. Scheduler build DAG, urutkan dengan topological sort
5. For each target → spawn target goroutine
6. Target goroutine:
   a. ResolveTarget (DNS, IP, reachability)
   b. Detect catchall (cached per host)
   c. Detect tech profile (cached per host)
   d. For each plugin in DAG order:
      - Submit ke worker pool
      - Plugin emit findings via event bus
   e. Wait all done atau context cancel
7. Aggregator consume events, write NDJSON in real time
8. On done atau signal → finalize report (HTML, TXT, summary)
9. Notify sinks (Telegram, Slack)
10. Exit code: 0=clean, 1=findings, 2=error
```

### 9.4 — State Management

State global v6.1 (RISK_CRITICAL, FINDINGS_DB, CATCHALL_CACHE) diganti dengan:

```go
type ScanContext struct {
    Targets       []Target
    Config        *Config
    Concurrency   ConcurrencyLimits
    Cache         *sync.Map      // catchall, tech profile, dns
    Findings      chan Finding   // event bus
    Logger        *slog.Logger
    Tracer        trace.Tracer   // OpenTelemetry
    Ctx           context.Context
    Cancel        context.CancelFunc
}
```

Sub-contexts (mis. `TargetContext`, `ModuleContext`) embed `*ScanContext` dan tambah field spesifik. Tidak ada global mutable state. Concurrency safe by construction.

---

## §10. Struktur Folder Professional

Berikut struktur GitHub-ready yang mengikuti idiom Go community (`golang-standards/project-layout`) dengan penyesuaian untuk security tool:

```
nightcrawler/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml                    # lint, test, build matrix
│   │   ├── release.yml               # goreleaser on tag
│   │   ├── codeql.yml                # security scanning
│   │   ├── docker.yml                # multi-arch image build
│   │   └── docs.yml                  # mkdocs deploy
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md
│   │   ├── feature_request.md
│   │   └── plugin_proposal.md
│   ├── PULL_REQUEST_TEMPLATE.md
│   ├── CODEOWNERS
│   ├── dependabot.yml
│   └── SECURITY.md
│
├── cmd/
│   ├── nightcrawler/
│   │   └── main.go                   # entry point
│   ├── nightcrawler-agent/
│   │   └── main.go                   # distributed worker (v7.1+)
│   └── nightcrawler-ctl/
│       └── main.go                   # control plane CLI (v7.1+)
│
├── internal/                         # tidak importable dari luar repo
│   ├── cli/
│   │   ├── root.go                   # cobra root
│   │   ├── scan.go                   # `scan` subcommand
│   │   ├── report.go                 # `report` subcommand
│   │   ├── plugin.go                 # `plugin list/install/info`
│   │   ├── agent.go                  # `agent serve` (v7.1+)
│   │   ├── config.go                 # `config init/import-v6`
│   │   └── version.go
│   ├── core/
│   │   ├── orchestrator.go           # ScanContext, lifecycle
│   │   ├── scheduler.go              # DAG, worker pool
│   │   ├── aggregator.go             # findings consumer
│   │   ├── cache.go                  # catchall, tech profile cache
│   │   └── target.go                 # Target resolution
│   ├── config/
│   │   ├── config.go                 # Viper bindings
│   │   ├── defaults.go               # baked-in defaults
│   │   ├── validate.go               # JSONSchema validation
│   │   └── migrate.go                # v6 → v7 importer
│   ├── http/
│   │   ├── client.go                 # singleton http.Client
│   │   ├── transport.go              # custom RoundTripper
│   │   ├── stealth.go                # UA pool, referer, jitter
│   │   ├── ratelimit.go              # adaptive backoff
│   │   └── pool.go                   # connection pool tuning
│   ├── dns/
│   │   ├── resolver.go               # miekg/dns wrapper
│   │   ├── bruteforce.go             # parallel sub enum
│   │   └── crtsh.go                  # crt.sh client
│   ├── tls/
│   │   ├── inspector.go              # cipher, cert, protocol enum
│   │   └── classifier.go             # severity classification
│   ├── plugin/
│   │   ├── plugin.go                 # Plugin interface
│   │   ├── manifest.go               # manifest YAML parser
│   │   ├── registry.go               # in-process plugin registry
│   │   ├── loader.go                 # discovery + init
│   │   └── grpc/                     # remote plugin proto (v7.1+)
│   ├── plugins/                      # built-in plugins
│   │   ├── dns/
│   │   ├── tls/
│   │   ├── headers/
│   │   ├── ports/
│   │   ├── paths/
│   │   ├── webshell/
│   │   ├── cms/
│   │   ├── methods/
│   │   ├── cors/
│   │   ├── gambling/
│   │   ├── redirect/
│   │   ├── disclosure/
│   │   ├── timing/
│   │   ├── xss/
│   │   ├── sqli/
│   │   └── crtsh/
│   ├── validator/
│   │   ├── engine.go                 # generic validator
│   │   ├── catchall.go               # soft-404 detector
│   │   ├── techprofile.go            # tech stack detection
│   │   ├── pathrelevance.go          # tech-aware filter
│   │   ├── ipdiff.go                 # smart IP differential
│   │   └── density.go                # keyword density check
│   ├── findings/
│   │   ├── finding.go                # Finding struct
│   │   ├── severity.go               # severity enum + scoring
│   │   ├── dedupe.go                 # deduplication
│   │   └── classifier.go             # tag, CWE/CVE mapping
│   ├── output/
│   │   ├── writer.go                 # multi-writer
│   │   ├── ndjson.go                 # NDJSON sink
│   │   ├── txt.go                    # legacy v6 TXT
│   │   ├── html.go                   # template-based HTML
│   │   ├── sarif.go                  # SARIF 2.1.0 output
│   │   └── notify/                   # Telegram/Slack/Discord/syslog
│   ├── report/
│   │   ├── builder.go                # assemble report data
│   │   ├── template/                 # embedded HTML templates
│   │   │   ├── report.html.tmpl
│   │   │   ├── components/
│   │   │   └── assets/               # CSS, JS, fonts (embedded)
│   │   └── pdf.go                    # PDF export via chromedp (P2)
│   ├── tui/
│   │   ├── app.go                    # Bubble Tea root model
│   │   ├── views/                    # views: home, scan, log, findings
│   │   ├── components/               # spinner, progress, table
│   │   └── theme.go                  # LipGloss styles
│   ├── telemetry/
│   │   ├── logger.go                 # slog wrapper
│   │   ├── tracer.go                 # OpenTelemetry init
│   │   └── metrics.go                # Prometheus metrics
│   └── version/
│       └── version.go                # set via ldflags at build
│
├── pkg/                              # API public, importable by 3rd party
│   ├── api/
│   │   ├── plugin.go                 # public Plugin interface
│   │   └── types.go                  # public types (Finding, Target)
│   └── client/
│       └── client.go                 # Go SDK untuk controller API
│
├── api/
│   ├── proto/                        # gRPC definitions (v7.1+)
│   │   ├── agent.proto
│   │   └── plugin.proto
│   └── openapi/
│       └── controller.yaml           # REST API spec
│
├── configs/
│   ├── nightcrawler.example.yaml     # full annotated default
│   └── profiles/
│       ├── stealth.yaml
│       ├── aggressive.yaml
│       ├── quick.yaml
│       └── compliance.yaml
│
├── signatures/                       # data-driven signature DB
│   ├── paths/
│   │   ├── sensitive-files.yaml
│   │   ├── webshells.yaml
│   │   └── id-gov-edu.yaml           # Indonesian gov/edu wordlist
│   ├── tech/
│   │   ├── frameworks.yaml
│   │   ├── cms.yaml
│   │   └── waf.yaml
│   ├── content/
│   │   ├── gambling.yaml
│   │   ├── stack-traces.yaml
│   │   └── malware-iocs.yaml
│   └── headers/
│       └── security-headers.yaml
│
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile                # multi-stage, distroless
│   │   ├── Dockerfile.agent
│   │   └── docker-compose.yaml       # local dev stack
│   ├── kubernetes/
│   │   ├── helm/
│   │   └── manifests/
│   └── systemd/
│       └── nightcrawler-agent.service
│
├── scripts/
│   ├── install.sh                    # curl|bash installer (signed checksum)
│   ├── lint.sh
│   ├── test.sh
│   └── release.sh
│
├── docs/
│   ├── ANALYSIS_AND_REDESIGN.md      # dokumen ini
│   ├── ARCHITECTURE.md
│   ├── PLUGIN_DEVELOPMENT.md
│   ├── DEPLOYMENT.md
│   ├── CONFIGURATION.md
│   ├── SECURITY.md
│   ├── MIGRATION_V6_TO_V7.md
│   ├── CONTRIBUTING.md
│   ├── ROADMAP.md
│   └── img/                          # diagrams, screenshots
│
├── examples/
│   ├── basic-scan/
│   ├── custom-plugin/
│   ├── ci-integration/
│   └── docker-deploy/
│
├── test/
│   ├── integration/
│   ├── fixtures/                     # recorded HTTP responses
│   │   ├── wordpress-site/
│   │   ├── laravel-site/
│   │   └── catchall-site/
│   ├── e2e/
│   └── benchmark/
│
├── third_party/                      # vendored embeds (fonts, etc)
│   └── fonts/
│
├── .editorconfig
├── .gitignore
├── .gitattributes
├── .golangci.yml                     # linter config
├── .goreleaser.yaml                  # release automation
├── .pre-commit-config.yaml
├── CHANGELOG.md
├── CODE_OF_CONDUCT.md
├── CONTRIBUTING.md
├── LICENSE                           # Apache 2.0 atau MIT
├── Makefile
├── README.md                         # bilingual EN + ID
├── README.id.md
├── SECURITY.md
├── go.mod
└── go.sum
```

**Justifikasi struktur:**
- `cmd/` plural → multi-binary (main, agent, ctl) realistis untuk roadmap distributed.
- `internal/` di-enforce oleh Go compiler → API stability bukan kewajiban semua sub-package, hanya `pkg/`.
- `pkg/api/` minimalis (Plugin interface + types) → menjaga API surface kecil supaya breaking change rare.
- `signatures/` dipisah dari `internal/plugins/` → community PR ke signature pack tidak butuh Go review.
- `deployments/` follow K8s convention.
- `configs/profiles/` → preset siap pakai (stealth, aggressive, quick, compliance) mengurangi friction adopsi.

---

## §11. Library Golang Terbaik (Stack Justified)

Pilihan dibatasi pada library yang **mature (≥1.0)**, **maintained dalam 6 bulan terakhir**, dan **license permissive**. Setiap pilihan dijustifikasi.

### 11.1 — CLI & Config

| Library | Versi | Justifikasi | Alternatif Dipertimbangkan |
|---|---|---|---|
| `github.com/spf13/cobra` | latest | Standar de-facto di ekosistem Go (kubectl, hugo, gh). Subcommand + flag inheritance bersih. | urfave/cli — kurang feature kaya |
| `github.com/spf13/viper` | latest | Sister-project Cobra. Layered config (defaults < file < env < flag). | koanf — bagus tapi ekosistem lebih kecil |
| `github.com/santhosh-tekuri/jsonschema/v6` | latest | Validate YAML config terhadap JSONSchema. Mature, zero-dep. | gojsonschema (deprecated) |

### 11.2 — Concurrency & Async

| Library | Justifikasi |
|---|---|
| `golang.org/x/sync/errgroup` | Worker pool dengan first-error semantics |
| `golang.org/x/sync/semaphore` | Concurrency cap, weighted semaphore |
| `golang.org/x/time/rate` | Token bucket rate limiter |
| `github.com/sourcegraph/conc` | Higher-level patterns: `pool.New().WithMaxGoroutines(n)` lebih ergonomik dari raw errgroup |
| stdlib `context` | Cancellation, deadline propagation (P0) |

### 11.3 — Networking

| Library | Justifikasi |
|---|---|
| stdlib `net/http` | Baseline. Custom transport untuk pool tuning. |
| `github.com/miekg/dns` | Standar Go untuk DNS query. Digunakan oleh CoreDNS. |
| `github.com/refraction-networking/utls` | JA3/JA4 fingerprint customization — diperlukan untuk WAF evasion (legitimate red-team) dan TLS fingerprinting (Wave B+) |
| `github.com/quic-go/quic-go` | HTTP/3 support (P1) |
| `golang.org/x/net/http2` | HTTP/2 tuning |
| `github.com/projectdiscovery/fastdialer` | Pluggable resolver dengan multi-backend (system, custom DNS) |

### 11.4 — Parsing & Manipulation

| Library | Justifikasi |
|---|---|
| stdlib `encoding/json` | NDJSON output. Untuk parse berat: `github.com/goccy/go-json` (faster, drop-in) |
| `gopkg.in/yaml.v3` | YAML config & signature DB. v3 dengan node API untuk preserve comments |
| `github.com/PuerkitoBio/goquery` | jQuery-like HTML parsing untuk extract endpoints, forms (lebih baik dari regex) |
| `golang.org/x/net/html` | HTML5 tokenizer underlying goquery |
| `github.com/tidwall/gjson` | Fast JSON path-based extraction untuk API responses |

### 11.5 — TUI & Terminal

| Library | Justifikasi |
|---|---|
| `github.com/charmbracelet/bubbletea` | Elm-architecture TUI, ekosistem matang (digunakan glow, gum, soft-serve) |
| `github.com/charmbracelet/lipgloss` | CSS-like styling, adaptif color |
| `github.com/charmbracelet/bubbles` | Pre-built components: spinner, progress, table, viewport |
| `github.com/charmbracelet/glamour` | Markdown rendering di terminal (untuk help & mitigasi) |
| `github.com/muesli/termenv` | Color detection, dim/no-color modes |

### 11.6 — Logging & Observability

| Library | Justifikasi |
|---|---|
| stdlib `log/slog` (Go 1.21+) | Structured logging, JSON output siap untuk SIEM. Standar baru. |
| `github.com/lmittmann/tint` | Pretty handler untuk slog di terminal interaktif |
| `go.opentelemetry.io/otel` | Tracing untuk distributed mode (P1) |
| `github.com/prometheus/client_golang` | Metrics endpoint untuk dashboard (P1) |

### 11.7 — Crypto & Security

| Library | Justifikasi |
|---|---|
| stdlib `crypto/tls`, `crypto/x509` | Cert inspection, cipher enumeration |
| `filippo.io/age` | Encrypt-at-rest output (P1) |
| `github.com/google/go-attestation` | Plugin signing (P2) |
| stdlib `crypto/ed25519` | Plugin signature verification |

### 11.8 — Web (Dashboard, v7.1+)

| Library | Justifikasi |
|---|---|
| stdlib `net/http` + `http.ServeMux` (Go 1.22+) | Pattern routing native, hindari heavy framework |
| `github.com/go-chi/chi/v5` | Bila pattern routing native tidak cukup ergonomis |
| `nhooyr.io/websocket` | WebSocket modern (preferred over gorilla/websocket yang archived) |
| `github.com/a-h/templ` | Type-safe Go templates (alternatif: html/template + htmx) |
| `htmx` (client-side) | Tanpa SPA framework, dashboard tetap reactive |

### 11.9 — Storage (Optional Enterprise Tier)

| Library | Justifikasi |
|---|---|
| `github.com/uptrace/bun` atau stdlib `database/sql` + `lib/pq` | PostgreSQL untuk scan history (P1) |
| `go.etcd.io/bbolt` | Embedded KV untuk single-node deployment |
| `github.com/redis/go-redis/v9` | Queue + cache di distributed mode (P1) |

### 11.10 — Testing & Quality

| Library | Justifikasi |
|---|---|
| stdlib `testing` | Baseline |
| `github.com/stretchr/testify` | Assert/require ergonomic, mature |
| `github.com/h2non/gock` atau `httptest` | HTTP mocking |
| `github.com/dnaeon/go-vcr` | Record/replay real HTTP untuk integration test |
| `github.com/golangci/golangci-lint` | Aggregator linter (gofmt, govet, staticcheck, gosec, errcheck) |
| `mvdan.cc/gofumpt` | Stricter formatter |

### 11.11 — Build & Release

| Tool | Justifikasi |
|---|---|
| `goreleaser` | Cross-compile, archive, checksum, sign, GitHub release dalam satu YAML |
| `cosign` | Sigstore signing untuk binary + container image |
| `syft` + `grype` | SBOM generation + vulnerability scan |
| Distroless base image (gcr.io/distroless/static) | Minimal container, no shell |

### 11.12 — Yang TIDAK Dipilih (Justifikasi)

| Library | Alasan Reject |
|---|---|
| `gin` / `echo` / `fiber` | Web dashboard tidak perlu heavy framework; stdlib + chi cukup |
| `logrus` / `zap` | `slog` adalah masa depan; logrus maintenance-only, zap masih relevan tapi slog cukup |
| `gorilla/*` (most) | Archived 2022. Pakai stdlib atau nhooyr |
| `viper` untuk plugin config | Plugin manifest pakai yaml.v3 langsung (lebih ringan, predictable) |
| Plugin via Go `plugin` package | Tidak portable (Linux only, kompiler version-sensitive). Pakai in-process registration + gRPC sidecar untuk remote |
| ORMs heavy (`gorm`) | Overhead besar untuk schema sederhana; pakai bun atau sqlx |

**Total runtime dependency target:** ≤ 35 modul direct (saat ini draft sekitar 28). Setiap dep harus melewati gate: license OK, maintained, ≥100 stars atau used by reputable project, alternatif sudah dievaluasi.

---

## §12. Plugin Architecture Design

### 12.1 — Filosofi Plugin

Plugin di v7.0 bukan barang ajaib. Setiap plugin adalah Go package yang implement satu interface kecil. Built-in plugin di-import secara statis. Out-of-tree plugin (v7.1+) tersedia via gRPC sidecar — bukan Go `plugin` package (alasan: tidak portable, version-sensitive, tidak bisa di-cross-compile).

### 12.2 — Plugin Interface (Public API)

`pkg/api/plugin.go`:

```go
package api

import "context"

type Plugin interface {
    // Manifest: nama, versi, deskripsi, dependencies.
    Manifest() Manifest

    // Init: dipanggil sekali saat startup. Inject HTTP client, logger, config.
    Init(ctx context.Context, deps Deps) error

    // Run: dipanggil sekali per target. Emit findings via emitter.
    Run(ctx context.Context, target Target, emit Emitter) error
}

type Manifest struct {
    Name         string   // "paths"
    Version      string   // "1.2.0", semver
    Author       string
    Description  string
    Category     Category // CategoryRecon | CategoryVuln | CategoryFingerprint
    Tags         []string // "sensitive-files", "stealth-safe"
    DependsOn    []string // ["dns", "tech-profile"]
    Profile      Profile  // Stealth | Default | Aggressive
    OutputFields []string // dokumentasi field
    CWE          []string // CWE-200, dst
}

type Deps struct {
    HTTP      *http.Client      // shared, pooled
    DNS       DNSResolver
    Logger    *slog.Logger
    Validator *validator.Engine // load YAML signature
    Cache     *cache.Cache      // catchall, tech profile cache
    Config    plugin.Config     // sub-config plugin (map dari YAML)
}

type Emitter func(Finding)

type Finding struct {
    PluginName  string
    Level       Severity // Critical | High | Medium | Low | Info
    Resource    string   // "https://target.com/.env"
    Title       string
    Description string
    Evidence    []Evidence
    Mitigation  string   // bilingual via locale
    References  []Ref    // CWE/CVE/OWASP
    Tags        []string
    Timestamp   time.Time
}
```

### 12.3 — Plugin Manifest (YAML)

Setiap built-in plugin juga punya manifest YAML untuk introspection (`nightcrawler plugin info <name>`):

`internal/plugins/paths/plugin.yaml`:
```yaml
apiVersion: nightcrawler.io/v1
kind: Plugin
metadata:
  name: paths
  version: 1.0.0
  author: 1607-NetEnginee
spec:
  description: |
    Sensitive file & directory discovery dengan tech-aware filtering.
    Port dari scan_dirs() v6.1 + validation layer.
  category: vuln
  profile: default
  tags: [sensitive-files, stealth-safe, builtin]
  dependsOn: [dns, tech-profile, catchall]
  signatures:
    - signatures/paths/sensitive-files.yaml
    - signatures/paths/id-gov-edu.yaml
  config:
    paths_file: signatures/paths/sensitive-files.yaml
    enable_gobuster_fallback: false
    max_concurrent_probes: 20
  outputFields:
    - resource
    - status_code
    - content_validated
    - tech_filter_applied
  cwe: [CWE-200, CWE-538, CWE-552]
```

### 12.4 — Implementasi Plugin Contoh

`internal/plugins/paths/paths.go` (skeleton):

```go
package paths

import (
    "context"
    "fmt"
    "net/http"
    "sync"

    "github.com/1607-NetEnginee/NightCrawler/pkg/api"
    "golang.org/x/sync/semaphore"
)

type Plugin struct {
    cfg     Config
    http    *http.Client
    valid   *validator.Engine
    logger  *slog.Logger
    sigDB   *SignatureDB
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Manifest() api.Manifest {
    return api.Manifest{
        Name: "paths", Version: "1.0.0", Author: "1607-NetEnginee",
        Category: api.CategoryVuln, Profile: api.ProfileDefault,
        DependsOn: []string{"dns", "tech-profile", "catchall"},
    }
}

func (p *Plugin) Init(ctx context.Context, deps api.Deps) error {
    p.http, p.valid, p.logger = deps.HTTP, deps.Validator, deps.Logger
    var err error
    p.sigDB, err = LoadSignatures(deps.Config.GetString("paths_file"))
    return err
}

func (p *Plugin) Run(ctx context.Context, target api.Target, emit api.Emitter) error {
    techProfile, _ := target.Cache.Get("tech-profile")
    catchall, _ := target.Cache.Get("catchall")

    sem := semaphore.NewWeighted(int64(p.cfg.MaxConcurrentProbes))
    var wg sync.WaitGroup

    for _, path := range p.sigDB.Paths {
        if !path.RelevantFor(techProfile) {
            continue
        }
        wg.Add(1)
        sem.Acquire(ctx, 1)
        go func(path PathSignature) {
            defer wg.Done()
            defer sem.Release(1)
            p.probePath(ctx, target, path, catchall, emit)
        }(path)
    }
    wg.Wait()
    return nil
}

func (p *Plugin) probePath(ctx context.Context, t api.Target, sig PathSignature,
    catchall *Catchall, emit api.Emitter) {
    resp, body, err := p.http.GetWithBody(ctx, t.URL+sig.Path)
    if err != nil || resp.StatusCode != 200 {
        return
    }
    if catchall.Matches(body) {
        return
    }
    if !sig.Validate(body) {
        p.logger.Debug("content mismatch", "path", sig.Path)
        return
    }
    level := sig.Severity
    if sig.ContainsSecrets(body) {
        level = api.SeverityCritical
    }
    emit(api.Finding{
        PluginName:  "paths", Level: level,
        Resource:    t.URL + sig.Path,
        Title:       fmt.Sprintf("Sensitive file accessible: %s", sig.Path),
        Mitigation:  sig.Mitigation,
        References:  sig.References,
    })
}
```

### 12.5 — Plugin Registration

Built-in plugins di-register di package `init`:

```go
// internal/plugin/registry/builtins.go
package registry

import (
    "github.com/1607-NetEnginee/NightCrawler/internal/plugins/paths"
    "github.com/1607-NetEnginee/NightCrawler/internal/plugins/dns"
    // ... 15 lainnya
)

func init() {
    Register(paths.New())
    Register(dns.New())
    // ...
}
```

User memilih plugin via:
```bash
nightcrawler scan --target example.com --plugins paths,dns,tls
nightcrawler scan --target example.com --profile aggressive  # plugin set dari profile YAML
nightcrawler scan --target example.com --exclude nikto       # semua kecuali nikto
```

### 12.6 — Remote Plugin (gRPC, v7.1+)

Untuk plugin pihak ketiga, sidecar gRPC:

```proto
service Plugin {
  rpc Manifest(google.protobuf.Empty) returns (ManifestResponse);
  rpc Run(RunRequest) returns (stream FindingEvent);
}
```

Plugin developer bisa pakai bahasa apa saja yang punya gRPC binding. Setup:

```bash
nightcrawler plugin install github.com/community/nuclei-bridge
# → unduh binary, verify sig, register di ~/.config/nightcrawler/plugins.d/
```

Plugin remote di-isolasi: namespace berbeda untuk filesystem, capability terbatas (lihat §23 Security Architecture).

---

## §13. Worker Pool Architecture

### 13.1 — Tiga Lapis Concurrency

```
Layer 1 — Target-level pool
   max_concurrent_targets (mis. 5)
   ↓ spawns
Layer 2 — Per-target plugin DAG executor
   max_concurrent_plugins_per_target (mis. 4)
   ↓ spawns
Layer 3 — Per-plugin probe pool
   max_concurrent_probes_per_plugin (mis. 20)
```

Total maksimum goroutine "berat" (network IO) ≈ 5 × 4 × 20 = 400. Aman untuk VPS 2 vCPU / 4 GB.

### 13.2 — Implementasi (sourcegraph/conc + semaphore)

```go
// Layer 1: target pool
targetPool := pool.New().WithMaxGoroutines(cfg.MaxTargets).WithContext(ctx)

for _, target := range targets {
    target := target
    targetPool.Go(func(ctx context.Context) error {
        return runTarget(ctx, target)
    })
}
targetPool.Wait()

// Layer 2: plugin DAG executor per target
func runTarget(ctx context.Context, t Target) error {
    dag := scheduler.BuildDAG(plugins)
    pluginPool := pool.New().WithMaxGoroutines(cfg.MaxPluginsPerTarget).WithContext(ctx)
    for _, batch := range dag.TopologicalLayers() {
        for _, p := range batch {
            p := p
            pluginPool.Go(func(ctx context.Context) error {
                return p.Run(ctx, t, emit)
            })
        }
        pluginPool.Wait()  // wait satu layer selesai sebelum next layer
    }
    return nil
}

// Layer 3: probe pool dalam tiap plugin
sem := semaphore.NewWeighted(int64(cfg.MaxProbesPerPlugin))
for _, probe := range probes {
    sem.Acquire(ctx, 1)
    go func() { defer sem.Release(1); doProbe(probe) }()
}
```

### 13.3 — Adaptive Concurrency

Worker pool sadar terhadap respons target:

- Default cap dari config.
- Pada N=10 request berturut yang return 429 atau 503 dalam window 30 detik → halve concurrency.
- Pada N=20 request 200 OK setelah throttle → recover gradually (multiplicative decrease + additive increase, mirip TCP).
- Hormati `Retry-After` header.

### 13.4 — Dependency Graph (DAG)

Tidak semua plugin independen. Contoh:
- `paths` butuh `tech-profile` (Laravel/WP detection).
- `webshell` butuh `catchall`.
- `attack-surface` butuh semua plugin lain.

Plugin declare via `DependsOn` di manifest. Scheduler topological sort:

```
Layer 0 (parallel):  dns, tls, headers, ports
Layer 1 (parallel):  tech-profile, catchall, cms, methods, cors
Layer 2 (parallel):  paths, webshell, gambling, redirect, disclosure
Layer 3 (parallel):  timing, sqli, xss
Layer 4:             attack-surface, report
```

Layer 0 jalan paralel, layer 1 menunggu layer 0, dst. Wall-clock time = sum of slowest plugin per layer, bukan sum of all plugins.

### 13.5 — Backpressure & Buffering

Event bus (channel findings) tidak boleh blok plugin. Strategi:
- Channel buffer = 1024.
- Jika penuh (aggregator lambat ke disk) → drop ke local buffer plugin + emit warning.
- Aggregator menulis NDJSON dengan `bufio.Writer` + periodic flush.

### 13.6 — Cancellation

Sinyal SIGINT → `cancel()` ke root context → semua goroutine yang `select` pada `ctx.Done()` exit clean. `trap cleanup INT TERM` v6.1 di-replicate sebagai signal handler Go:

```go
ctx, stop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
defer stop()
defer func() {
    if ctx.Err() != nil {
        report.FinalizePartial(ctx)
    }
}()
```

---

## §14. Distributed Scanning Concept (v7.1+)

### 14.1 — Use Case

- Audit multi-region: scan dari node US, EU, ID untuk membandingkan geo-blocking.
- Skalabilitas: 10.000 target → distribute ke 20 worker, masing-masing 500.
- Network diversity: tiap agent IP berbeda untuk menghindari rate limit single IP.

### 14.2 — Topologi

```
        ┌──────────────────────────┐
        │   Controller (1 node)    │
        │  - REST API + gRPC       │
        │  - Job queue (Redis)     │
        │  - Aggregator            │
        │  - Web dashboard         │
        └──────────┬───────────────┘
                   │ gRPC bi-streaming
        ┌──────────┴───────────────┐
        ▼          ▼               ▼
   ┌────────┐ ┌────────┐  ...  ┌────────┐
   │ Agent  │ │ Agent  │       │ Agent  │
   │ (Go    │ │ (Go    │       │ (Go    │
   │  bin)  │ │  bin)  │       │  bin)  │
   └────────┘ └────────┘       └────────┘
```

### 14.3 — Protocol (gRPC)

```proto
service Controller {
  rpc Register(AgentInfo) returns (RegisterResponse);
  rpc Heartbeat(stream HeartbeatMsg) returns (stream ControlMsg);
  rpc StreamFindings(stream Finding) returns (Ack);
}

service Agent {
  rpc AssignJob(Job) returns (JobAck);
  rpc CancelJob(JobID) returns (CancelAck);
}
```

### 14.4 — Job Distribution

Controller membagi target list per agent dengan strategi:
- **Round-robin** (default): target i → agent[i mod N].
- **Capacity-weighted:** agent dengan beban rendah dapat lebih banyak.
- **Geo-affinity:** target Asia → agent Asia.

Setiap job punya ID, retry count, deadline. Agent timeout → re-queue ke agent lain.

### 14.5 — Hasil Aggregation

Setiap agent stream finding via `StreamFindings`. Controller merge ke single NDJSON + DB (PostgreSQL untuk history). Web dashboard subscribe via WebSocket ke event stream.

### 14.6 — Deployment

```
docker run -d --name nc-controller \
    -p 8080:8080 -p 9090:9090 \
    nightcrawler:controller-v7.1

docker run -d --name nc-agent-1 \
    -e CONTROLLER_ADDR=controller.local:9090 \
    -e AGENT_TAGS=region=id,bandwidth=high \
    nightcrawler:agent-v7.1
```

Skala agent dengan K8s HPA atau systemd-nspawn cluster sederhana.

### 14.7 — Single-Node Default

**Penting:** Mode distributed adalah opt-in. Default `nightcrawler scan` adalah single-binary, single-host. Operator yang scan 1-10 target tidak perlu setup controller. Filosofi: "scale up only when needed".

---

## §15. Terminal UI Concept (BubbleTea + LipGloss)

### 15.1 — Layout Utama

```
┌─────────────────────────────────────────────────────────────────────────┐
│   ▓▓▓  NIGHTCRAWLER v7.0    ▓▓▓                       17:42:08          │
├─────────────────────────────────────────────────────────────────────────┤
│  TARGET   ▸ corp.example.com                                            │
│  PROFILE  ▸ default        OPERATOR ▸ 1607-NetEnginee    CLIENT ▸ Acme Corp   │
├──────────────────┬──────────────────────────────────────────────────────┤
│ MODULES          │  LIVE ACTIVITY                                       │
│                  │                                                      │
│ ◉ dns       100% │  [17:42:01] dns      ▸ resolved 23 subdomains       │
│ ◉ tls       100% │  [17:42:03] tls      ▸ TLS 1.3, valid cert (89 d)  │
│ ◉ headers   100% │  [17:42:04] headers  ▸ score 4/8 — HSTS missing    │
│ ◉ ports      87% │  [17:42:11] ports    ▸ probing 22/65535...         │
│ ● paths      42% │  [17:42:12] paths    ▸ .env not found (validated)  │
│ ○ webshell      │  [17:42:13] paths    ▸ /backup.sql → 403            │
│ ○ cms           │                                                      │
│ ○ gambling      │  ⚠ 1 CRITICAL  ⚠ 3 HIGH  ! 7 MEDIUM  · 2 LOW         │
│                  │                                                      │
├──────────────────┴──────────────────────────────────────────────────────┤
│ WORKERS  ║ ████████████████░░░░░░░░░░░░ 12/20 busy                      │
│ QUEUE    ║ ▓▓▓▓▓▓░░░░░░░░░░░░░░░░░░░░░░  124 pending                   │
│ RPS      ║ ▁▂▄▆█▆▄▂▁▂▄▆█▇▅▃ 38 r/s        ELAPSED  ║ 04:21              │
└─────────────────────────────────────────────────────────────────────────┘
  [q] quit   [p] pause   [f] findings   [l] log   [?] help
```

### 15.2 — Model (Bubble Tea)

```go
type Model struct {
    ctx          context.Context
    width, height int

    // panels
    header   HeaderPanel
    modules  ModulesPanel
    activity ActivityPanel
    stats    StatsPanel
    footer   FooterPanel

    // data
    findings  []api.Finding
    events    chan Event
    workerInfo WorkerInfo
}

func (m Model) Init() tea.Cmd {
    return tea.Batch(
        m.listenEvents(),
        m.tick(),
    )
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.resize(msg.Width, msg.Height)
    case tea.KeyMsg:
        return m.handleKey(msg)
    case EventMsg:
        return m.handleEvent(msg)
    case tickMsg:
        return m, m.tick()
    }
    return m, nil
}

func (m Model) View() string {
    return lipgloss.JoinVertical(lipgloss.Left,
        m.header.View(),
        lipgloss.JoinHorizontal(lipgloss.Top,
            m.modules.View(),
            m.activity.View()),
        m.stats.View(),
        m.footer.View(),
    )
}
```

### 15.3 — Theme (LipGloss)

```go
var Theme = struct {
    Primary, Secondary, Accent, Background lipgloss.AdaptiveColor
    Critical, High, Medium, Low, Info       lipgloss.AdaptiveColor
}{
    Primary:    lipgloss.AdaptiveColor{Light: "#5E2BFF", Dark: "#8B5CF6"}, // deep purple
    Secondary:  lipgloss.AdaptiveColor{Light: "#0066FF", Dark: "#3B82F6"}, // cyber blue
    Accent:     lipgloss.AdaptiveColor{Light: "#00BCD4", Dark: "#22D3EE"}, // neon cyan
    Background: lipgloss.AdaptiveColor{Light: "#0A0A0A", Dark: "#0A0A0A"},

    Info:     lipgloss.AdaptiveColor{Light: "#22D3EE", Dark: "#22D3EE"},
    Low:      lipgloss.AdaptiveColor{Light: "#10B981", Dark: "#34D399"},
    Medium:   lipgloss.AdaptiveColor{Light: "#F59E0B", Dark: "#FBBF24"},
    High:     lipgloss.AdaptiveColor{Light: "#EF4444", Dark: "#F87171"},
    Critical: lipgloss.AdaptiveColor{Light: "#C026D3", Dark: "#E879F9"},
}
```

### 15.4 — Komponen Reusable

- `SeverityBadge(level)` → rendered colored box dengan icon.
- `ProgressBar(pct, width)` → blockchar progress dengan gradient.
- `Sparkline(values[])` → RPS chart historis.
- `StatusDot(state)` → ●/◉/○ untuk pending/running/done.
- `Table(headers, rows)` → bordered table.

### 15.5 — Non-TUI Mode

Untuk CI/CD atau pipe ke file, flag `--no-tui` atau auto-detect (`!isatty(stdout)`):
- Output structured JSONL ke stdout.
- Tidak ada animasi, tidak ada cursor manipulation.
- Logs ke stderr dengan tint pretty handler.

### 15.6 — Startup Animation Ringan

Bukan matrix rain panjang. Cukup:
- 800ms fade-in banner.
- Lightweight glyph reveal (typewriter effect 30ms/char untuk versi & author).
- Tidak ada glitch_text yang berlebihan (v6.1 punya ini, dihilangkan).

Filosofi: **clean minimal, tidak norak** (sesuai brief).

---

## §16. ASCII Banner Modern (4 Variasi)

Banner harus: sederhana, modern, clean, profesional, tidak terlalu tinggi (max 8 baris), cocok untuk offensive security framework. Empat varian; default pilih satu, lainnya tersedia via `--banner=<name>`.

### 16.1 — Varian A: "Geometric" (Default)

```
 ███╗   ██╗ ██████╗
 ████╗  ██║██╔════╝   NIGHTCRAWLER  v7.0
 ██╔██╗ ██║██║        ─────────────────────────────────
 ██║╚██╗██║██║        Offensive Security Framework
 ██║ ╚████║╚██████╗   Author: 1607-NetEnginee · Cyberoutcast
 ╚═╝  ╚═══╝ ╚═════╝   ignored, but critical · 2026
```

Inspirasi: Nuclei, Naabu — typeface lebar block char, info block di samping kanan.

### 16.2 — Varian B: "Minimalist Slash"

```
 ╱╱╱  N I G H T C R A W L E R  ╱╱╱
      v7.0 · by 1607-NetEnginee · Cyberoutcast
```

2 baris. Untuk environment yang sangat kecil (CI logs).

### 16.3 — Varian C: "Sigil"

```
        ▄▄▄
      ▄█████▄         NIGHTCRAWLER
    ▄█████████▄       version 7.0
    █▀█████▀█         ────────────────────────────
    ▀  ▀▀▀  ▀         Advanced Offensive Platform
   ╱╲╱   ╲╱╲          1607-NetEnginee · 2026
```

Tetap spider-inspired tapi minimalist (vs v6.1 yang 25 baris).

### 16.4 — Varian D: "Bracket"

```
 ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
 ┃   N I G H T C R A W L E R   v 7 . 0                    ┃
 ┃   Offensive Security Framework · by 1607-NetEnginee          ┃
 ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

Untuk profile compliance / formal report header.

### 16.5 — Tagline & Versioning

Tagline tetap dari v6.1: **"ignored, but critical"** — frasa khas yang sudah jadi identitas project. Wajib pertahankan di setiap banner.

Format versi: `vMAJOR.MINOR.PATCH[-prerelease][+build]` (SemVer 2.0). Banner tampilkan `vMAJOR.MINOR` saja untuk kebersihan; `--version` flag tampilkan full string.

### 16.6 — Implementasi

Banner di-embed via `//go:embed banners/*.txt` di Go. Tidak ada interpolasi runtime kecuali version & author. Render dengan lipgloss untuk warna (deep purple → cyan gradient via 256-color ANSI).

---

## §17. Web Dashboard Concept (v7.1+)

### 17.1 — Filosofi

Dashboard adalah **read-only operator UI** untuk:
- Live monitoring scan yang sedang jalan.
- Browse scan history dengan filter.
- Asset inventory cross-target.
- Attack surface visualization.
- Export report.

**Bukan** RBAC tool, **bukan** vulnerability management platform, **bukan** ticketing. Untuk itu integrasi dengan DefectDojo/Jira via webhook.

### 17.2 — Stack

- Backend: Go (chi router, embed templ + htmx, WebSocket via nhooyr).
- Frontend: server-rendered templ + htmx untuk reactivity, tanpa SPA framework.
- Styling: Tailwind CSS dengan dark mode default; lucide-react icons (atau equivalent SVG inline).
- Charts: Chart.js atau ApexCharts via CDN (bisa di-self-host untuk airgap).

### 17.3 — Halaman Utama (Wireframe)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  NIGHTCRAWLER                                            [⚙] [👤 1607-NetEnginee] │
├────────────┬────────────────────────────────────────────────────────────────┤
│ ▣ Dashboard│  ┌────────────┬────────────┬────────────┬────────────┐        │
│ ◉ Scans    │  │  ACTIVE    │  TODAY     │  THIS WEEK │ ATTK SURF  │        │
│ ⊞ Assets   │  │     3      │    127     │    842     │   1,204    │        │
│ ◇ Findings │  │  scans     │  findings  │  findings  │   endpoints│        │
│ ⚑ Reports  │  └────────────┴────────────┴────────────┴────────────┘        │
│ ⚙ Settings │                                                                │
│            │  ── FINDINGS TIMELINE (7d) ──────────────────────────────────  │
│            │  ▁▃▆█▇▅▃▂▁▂▄▆█▇▅▃▂                                            │
│            │                                                                │
│            │  ── ACTIVE SCANS ────────────────────────────────────────────  │
│            │  corp.example.com    ████████████░░░░  67% · 12 findings  ⓘ   │
│            │  shop.example.com    ███████░░░░░░░░░  40% · 4 findings   ⓘ   │
│            │  api.example.com     █░░░░░░░░░░░░░░░  5%  · 0 findings   ⓘ   │
│            │                                                                │
│            │  ── TOP SEVERITY ────────────────────────────────────────────  │
│            │  ▲ CRITICAL  Webshell at shop.example.com/uploads/x.php       │
│            │  ▲ CRITICAL  SQLi at api.example.com/?id=                     │
│            │  ▲ HIGH      .env exposed at dev.corp.example.com             │
└────────────┴────────────────────────────────────────────────────────────────┘
```

### 17.4 — Realtime Updates

Setiap scan punya WebSocket endpoint `/ws/scans/{id}`. Client subscribe → server stream:
```json
{"type":"finding","data":{...}}
{"type":"progress","data":{"plugin":"paths","pct":42}}
{"type":"log","data":{"level":"info","msg":"..."}}
```

Frontend pakai htmx `hx-swap-oob` untuk update panel tanpa refresh page.

### 17.5 — Attack Surface View

Visualisasi graph:
- Root: target domain.
- Children: subdomain (warna by risk level).
- Edges: relationship (CNAME, A record, redirect).
- Click node → detail panel.

Library: `d3-force` atau `cytoscape.js` (CDN).

### 17.6 — Export

Setiap scan/report bisa di-export:
- **PDF** (P1): server-side render via `chromedp` headless Chrome.
- **HTML**: download self-contained.
- **JSON/NDJSON**: raw data.
- **SARIF**: untuk GitHub code scanning integration.
- **CSV**: untuk Excel pivot.

### 17.7 — Auth (Minimal)

v7.1 dashboard: single-user, password disetel di config + opsional TOTP.
v7.5+: OIDC support (Keycloak, Auth0), RBAC.

### 17.8 — Deployment

```bash
nightcrawler serve --bind :8080 --data-dir /var/lib/nightcrawler
```

Atau container Docker compose dengan PostgreSQL backend untuk history.

---

## §18. Reporting Engine Design

### 18.1 — Pipeline

```
Scan events → Aggregator → Findings DB (in-memory + NDJSON)
                                ↓
                       Report Builder
                          ↓        ↓        ↓        ↓        ↓
                       NDJSON    TXT      HTML    SARIF    PDF
                          ↓
                       Sinks: file, stdout, webhook, S3, syslog
```

NDJSON adalah **format kanonik**. Semua format lain di-derive dari NDJSON, bukan dari findings in-memory. Ini memungkinkan re-generate report dari file lama:

```bash
nightcrawler report render --input scan-20260514.ndjson --format html -o report.html
```

### 18.2 — Finding Schema (NDJSON Event)

```json
{
  "schema": "nightcrawler.io/v1/finding",
  "id": "f-3a2c8b1e",
  "scan_id": "s-2026-05-14-1742-abc",
  "timestamp": "2026-05-14T17:42:13.842Z",
  "plugin": "paths",
  "plugin_version": "1.0.0",
  "level": "high",
  "title": "Sensitive file accessible: /.env",
  "resource": {
    "url": "https://corp.example.com/.env",
    "method": "GET",
    "status_code": 200
  },
  "target": {
    "domain": "corp.example.com",
    "ip": "203.0.113.42"
  },
  "evidence": [
    {
      "type": "response_excerpt",
      "data": "APP_KEY=base64:....\nDB_HOST=10.0.0.5\n..."
    }
  ],
  "validation": {
    "catchall_filtered": true,
    "content_validated": true,
    "tech_filter_applied": "laravel"
  },
  "mitigation": {
    "id": "DENY-ENV",
    "title_id": "Tutup akses file .env",
    "title_en": "Block public access to .env",
    "steps_id": ["Tambahkan rule di nginx: location ~ /\\.env { deny all; }", "..."],
    "steps_en": ["Add nginx rule: location ~ /\\.env { deny all; }", "..."]
  },
  "references": [
    {"type": "cwe", "id": "CWE-538", "url": "https://cwe.mitre.org/data/definitions/538.html"},
    {"type": "owasp", "id": "A05:2021", "title": "Security Misconfiguration"}
  ],
  "tags": ["sensitive-file", "credential-leak", "laravel"],
  "risk_score": 8.7
}
```

### 18.3 — Template-Based HTML Report

`internal/report/template/report.html.tmpl`:
- Embedded via `//go:embed`.
- Self-contained: CSS, fonts (JetBrains Mono), JS inline atau via embed.
- Sections: Executive Summary, Risk Gauge, Findings by Severity, Asset Inventory, Detailed Findings (collapsible), Methodology, Appendix.
- Print-friendly stylesheet.
- Dark mode default, toggle ke light.
- Interactive: filter, search, copy-as-curl untuk evidence.

### 18.4 — Multi-Target Aggregate Report

Untuk client dengan multi-domain (Cyberoutcast use case), report multi-target:
- Halaman pertama: cross-target summary.
- Per-target deep dive.
- Cross-target patterns (mis. "5/7 domain klien missing HSTS").

### 18.5 — Locale (i18n)

Mitigasi disimpan bilingual di YAML signature DB:
```yaml
mitigations:
  DENY-ENV:
    title:
      id: "Tutup akses file .env"
      en: "Block public access to .env"
    steps:
      id:
        - "Tambahkan rule nginx: ..."
      en:
        - "Add nginx rule: ..."
```

Flag `--locale=id|en` (default: auto-detect dari `$LANG`).

### 18.6 — Reproducibility

Report mengandung metadata:
- Scanner version + commit SHA.
- Plugin versions yang dipakai.
- Signature DB hash.
- Config snapshot.

Sehingga finding dari scan 6 bulan lalu bisa di-reproduce / di-audit.

---

## §19. AI Integration Concept (P2, v7.5+)

### 19.1 — Use Case Realistic (Bukan Hype)

AI di v7.x **bukan** "AI replaces pentester". AI di v7.x adalah:
1. **Mitigation copywriting:** Generate mitigasi natural-language dari rule ID + evidence.
2. **Risk re-scoring:** Adjust severity berdasarkan business context (mis. PCI scope, regulated industry).
3. **False positive triage:** Klasifikasi finding sebagai likely-FP berdasarkan pattern.
4. **Evidence summarization:** Kumpulkan 50 finding sejenis → 1 kalimat exec summary.
5. **Q&A on report:** "Apa 3 prioritas terbesar?" → LLM jawab dari NDJSON.

### 19.2 — Arsitektur

```
NDJSON findings ──→ AI Enrichment Service ──→ Enriched NDJSON
                          │
                          ├─ Local: Ollama (llama3.1, qwen2.5) - default
                          ├─ OpenAI-compatible API (groq, fireworks)
                          └─ Anthropic Claude (via API key)
```

Provider abstraksi via `LLMProvider` interface. Local-first (Ollama) untuk privacy. Cloud opt-in.

### 19.3 — Privacy Pertanyaan

Finding dapat berisi credential leaked. **Wajib** ada filter:
- Strip evidence body dari secret value sebelum kirim ke LLM eksternal.
- Hanya kirim metadata (level, title, plugin, CWE).
- User opt-in eksplisit per scan.
- Local provider sebagai default.

### 19.4 — Output

AI hasil disimpan sebagai field tambahan di NDJSON:
```json
{
  "ai_enrichment": {
    "model": "llama3.1:8b",
    "summary_id": "Endpoint debug dibiarkan terbuka di production...",
    "summary_en": "Debug endpoint left open in production...",
    "fp_probability": 0.05,
    "business_risk_adjusted": "high",
    "remediation_complexity": "low"
  }
}
```

Original finding tidak diubah; enrichment additif. Operator bisa toggle on/off di report viewer.

### 19.5 — Q&A Mode

```bash
nightcrawler ask --scan s-2026-05-14-1742-abc "Apa 3 prioritas terbesar?"
```

Loader baca NDJSON, build retrieval index (BM25 atau embeddings), LLM jawab dengan citation ke finding ID.

---

## §20. CI/CD Strategy

### 20.1 — Pipeline

GitHub Actions, 4 workflow:

**a) `.github/workflows/ci.yml`** — pada setiap PR & push ke main:
- Setup Go 1.22+
- `go mod verify`
- `golangci-lint run`
- `gofumpt -l -d .` (fail jika ada diff)
- `go test ./... -race -count=1 -coverprofile=cover.out`
- `go test -bench=. -benchmem ./...` (smoke benchmark, fail jika regress > 30%)
- Build matrix: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
- Upload coverage ke Codecov

**b) `.github/workflows/codeql.yml`** — security scan weekly + per PR:
- GitHub CodeQL untuk Go.
- `gosec` untuk security-specific Go issues.
- `trivy` untuk dependency CVE.
- `syft` SBOM upload sebagai artifact.

**c) `.github/workflows/release.yml`** — pada tag `v*`:
- Run goreleaser:
  - Cross-compile binary (amd64, arm64) untuk Linux/macOS.
  - Build multi-arch Docker image (linux/amd64, linux/arm64).
  - Push ke `ghcr.io/1607-NetEnginee/NightCrawler:vX.Y.Z` + `:latest`.
  - Generate changelog dari conventional commits.
  - Sign artifact dengan cosign (keyless via OIDC).
  - Generate SBOM dengan syft.
  - Buat GitHub Release dengan checksum.

**d) `.github/workflows/docs.yml`** — pada push ke main + tag:
- Build dokumentasi MkDocs.
- Deploy ke GitHub Pages.

### 20.2 — Branch Strategy

- `main`: stable, selalu releasable.
- `develop`: integration branch (opsional, bila tim besar).
- `feat/*`, `fix/*`, `chore/*`: feature branches.
- Tag `v*` untuk release.
- Pre-release tag `v*-rc.N`, `v*-beta.N`.

PR ke `main` butuh:
- 1 approving review (CODEOWNERS auto-assign).
- CI hijau.
- Conventional commit subject.
- Linked issue.

### 20.3 — Versioning

SemVer 2.0:
- **MAJOR:** breaking API change (plugin interface change).
- **MINOR:** new plugin, new flag, new output format.
- **PATCH:** bug fix, signature DB update.

`v7.0.0` GA, lalu monthly minor (`v7.1.0`, `v7.2.0`), weekly patch bila ada signature DB update.

### 20.4 — Dependabot

`.github/dependabot.yml`:
- Weekly check go modules.
- Monthly check Docker base image.
- Auto-merge security patch jika tests pass dan no breaking change (via PR auto-merge).

### 20.5 — Pre-commit

`.pre-commit-config.yaml`:
- gofumpt
- golangci-lint
- gitleaks (no secret in commit)
- conventional commit linter

---

## §21. Docker Strategy

### 21.1 — Multi-Stage Dockerfile

```dockerfile
# syntax=docker/dockerfile:1.7

# ───── Stage 1: builder ─────
FROM golang:1.22-alpine AS builder
WORKDIR /src
ENV CGO_ENABLED=0 GOOS=linux

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags " \
      -s -w \
      -X github.com/1607-NetEnginee/NightCrawler/internal/version.Version=${VERSION} \
      -X github.com/1607-NetEnginee/NightCrawler/internal/version.Commit=${COMMIT} \
      -X github.com/1607-NetEnginee/NightCrawler/internal/version.BuildDate=${BUILD_DATE}" \
    -o /out/nightcrawler ./cmd/nightcrawler

# ───── Stage 2: distroless runtime ─────
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="nightcrawler"
LABEL org.opencontainers.image.description="Offensive Security Framework"
LABEL org.opencontainers.image.source="https://github.com/1607-NetEnginee/NightCrawler"
LABEL org.opencontainers.image.licenses="Apache-2.0"

COPY --from=builder /out/nightcrawler /usr/local/bin/nightcrawler
COPY --from=builder /src/signatures /etc/nightcrawler/signatures
COPY --from=builder /src/configs/nightcrawler.example.yaml /etc/nightcrawler/config.yaml

USER nonroot:nonroot
WORKDIR /work
ENTRYPOINT ["/usr/local/bin/nightcrawler"]
CMD ["--help"]
```

### 21.2 — Image Sizes (Target)

- Builder stage: ~700 MB (ephemeral).
- Final image: ≤ 25 MB (distroless static + binary + signature YAML).
- Konsisten dengan filosofi compact v6.1 (44 KB tarball).

### 21.3 — Multi-Arch

Build untuk `linux/amd64` dan `linux/arm64` (Raspberry Pi, AWS Graviton, M1 server). Via `docker buildx` di CI:

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --push \
  -t ghcr.io/1607-NetEnginee/NightCrawler:v7.0.0 \
  -t ghcr.io/1607-NetEnginee/NightCrawler:latest \
  .
```

### 21.4 — docker-compose Dev Stack

`deployments/docker/docker-compose.yaml`:
```yaml
services:
  nightcrawler:
    image: ghcr.io/1607-NetEnginee/NightCrawler:latest
    volumes:
      - ./reports:/work/reports
      - ./config.yaml:/etc/nightcrawler/config.yaml:ro
    networks: [nc-net]
    command: ["scan", "--config", "/etc/nightcrawler/config.yaml"]

  # v7.1+: controller mode
  controller:
    image: ghcr.io/1607-NetEnginee/NightCrawler-controller:latest
    ports: ["8080:8080", "9090:9090"]
    environment:
      DATABASE_URL: postgres://nc:nc@db:5432/nc
    depends_on: [db]
    networks: [nc-net]

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: nc
      POSTGRES_PASSWORD: nc
      POSTGRES_DB: nc
    volumes: [pgdata:/var/lib/postgresql/data]
    networks: [nc-net]

volumes:
  pgdata:

networks:
  nc-net:
```

### 21.5 — Capability Requirements

Distroless image **tidak butuh** root. Untuk port scan SYN (jika operator pilih), butuh `CAP_NET_RAW`:
```bash
docker run --cap-add=NET_RAW nightcrawler:v7.0.0 scan ...
```

Default TCP connect scan tidak butuh capability tambahan.

### 21.6 — Image Signing

```bash
cosign sign --yes ghcr.io/1607-NetEnginee/NightCrawler@sha256:...
```

Verifikasi:
```bash
cosign verify --certificate-identity-regexp '.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  ghcr.io/1607-NetEnginee/NightCrawler:v7.0.0
```

---

## §22. GitHub Release Workflow

### 22.1 — goreleaser Config (`.goreleaser.yaml`)

```yaml
version: 2

before:
  hooks:
    - go mod tidy
    - go generate ./...

builds:
  - id: nightcrawler
    main: ./cmd/nightcrawler
    binary: nightcrawler
    env: [CGO_ENABLED=0]
    goos:   [linux, darwin]
    goarch: [amd64, arm64]
    flags: [-trimpath]
    ldflags:
      - -s -w
      - -X github.com/1607-NetEnginee/NightCrawler/internal/version.Version={{.Version}}
      - -X github.com/1607-NetEnginee/NightCrawler/internal/version.Commit={{.Commit}}
      - -X github.com/1607-NetEnginee/NightCrawler/internal/version.BuildDate={{.Date}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - LICENSE
      - README.md
      - signatures/**/*
      - configs/nightcrawler.example.yaml

checksum:
  name_template: "checksums.txt"
  algorithm: sha256

signs:
  - cmd: cosign
    args: ["sign-blob", "--yes", "--output-signature=${signature}", "${artifact}"]
    artifacts: checksum

sboms:
  - artifacts: archive

dockers:
  - image_templates:
      - "ghcr.io/1607-NetEnginee/NightCrawler:{{ .Version }}-amd64"
    dockerfile: deployments/docker/Dockerfile
    use: buildx
    build_flag_templates: [--platform=linux/amd64]
  - image_templates:
      - "ghcr.io/1607-NetEnginee/NightCrawler:{{ .Version }}-arm64"
    dockerfile: deployments/docker/Dockerfile
    use: buildx
    build_flag_templates: [--platform=linux/arm64]
    goarch: arm64

docker_manifests:
  - name_template: "ghcr.io/1607-NetEnginee/NightCrawler:{{ .Version }}"
    image_templates:
      - "ghcr.io/1607-NetEnginee/NightCrawler:{{ .Version }}-amd64"
      - "ghcr.io/1607-NetEnginee/NightCrawler:{{ .Version }}-arm64"
  - name_template: "ghcr.io/1607-NetEnginee/NightCrawler:latest"
    image_templates:
      - "ghcr.io/1607-NetEnginee/NightCrawler:{{ .Version }}-amd64"
      - "ghcr.io/1607-NetEnginee/NightCrawler:{{ .Version }}-arm64"

changelog:
  sort: asc
  use: github
  groups:
    - title: "Features"
      regexp: '^.*?feat(\(.+\))??!?:.+$'
    - title: "Bug fixes"
      regexp: '^.*?fix(\(.+\))??!?:.+$'
    - title: "Performance"
      regexp: '^.*?perf(\(.+\))??!?:.+$'
    - title: "Security"
      regexp: '^.*?sec(\(.+\))??!?:.+$'

release:
  github:
    owner: 1607-NetEnginee
    name: nightcrawler
  prerelease: auto
  header: |
    ## NIGHTCRAWLER {{ .Tag }}

    Offensive Security Framework — by 1607-NetEnginee / Cyberoutcast

    **Quickstart:**
    ```bash
    curl -sSL https://raw.githubusercontent.com/1607-NetEnginee/NightCrawler/main/scripts/install.sh | bash
    ```
  footer: |
    **Verification:**
    ```bash
    cosign verify-blob --signature checksums.txt.sig checksums.txt
    sha256sum -c checksums.txt
    ```

brews:
  - name: nightcrawler
    homepage: https://github.com/1607-NetEnginee/NightCrawler
    description: "Offensive Security Framework"
    license: "Apache-2.0"
    repository:
      owner: 1607-NetEnginee
      name: homebrew-tap
```

### 22.2 — Release Cadence

- **Patch:** ad-hoc, dalam jam jika ada security fix.
- **Minor:** monthly, fitur baru + signature pack update.
- **Major:** annual atau breaking API change.

### 22.3 — Pre-Release

Setiap minor punya seri RC sebelum GA:
- `v7.1.0-rc.1`, `rc.2`, ...
- RC duduk di branch `release-v7.1` minimal 7 hari sebelum tag final.
- Operator early-adopter di-tag di issue tracker untuk testing.

### 22.4 — Hotfix

Branch `hotfix/v7.0.x` dari tag terakhir. Cherry-pick fix, tag patch, release. Tidak butuh tunggu cycle minor.

---

## §23. Security Architecture (Scanner Hardening)

### 23.1 — Threat Model

**Yang DILINDUNGI:**
- Operator credential (Telegram token, API key, dst).
- Output (findings yang berisi data sensitif target).
- Supply chain (binary, plugin, signature pack).

**Asumsi attacker:**
- Pasif: bisa baca file di disk shared/backup.
- Aktif: bisa MITM jaringan operator.
- Supply chain: bisa typosquat package atau compromise dependency.
- Target: bisa craft response yang exploit parser (ReDoS, decompression bomb).

### 23.2 — Mitigasi Per Threat

| Threat | Mitigasi v7.0 |
|---|---|
| Root execution risk | Default run as non-root. SYN scan via capability `cap_net_raw` (opt-in flag). Drop privileges di main()`. |
| Path traversal via input | Whitelist regex untuk client name, target. Output path resolved + checked terhadap base dir. |
| Plain text credential | Secret dari env var atau `~/.config/nightcrawler/secrets.age` (encrypted via `age`). Tidak dari argv (visible di `ps`). |
| Output secret leakage | Opsi `--encrypt-output` dengan `age` recipient. Default mode 0600. |
| MITM operator-target | TLS verify default ON. Flag `--insecure` butuh `--i-know-what-i-am-doing` opt-in. |
| Supply chain | Binary di-sign cosign. Docker image SBOM via syft. Plugin verifikasi via signature ed25519. |
| ReDoS dari target | Body cap `io.LimitReader(resp.Body, 5*1024*1024)`. Regex dengan complexity scoring sebelum execute. |
| Decompression bomb | Disable auto-decompress untuk content > 50 MB. Stream decompress dengan size cap. |
| HTML parser exploit | goquery dengan node count limit. |
| External tool exploitation | Tool eksternal (nmap, nikto) opt-in plugin. Run dengan absolute path. Output di-parse di sandbox goroutine. |
| Telegram token exposure | Token via env var atau secret store. Tidak pernah di-log. Redact otomatis di log handler. |
| Plugin malicious | Plugin remote run di sandbox: namespace berbeda, no filesystem write di luar /tmp, network egress filter |

### 23.3 — Defense in Depth

```
Layer 1: Distroless container (no shell, no apt, no curl)
Layer 2: Non-root user dalam container
Layer 3: Read-only filesystem (kecuali /work/reports)
Layer 4: Capabilities minimum (drop all kecuali yang dibutuhkan)
Layer 5: Network egress allowlist (mis. hanya target + crt.sh + signature update server)
Layer 6: Seccomp profile (restrict syscall)
Layer 7: AppArmor / SELinux policy (opsional)
```

K8s manifest contoh:
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: [ALL]
    add: [NET_RAW]   # opsional, hanya bila SYN scan
  seccompProfile:
    type: RuntimeDefault
```

### 23.4 — Secret Redaction di Log

Custom slog handler yang strip pattern:
- `password=`, `token=`, `api_key=`, `secret=`, `Bearer ...`
- Base64 string > 32 char di field "evidence".
- Email + IP (opsional, untuk audit privacy).

### 23.5 — Audit Log

Setiap action operator-significant (start scan, change config, install plugin) tulis ke `/var/log/nightcrawler/audit.log` (atau journald) dengan format:
```json
{"ts":"...","actor":"1607-netengineee","action":"scan.start","target":"corp.example.com","scan_id":"s-..."}
```

Append-only, syslog-shippable, tamper-evident hash chain (P2).

### 23.6 — Vulnerability Disclosure

`SECURITY.md` di repo:
- Email: security@cyberoutcast.id (atau GitHub Security Advisory).
- PGP key.
- 90-day disclosure timeline.
- Bounty (jika ada budget).

### 23.7 — License & Compliance

- License: Apache 2.0 (permissive, kompatibel dengan komersial).
- Notice file dengan attribution dependency.
- Contributor License Agreement opsional.
- Tidak include payload yang ilegal di yurisdiksi target (sqlmap payload tetap, tapi documented sebagai authorized use only).

---

## §24. Scaling Strategy

### 24.1 — Vertical Scaling (Single Node)

Target hardware single-node:
- 2 vCPU / 2 GB RAM: 1-3 target concurrent, full scan.
- 4 vCPU / 4 GB RAM: 5-10 target concurrent (sweet spot VPS).
- 16 vCPU / 16 GB RAM: 30+ target concurrent.

Konfigurasi auto-tune:
```yaml
concurrency:
  max_targets: auto                    # = NumCPU()
  max_plugins_per_target: auto         # = max(4, NumCPU()/2)
  max_probes_per_plugin: 20
  http_connections_per_host: 10
  global_rps_cap: 200
```

### 24.2 — Horizontal Scaling (Distributed)

v7.1+: controller + N agents (lihat §14).
- Linear scale-out hingga ~100 agents (controller bottleneck di aggregator).
- Untuk skala >100 agent: shard controller per-region atau per-client.

### 24.3 — Memory Profile

Target memory footprint single scan 1 target:
- Idle: 30-50 MB.
- Active (10 plugin concurrent): 100-300 MB.
- Peak (gobuster equivalent dengan wordlist 50k): 500-700 MB.

Hindari load entire wordlist ke memory: stream from file, scan dengan `bufio.Scanner`.

### 24.4 — Disk Profile

Per-scan disk usage:
- NDJSON event stream: ~500 KB - 5 MB.
- HTML report: ~200 KB - 2 MB.
- Raw artifacts (nmap output, nikto output): ~1-10 MB.
- Target total per scan per domain: ~5-20 MB.

Retention policy (config-driven):
```yaml
retention:
  keep_raw_artifacts_days: 30
  keep_reports_days: 365
  compress_after_days: 7
```

### 24.5 — Network Profile

Per-target full scan:
- DNS: 50-200 queries.
- HTTP: 500-2000 requests.
- Bandwidth: 5-50 MB (kebanyakan HTTP HEAD + small body sample).

RPS cap (default 200 global, 10 per-host) untuk menghormati target dan rate limiter.

### 24.6 — Database (P1+)

PostgreSQL untuk:
- Scan history.
- Findings searchable.
- Asset inventory.

Schema sederhana:
```sql
CREATE TABLE scans (id UUID PRIMARY KEY, started_at TIMESTAMPTZ, ...);
CREATE TABLE findings (
  id UUID PRIMARY KEY, scan_id UUID REFERENCES scans(id),
  level VARCHAR, plugin VARCHAR, resource TEXT,
  data JSONB,
  hash CHAR(64),  -- deduplication
  created_at TIMESTAMPTZ
);
CREATE INDEX ON findings (scan_id, level);
CREATE INDEX ON findings USING GIN (data);
```

Partitioning by month untuk retention efisien.

---

## §25. Deployment Strategy

### 25.1 — Skenario Deployment

| Skenario | Topologi | Target Pengguna |
|---|---|---|
| **Solo operator** | Binary di laptop / VPS | 1607-NetEnginee, freelance pentester |
| **Tim kecil** | Single VPS dengan docker-compose | Cyberoutcast team |
| **Tim besar / MSSP** | Controller + 5 agents (K8s) | Enterprise SOC |
| **Air-gapped** | Standalone offline | Government / regulated |
| **CI/CD pipeline** | Container in GitHub Actions / GitLab CI | DevSecOps |

### 25.2 — Installer Script

`scripts/install.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail

VERSION="${NC_VERSION:-latest}"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
URL="https://github.com/1607-NetEnginee/NightCrawler/releases/download/${VERSION}/nightcrawler_${VERSION}_${OS}_${ARCH}.tar.gz"

# Verify checksum
CHECKSUM_URL="https://github.com/1607-NetEnginee/NightCrawler/releases/download/${VERSION}/checksums.txt"
curl -fsSL "$CHECKSUM_URL" -o checksums.txt
curl -fsSL "$URL" -o nightcrawler.tar.gz
grep "$(basename "$URL")" checksums.txt | sha256sum -c -

tar -xzf nightcrawler.tar.gz
sudo install -m 0755 nightcrawler /usr/local/bin/
echo "✓ NIGHTCRAWLER installed: $(nightcrawler --version)"
```

Verifikasi cosign signature jika cosign tersedia.

### 25.3 — systemd Unit (Distributed Agent)

`deployments/systemd/nightcrawler-agent.service`:
```ini
[Unit]
Description=Nightcrawler Agent
After=network-online.target

[Service]
Type=simple
User=nightcrawler
Group=nightcrawler
ExecStart=/usr/local/bin/nightcrawler-agent serve \
  --controller=controller.example.com:9090 \
  --tags=region=id,env=prod
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/var/lib/nightcrawler
ProtectHome=true
AmbientCapabilities=CAP_NET_RAW

[Install]
WantedBy=multi-user.target
```

### 25.4 — Kubernetes (Helm Chart)

`deployments/kubernetes/helm/`:
```yaml
# values.yaml
controller:
  enabled: true
  replicas: 1
  resources:
    requests: { cpu: 200m, memory: 256Mi }
    limits:   { cpu: 1000m, memory: 1Gi }

agents:
  replicas: 3
  resources:
    requests: { cpu: 500m, memory: 512Mi }
    limits:   { cpu: 2000m, memory: 2Gi }
  nodeSelector:
    nightcrawler.io/agent: "true"

postgresql:
  enabled: true
  auth: { username: nc, database: nc }

ingress:
  enabled: true
  host: nightcrawler.example.com
```

### 25.5 — Air-Gapped Deployment

Bundle:
- Single binary.
- Signature DB (versioned).
- Docker image tarball (`docker save`).
- Offline documentation (PDF).
- Checksums.

Update mechanism: USB drop dengan signature pack baru. Binary verifies signature lokal (no internet).

### 25.6 — CI/CD Integration

GitHub Actions example:
```yaml
- name: Run NIGHTCRAWLER
  uses: 1607-NetEnginee/NightCrawler-action@v7
  with:
    target: ${{ secrets.STAGING_URL }}
    profile: stealth
    fail-on: critical
    output-format: sarif
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: nightcrawler-output.sarif
```

GitLab CI example:
```yaml
nightcrawler:
  image: ghcr.io/1607-NetEnginee/NightCrawler:latest
  script:
    - nightcrawler scan --target $TARGET --profile stealth --format sarif -o nc.sarif
  artifacts:
    reports: { sast: nc.sarif }
```

---

## §26. Modern YAML Config Example

`~/.config/nightcrawler/config.yaml` (full annotated):

```yaml
# ════════════════════════════════════════════════════════════════════
# NIGHTCRAWLER v7.0 — Configuration
# Docs: https://github.com/1607-NetEnginee/NightCrawler/blob/main/docs/CONFIGURATION.md
# ════════════════════════════════════════════════════════════════════
apiVersion: nightcrawler.io/v1
kind: Configuration

# ── Identity ────────────────────────────────────────────────────────
identity:
  operator: 1607-NetEnginee
  organization: Cyberoutcast
  contact: ops@cyberoutcast.id

# ── Output ──────────────────────────────────────────────────────────
output:
  base_dir: ${HOME}/nightcrawler-reports
  format: [ndjson, html, sarif]              # legacy: txt
  filename_template: "{date}_{target}_{scan_id}"
  encrypt:
    enabled: false
    recipient: ""                            # age public key
  retention:
    keep_raw_artifacts_days: 30
    keep_reports_days: 365
    compress_after_days: 7

# ── HTTP Engine ─────────────────────────────────────────────────────
http:
  timeout: 15s
  max_redirects: 5
  max_body_size: 5MiB
  user_agent_pool: pool://default            # built-in pool atau path file
  follow_redirects: true
  proxy: ""                                  # http://127.0.0.1:8080 untuk Burp
  tls:
    verify: true
    min_version: "1.2"
  connection_pool:
    max_per_host: 10
    idle_timeout: 90s
  rate_limit:
    global_rps: 200
    per_host_rps: 10
    adaptive: true                           # backoff pada 429/503
  stealth:
    enabled: true
    jitter_min: 0.3s
    jitter_max: 0.8s
    rotate_user_agent: true
    rotate_referer: true
    sec_fetch_headers: true

# ── DNS ─────────────────────────────────────────────────────────────
dns:
  resolvers: [system, 1.1.1.1, 8.8.8.8]
  timeout: 3s
  retries: 1
  parallel: 50
  bruteforce:
    wordlist: builtin                        # atau path
    max_concurrent: 50

# ── Concurrency ─────────────────────────────────────────────────────
concurrency:
  max_targets: auto                          # NumCPU()
  max_plugins_per_target: auto
  max_probes_per_plugin: 20

# ── Plugins ─────────────────────────────────────────────────────────
plugins:
  enabled:                                   # urutan tidak penting (DAG)
    - dns
    - tls
    - headers
    - ports
    - paths
    - webshell
    - cms
    - methods
    - cors
    - gambling
    - redirect
    - disclosure
    - timing
    - xss
    - sqli
    - crtsh
  disabled:
    - nikto                                  # opt-in
    - sqlmap-deep
  config:
    paths:
      max_concurrent_probes: 20
      include_id_gov_edu: true
    crtsh:
      timeout: 30s
      cache_ttl: 24h
    webshell:
      paranoid: false                        # noisy probe

# ── Signature Database ──────────────────────────────────────────────
signatures:
  base_dir: /etc/nightcrawler/signatures
  auto_update: false                         # opt-in
  update_url: https://signatures.cyberoutcast.id/v7
  verify_signature: true

# ── Reporting ───────────────────────────────────────────────────────
report:
  locale: id                                 # id | en | auto
  include_evidence: true
  redact_secrets: true                       # mask token, password value
  html:
    theme: dark
    print_friendly: true
    self_contained: true

# ── Notifications ───────────────────────────────────────────────────
notify:
  on:
    - scan_complete
    - severity: [critical, high]
  channels:
    telegram:
      enabled: false
      token: ${TELEGRAM_TOKEN}               # dari env
      chat_id: "-100123456789"
    slack:
      enabled: false
      webhook: ${SLACK_WEBHOOK}
    syslog:
      enabled: false
      address: localhost:514
      facility: local0

# ── Telemetry ───────────────────────────────────────────────────────
telemetry:
  log_level: info                            # debug, info, warn, error
  log_format: tint                           # tint | json
  metrics:
    enabled: false
    bind: :9091
  tracing:
    enabled: false
    otlp_endpoint: localhost:4317

# ── Profiles (preset bundles) ───────────────────────────────────────
profiles:
  stealth:
    http: { stealth: { jitter_max: 2s }, rate_limit: { global_rps: 30 } }
    plugins: { disabled: [webshell, sqli, xss, nikto] }
  aggressive:
    http: { rate_limit: { global_rps: 500 } }
    plugins: { enabled: ["+nikto", "+sqlmap-deep"] }
  quick:
    plugins: { enabled: [dns, tls, headers] }
  compliance:
    plugins: { enabled: [tls, headers, methods, cors, disclosure] }
    report: { format: [html, sarif, pdf] }
```

Validasi via JSONSchema. CLI: `nightcrawler config validate`.

---

## §27. Modern JSON Output Example (NDJSON Stream)

Setiap baris adalah event terpisah. Stream-friendly untuk pipe ke `jq`, `vector`, atau SIEM agent.

```jsonl
{"schema":"nightcrawler.io/v1/scan_start","scan_id":"s-2026-05-14-1742-3a2c","timestamp":"2026-05-14T17:42:00.000Z","operator":"1607-NetEnginee","organization":"Cyberoutcast","targets":["corp.example.com"],"plugins_enabled":["dns","tls","headers","ports","paths","webshell","cms","methods","cors","gambling","redirect","disclosure","timing","xss","sqli","crtsh"],"profile":"default","scanner_version":"7.0.0","scanner_commit":"a1b2c3d"}
{"schema":"nightcrawler.io/v1/target_start","scan_id":"s-2026-05-14-1742-3a2c","target":"corp.example.com","timestamp":"2026-05-14T17:42:00.500Z"}
{"schema":"nightcrawler.io/v1/plugin_start","scan_id":"s-2026-05-14-1742-3a2c","plugin":"dns","plugin_version":"1.0.0","target":"corp.example.com","timestamp":"2026-05-14T17:42:00.501Z"}
{"schema":"nightcrawler.io/v1/finding","id":"f-3a2c8b1e","scan_id":"s-2026-05-14-1742-3a2c","timestamp":"2026-05-14T17:42:13.842Z","plugin":"paths","plugin_version":"1.0.0","level":"high","title":"Sensitive file accessible: /.env","resource":{"url":"https://corp.example.com/.env","method":"GET","status_code":200},"target":{"domain":"corp.example.com","ip":"203.0.113.42"},"evidence":[{"type":"response_excerpt","data":"APP_KEY=base64:****REDACTED****\nDB_HOST=10.0.0.5\n"}],"validation":{"catchall_filtered":true,"content_validated":true,"tech_filter_applied":"laravel"},"mitigation":{"id":"DENY-ENV","title_id":"Tutup akses file .env","title_en":"Block public access to .env","steps_id":["Tambahkan nginx rule: location ~ /\\.env { deny all; }"],"steps_en":["Add nginx rule: location ~ /\\.env { deny all; }"]},"references":[{"type":"cwe","id":"CWE-538","url":"https://cwe.mitre.org/data/definitions/538.html"},{"type":"owasp","id":"A05:2021","title":"Security Misconfiguration"}],"tags":["sensitive-file","credential-leak","laravel"],"risk_score":8.7}
{"schema":"nightcrawler.io/v1/plugin_end","scan_id":"s-2026-05-14-1742-3a2c","plugin":"paths","target":"corp.example.com","duration_ms":1842,"findings_count":3,"timestamp":"2026-05-14T17:42:15.343Z"}
{"schema":"nightcrawler.io/v1/target_end","scan_id":"s-2026-05-14-1742-3a2c","target":"corp.example.com","duration_ms":124000,"findings_summary":{"critical":1,"high":3,"medium":7,"low":2},"risk_score":68,"risk_level":"high","timestamp":"2026-05-14T17:44:04.500Z"}
{"schema":"nightcrawler.io/v1/scan_end","scan_id":"s-2026-05-14-1742-3a2c","duration_ms":124500,"targets_scanned":1,"findings_total":13,"timestamp":"2026-05-14T17:44:05.000Z"}
```

### 27.1 — Konsumsi Hilir

**Filter critical only:**
```bash
jq -c 'select(.schema=="nightcrawler.io/v1/finding" and .level=="critical")' scan.ndjson
```

**Pipe ke SIEM (Vector):**
```toml
[sources.nc]
type = "file"
include = ["/var/log/nightcrawler/*.ndjson"]
[sinks.elastic]
type = "elasticsearch"
inputs = ["nc"]
endpoints = ["https://elastic:9200"]
```

**Aggregate per plugin:**
```bash
jq -s 'map(select(.schema=="nightcrawler.io/v1/finding")) | group_by(.plugin) | map({plugin: .[0].plugin, count: length})' scan.ndjson
```

---

## §28. Example HTML Report Structure

Single-page, self-contained HTML (~150-300 KB termasuk embedded CSS+JS+font). Struktur:

```
┌─ <header>: Banner + meta (operator, client, scan_id, timestamp, scanner version)
│
├─ <section id="exec-summary">: Executive Summary
│    ├─ Risk gauge (SVG, 0-100, animated)
│    ├─ Severity counter cards (Critical/High/Medium/Low)
│    ├─ Top 3 priority findings
│    └─ Recommendation TL;DR (1 paragraph, locale-aware)
│
├─ <section id="overview">: Scope & Methodology
│    ├─ Targets table (domain, IP, accessibility)
│    ├─ Plugins executed (with versions)
│    ├─ Profile used + config snapshot hash
│    └─ Scan duration breakdown
│
├─ <section id="attack-surface">: Attack Surface
│    ├─ Subdomain inventory (filterable table)
│    ├─ Open ports (per host)
│    ├─ Tech stack detected
│    └─ Trust boundaries diagram (SVG, derived from data)
│
├─ <section id="findings">: Detailed Findings
│    ├─ Filter bar (severity, plugin, tag, search)
│    ├─ Sort: severity desc | risk score desc | resource asc
│    └─ Finding cards (collapsible):
│         ├─ Severity badge + plugin name + timestamp
│         ├─ Title
│         ├─ Resource (URL with copy-as-curl button)
│         ├─ Evidence (redacted)
│         ├─ Validation provenance
│         ├─ Mitigation (bilingual toggle)
│         └─ References (CWE/CVE/OWASP links)
│
├─ <section id="appendix">: Appendix
│    ├─ Raw artifact list (download links)
│    ├─ Plugin manifest list
│    ├─ Glossary
│    └─ Methodology disclaimer
│
└─ <footer>: Generated by NIGHTCRAWLER v7.0 · 1607-NetEnginee · 2026
```

Interaktivitas:
- Filter/search dengan vanilla JS (tidak butuh framework).
- Collapsible via `<details>` HTML5 native.
- Print stylesheet auto-flatten collapsible.
- Dark mode toggle dengan `prefers-color-scheme` default.

Template engine: Go `html/template` dengan partial composition. Tested rendering dengan corpus 1000 findings tetap < 2 detik render.

---

## §29. Example Modern Terminal Output (Non-TUI Mode)

Untuk pipeline atau CI logs (no animation, structured prose). Default jika `!isatty(stdout)`:

```
$ nightcrawler scan --target corp.example.com --profile default

  ╱╱╱  NIGHTCRAWLER v7.0 · 1607-NetEnginee · Cyberoutcast  ╱╱╱

  scan_id      s-2026-05-14-1742-3a2c
  operator     1607-NetEnginee
  organization Cyberoutcast
  target       corp.example.com
  profile      default
  plugins      16 enabled
  output       ~/nightcrawler-reports/corp.example.com_20260514_174200/

  ────────────────────────────────────────────────────────────────────
  17:42:00  start    target=corp.example.com
  17:42:00  dns      ▸ resolving corp.example.com
  17:42:01  dns      ▸ A 203.0.113.42
  17:42:01  dns      ▸ enumerating subdomains (80 builtin + crt.sh)
  17:42:08  dns      ▸ 23 active subdomains found (15 brute + 8 passive)
  17:42:08  tls      ▸ TLS 1.3, cert valid (89 days remaining)
  17:42:09  tls      ▸ cipher suites: 4 supported, all modern
  17:42:10  ports    ▸ probing 30 common ports (TCP connect)
  17:42:14  ports    ▸ open: 22, 80, 443, 8080
  17:42:14  HIGH     ▸ ports: port 8080 exposed publicly (Tomcat?)
  17:42:14  headers  ▸ scoring security headers
  17:42:15  MEDIUM   ▸ headers: HSTS missing
  17:42:15  MEDIUM   ▸ headers: CSP missing
  17:42:15  MEDIUM   ▸ headers: X-Frame-Options missing
  17:42:15  techprof ▸ detected: Laravel (high confidence)
  17:42:16  catchall ▸ no soft-404 detected
  17:42:16  paths    ▸ probing 90 sensitive paths (laravel-filtered)
  17:42:23  HIGH     ▸ paths: https://corp.example.com/.env (200, validated)
  17:42:24  paths    ▸ /storage/logs/laravel.log → 403
  17:42:25  paths    ▸ /backup.sql → 404
  17:42:28  paths    ▸ 1 finding (1 high)
  17:42:28  webshell ▸ probing 30 webshell patterns
  17:42:34  webshell ▸ no webshells detected ✓
  17:42:34  cms      ▸ Laravel (already detected)
  ...
  17:44:04  end      target=corp.example.com  duration=124s
  ────────────────────────────────────────────────────────────────────

  RISK SCORE         68 / 100  (HIGH)
  ─────────────────────────────────
   CRITICAL           1
   HIGH               3
   MEDIUM             7
   LOW                2
   total findings    13

  TOP PRIORITIES
  ──────────────
  [1] CRITICAL  Webshell at https://corp.example.com/uploads/x.php
  [2] HIGH      .env file exposed (Laravel credentials)
  [3] HIGH      Port 8080 exposed publicly (Tomcat manager?)

  REPORT
  ──────
  ~/nightcrawler-reports/corp.example.com_20260514_174200/
    ├── scan.ndjson         (canonical event stream)
    ├── REPORT.html         (interactive report)
    ├── REPORT.sarif        (SARIF 2.1.0)
    └── raw/                (per-plugin artifacts)
```

Karakteristik:
- Setiap baris timestamped, structured, grep-able.
- Severity label inline (mudah `grep -E "CRITICAL|HIGH"`).
- Tidak ada animasi, tidak ada cursor manipulation.
- Tetap visual hierarchy via spacing dan separator.
- Footer dengan TL;DR ringkas + file paths.

---

## §30. Long-Term Maintainability Strategy

### 30.1 — Pillars

1. **Modular boundary terjaga.** Plugin interface stable, breaking change butuh MAJOR. Internal package fluid.
2. **Domain knowledge sebagai data, bukan kode.** Signature DB, mitigasi, wordlist semua di YAML. PR ke signature pack tidak butuh Go review.
3. **Test coverage ≥ 75%** untuk `internal/core`, `internal/validator`, `internal/plugins/*` (kritis).
4. **Documentation as code.** Setiap fitur PR butuh update docs di same commit.
5. **Linter dijaga ketat.** golangci-lint dengan profile strict, no `nolint` tanpa comment justifikasi.
6. **Dependency hygiene.** Monthly review, drop unused, prefer stdlib.

### 30.2 — Cadence

| Aktivitas | Frekuensi |
|---|---|
| Signature DB update | Weekly (otomatis dari CVE feed + community PR) |
| Patch release | Ad-hoc (jam-level untuk security) |
| Minor release | Monthly |
| Major release | Annual (atau breaking) |
| Dependency audit | Monthly |
| Documentation review | Quarterly |
| Architecture decision review | Bi-annual |

### 30.3 — Governance

- **Maintainer:** 1607-NetEnginee (BDFL) + 1-2 maintainer dipilih dari kontributor aktif.
- **Triage rotation:** 1 maintainer per week handle issue & PR.
- **Decision process:**
  - Bug fix → langsung PR.
  - Minor feature → issue + 1 maintainer approve.
  - Breaking change → ADR (Architecture Decision Record) di `docs/adr/`.

### 30.4 — Onboarding Kontributor

- `CONTRIBUTING.md` lengkap dengan dev setup, test command, commit convention.
- `make dev` setup local env dalam < 5 menit.
- `examples/custom-plugin/` template untuk plugin baru.
- `good-first-issue` labels di-curate aktif.
- Discord/Matrix channel untuk diskusi cepat (opsional).

### 30.5 — Bus Factor Mitigation

Risiko terbesar single-author project: bus factor = 1. Mitigasi:
- Dokumen seperti yang Anda baca ini (analisa + redesign). Knowledge externalized.
- CI/CD otomatis, tidak ada step manual di laptop maintainer.
- Signing key di GitHub OIDC (keyless), bukan key file di laptop.
- Build reproducible (deterministic): siapa saja bisa rebuild binary identik dari source.
- Documented release process di `docs/RELEASE.md`.

### 30.6 — Deprecation Policy

- Fitur deprecated diumumkan di release notes minor.
- Warning di runtime selama minimum 2 minor release (≈ 2 bulan).
- Removal di major release berikutnya.
- Migration guide wajib untuk deprecated → replacement.

### 30.7 — Long-Term Roadmap (Tentative)

| Versi | Tema | Target |
|---|---|---|
| **v7.0** | Core rewrite | Paritas dengan v6.1 + concurrency + plugin |
| **v7.1** | TUI + Distributed | Bubble Tea UI + gRPC agent |
| **v7.2** | Web Dashboard | Read-only operator UI |
| **v7.3** | HTTP/3 + Headless | Modern web target support |
| **v7.4** | Encrypted output | Privacy hardening |
| **v7.5** | AI enrichment | Local LLM mitigation generator |
| **v7.6** | Plugin marketplace | Community ecosystem |
| **v8.0** | Plugin SDK rewrite | Breaking change bila perlu, dari lesson learned |

Tidak ada janji deadline; cadence "monthly minor" jadi guideline, bukan pressure.

### 30.8 — Success Metrics

- **Adoption:** ≥ 500 GitHub stars dalam 12 bulan.
- **Contribution:** ≥ 10 external contributor dalam 12 bulan.
- **Reliability:** zero P0 bug bertahan > 7 hari.
- **Performance:** full-scan 1 target rata-rata ≤ 3 menit (vs 8-15 menit v6.1).
- **Footprint:** binary ≤ 30 MB, container ≤ 25 MB, memory peak ≤ 700 MB.
- **Maintainability:** lead time PR merge median ≤ 3 hari.

---

# PART III — PENUTUP

## Ringkasan Eksekutif

NIGHTCRAWLER v6.1 sudah membawa identitas yang kuat: **defensive-first**, **Indonesia-aware**, **operator-centric**, **compact**. Validation layer-nya (catch-all, tech profile, density check, IP differential) adalah **aset diferensiasi nyata** yang sulit dicari di tool internasional. Wordlist gov/edu Indonesia adalah moat tambahan.

Kelemahan v6.1 (monolit Bash, sequential, hardcoded, tidak machine-friendly) bukan soal "kurang fitur" — semuanya soal **arsitektur yang tidak skalabel**. Tambah modul ke v6.1 hanya akan memperburuk technical debt.

NIGHTCRAWLER v7.0 yang diusulkan:
1. **Mempertahankan 100% domain knowledge v6.1** (validasi, wordlist, mitigasi bilingual, defensive posture).
2. **Memindahkan domain knowledge dari kode ke data** (YAML signature DB).
3. **Mengganti shell-out hell dengan native Go HTTP/DNS/TLS engine.**
4. **Memperkenalkan 3-layer concurrency** untuk 5-10× speedup wall-clock.
5. **Modular plugin architecture** untuk community contribution dan maintainability.
6. **NDJSON kanonik** untuk integrasi SIEM dan pipeline.
7. **Single binary distribution** konsisten dengan filosofi compact v6.1.
8. **Cloud + container + distributed-ready** tanpa overengineering — semua opt-in.
9. **Dashboard, TUI, AI, marketplace** masuk roadmap tapi **bukan** prasyarat GA.

Timeline realistic v7.0 GA: **8-12 minggu** dengan satu maintainer fokus, atau 4-6 minggu dengan 2-3 kontributor.

Langkah berikutnya: **scaffold skeleton Go project** sesuai struktur §10, dengan **kerangka kode yang siap dibangun**, **YAML config jadi**, **Dockerfile multi-stage**, **GitHub Actions CI/CD**, **goreleaser config**, dan **dokumentasi awal**. Itulah yang akan saya kerjakan di Part berikutnya.

---

**End of Analysis & Redesign Document**

*Author: 1607-NetEnginee · Cyberoutcast · 2026*
*"ignored, but critical"*
