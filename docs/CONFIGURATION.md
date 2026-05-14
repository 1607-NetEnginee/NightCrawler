# Konfigurasi NIGHTCRAWLER v7.0

> Panduan lengkap semua opsi konfigurasi di `~/.config/nightcrawler/config.yaml`

---

## Daftar Isi

- [Generate Config](#generate-config)
- [Struktur Config](#struktur-config)
- [Identity](#identity)
- [Output](#output)
- [HTTP Engine](#http-engine)
- [DNS](#dns)
- [Concurrency](#concurrency)
- [Plugins](#plugins)
- [Report](#report)
- [Notifikasi](#notifikasi)
- [Telemetry](#telemetry)
- [Environment Variables](#environment-variables)

---

## Generate Config

```bash
# Buat config default
nightcrawler config init

# Validasi config yang ada
nightcrawler config validate

# Lihat lokasi config aktif
nightcrawler config path
```

File config berada di: `~/.config/nightcrawler/config.yaml`

---

## Struktur Config

```yaml
apiVersion: nightcrawler.io/v1
kind: Configuration

identity:    # Identitas operator
output:      # Pengaturan laporan
http:        # HTTP engine
dns:         # DNS resolver
concurrency: # Parallelism
plugins:     # Plugin aktif/nonaktif
report:      # Format laporan
notify:      # Notifikasi
telemetry:   # Log & metrics
```

---

## Identity

Informasi operator yang muncul di laporan dan audit trail.

```yaml
identity:
  operator: NamaKamu        # default: $USER
  organization: Cyberoutcast
  contact: email@kamu.com
```

---

## Output

```yaml
output:
  base_dir: ${HOME}/nightcrawler-reports   # direktori output laporan
  format: [ndjson, html, sarif]            # format default
  filename_template: "{date}_{target}_{scan_id}"
  retention:
    keep_raw_artifacts_days: 30
    keep_reports_days: 365
    compress_after_days: 7
```

**Format yang tersedia:** `ndjson`, `html`, `sarif`, `txt`

---

## HTTP Engine

Pengaturan koneksi HTTP ke target.

```yaml
http:
  timeout: 15s              # timeout per request
  max_redirects: 5
  max_body_size: 5MiB
  follow_redirects: true
  proxy: ""                 # contoh: "http://127.0.0.1:8080" untuk Burp Suite

  tls:
    verify: true            # set false hanya untuk lab
    min_version: "1.2"

  rate_limit:
    global_rps: 200         # request per detik (global)
    per_host_rps: 10        # request per detik per host
    adaptive: true          # otomatis backoff saat kena 429/503

  stealth:
    enabled: true
    jitter_min: 0.3s        # jeda minimum antar request
    jitter_max: 0.8s        # jeda maksimum antar request
    rotate_user_agent: true # rotasi user-agent otomatis
    rotate_referer: true
    sec_fetch_headers: true
```

> 💡 **Tip Burp Suite:** Set `proxy: "http://127.0.0.1:8080"` dan `tls.verify: false` untuk intercept traffic di Burp.

---

## DNS

```yaml
dns:
  resolvers: [system, 1.1.1.1, 8.8.8.8]  # resolver yang dipakai
  timeout: 3s
  retries: 1
  parallel: 50                             # query paralel

  bruteforce:
    wordlist: builtin                      # atau path ke wordlist custom
    max_concurrent: 50
```

---

## Concurrency

```yaml
concurrency:
  max_targets: auto          # jumlah target paralel (auto = NumCPU)
  max_plugins_per_target: auto
  max_probes_per_plugin: 20
```

Untuk mesin dengan RAM terbatas, turunkan nilainya:
```yaml
concurrency:
  max_targets: 2
  max_plugins_per_target: 4
  max_probes_per_plugin: 10
```

---

## Plugins

```yaml
plugins:
  enabled:
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
    - nikto          # noisy, aktifkan manual kalau perlu
    - sqlmap-deep    # noisy, aktifkan manual kalau perlu

  config:
    paths:
      max_concurrent_probes: 20
      include_id_gov_edu: true   # aktifkan path database gov/edu Indonesia

    crtsh:
      timeout: 30s
      cache_ttl: 24h

    webshell:
      paranoid: false            # set true untuk probe lebih agresif (noisy)
```

### Aktifkan plugin via flag (tanpa edit config)

```bash
# Jalankan plugin tertentu saja
nightcrawler scan -t target.com --plugins dns,tls,headers

# Kecualikan plugin tertentu
nightcrawler scan -t target.com --exclude nikto,sqli
```

---

## Report

```yaml
report:
  locale: id                  # id = Bahasa Indonesia, en = English, auto = deteksi sistem
  include_evidence: true      # sertakan bukti/request di laporan
  redact_secrets: true        # sensor token/password di laporan

  html:
    theme: dark               # dark | light
    print_friendly: true
    self_contained: true      # HTML berdiri sendiri (semua aset inline)
```

> 💡 `locale: id` akan menampilkan panduan mitigasi dalam **Bahasa Indonesia**.

---

## Notifikasi

Kirim notifikasi saat scan selesai atau ada temuan kritis.

```yaml
notify:
  on:
    - scan_complete
    - severity: [critical, high]

  channels:
    telegram:
      enabled: false
      token: ${TELEGRAM_TOKEN}        # pakai environment variable
      chat_id: "-100123456789"

    slack:
      enabled: false
      webhook: ${SLACK_WEBHOOK}

    syslog:
      enabled: false
      address: localhost:514
      facility: local0
```

### Cara aktifkan notifikasi Telegram

```bash
# Set token via environment variable (jangan hardcode di config!)
export TELEGRAM_TOKEN=123456:ABCDEFxxx
export TELEGRAM_CHAT_ID=-100123456789

# Edit config
nightcrawler config init
# Set notify.channels.telegram.enabled: true
# Set notify.channels.telegram.chat_id: "-100123456789"
```

---

## Telemetry

```yaml
telemetry:
  log_level: info             # debug | info | warn | error
  log_format: tint            # tint (warna di terminal) | json (untuk log aggregator)

  metrics:
    enabled: false
    bind: :9091               # Prometheus metrics endpoint

  tracing:
    enabled: false
    otlp_endpoint: localhost:4317
```

Untuk debug masalah:
```bash
# Jalankan dengan log level debug
nightcrawler scan -t target.com  # lalu set log_level: debug di config
```

---

## Environment Variables

Variabel environment yang dikenali NIGHTCRAWLER:

| Variable | Keterangan |
|---|---|
| `NIGHTCRAWLER_CONFIG` | Path ke file config custom |
| `TELEGRAM_TOKEN` | Token bot Telegram untuk notifikasi |
| `SLACK_WEBHOOK` | Webhook URL Slack |
| `NC_VERSION` | Pin versi saat install via script |
| `NC_INSTALL_DIR` | Direktori instalasi (default: `/usr/local/bin`) |
| `NC_SKIP_VERIFY` | Set `1` untuk skip verifikasi checksum saat install |

---

*Dokumentasi ini bagian dari [NightCrawler v7.0](https://github.com/1607-NetEnginee/NightCrawler) oleh 1607-NetEnginee / Cyberoutcast.*
