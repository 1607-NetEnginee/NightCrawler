# NIGHTCRAWLER v7.0

> Offensive Security Framework — *"ignored, but critical"*
>
> oleh **1607-NetEnginee** / **Cyberoutcast**

[![ci](https://github.com/1607-NetEnginee/NightCrawler/actions/workflows/ci.yml/badge.svg)](https://github.com/1607-NetEnginee/NightCrawler/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/1607-NetEnginee/NightCrawler)](https://github.com/1607-NetEnginee/NightCrawler/releases)
[![license](https://img.shields.io/github/license/1607-NetEnginee/NightCrawler)](LICENSE)

NIGHTCRAWLER adalah framework offensive security yang modular dan berbasis plugin, ditulis ulang dalam Go. Ini adalah penerus produksi dari framework Bash v6.1 — re-arsitektur menyeluruh yang **mempertahankan** lima iterasi rekayasa *false-positive*, pengetahuan path pemerintah/pendidikan Indonesia, dan panduan mitigasi dwibahasa yang menjadi ciri khas v6.x, sambil **menghilangkan** mesin monolitik, sekuensial, dan boros subprocess yang membatasi versi sebelumnya.

Dokumen desain lengkap: [`docs/ANALYSIS_AND_REDESIGN.md`](docs/ANALYSIS_AND_REDESIGN.md) — audit reverse engineering v6.1 dan rasionalisasi arsitektur v7.0.

---

## Sorotan Utama

- **Satu binary statis tunggal.** Tanpa dependency runtime. ≤ 30 MB.
- **Concurrency 3 lapis.** Target-level, plugin DAG-level, per-plugin probe-level.
- **17 plugin bawaan**: DNS, TLS, headers, ports, file sensitif, fingerprint CMS, methods, CORS, deteksi inject judi, open redirect, info disclosure, timing, XSS, SQLi, webshell, recon pasif crt.sh.
- **Lapisan validasi terjaga.** Catch-all, filter path berbasis tech profile, IP differential cerdas, density check judi — semua hasil lima iterasi v6.1 dipindahkan menjadi YAML signature pack yang data-driven.
- **NDJSON kanonik.** Event stream cocok pipe ke SIEM (Vector, Fluent Bit, Logstash). HTML, SARIF, TXT, PDF di-derive dari NDJSON.
- **Mitigasi dwibahasa.** Setiap temuan menyertakan panduan remediasi dalam Bahasa Indonesia dan Inggris.
- **Berbasis profile.** `stealth`, `default`, `aggressive`, `quick`, `compliance` — ganti dengan satu flag.
- **Cloud-native.** Docker distroless. Helm chart untuk K8s. Mendukung deployment air-gapped.
- **Postur defensif.** Non-root by default. Verifikasi TLS aktif default. Rate limiter adaptif yang menghormati `Retry-After`.

---

## Instalasi

### Installer satu baris

```bash
curl -sSL https://raw.githubusercontent.com/1607-NetEnginee/NightCrawler/main/scripts/install.sh | bash
```

Installer memverifikasi SHA-256 checksum dan signature Sigstore dari artifact release sebelum melakukan instalasi.

### Unduh manual

Ambil archive yang sesuai dari [halaman releases](https://github.com/1607-NetEnginee/NightCrawler/releases). Binary bersifat portable — letakkan di mana saja di `$PATH`.

### Docker

```bash
> ⚠️ Docker image belum tersedia di versi ini.
```

### Build dari source

```bash
git clone https://github.com/1607-NetEnginee/NightCrawler
cd nightcrawler
make build
./bin/nightcrawler --help
```

---

## Mulai Cepat
## Mode Interaktif (nightcrawler-ui)

Untuk penggunaan yang dipandu, ramah pemula, dengan laporan HTML otomatis dan export PDF:

```bash
curl -sSL https://raw.githubusercontent.com/1607-NetEnginee/NightCrawler/main/scripts/nightcrawler-ui \
  -o nightcrawler-ui && chmod +x nightcrawler-ui && sudo mv nightcrawler-ui /usr/local/bin/
```

Jalankan:
```bash
nightcrawler-ui
```

Alur: **wizard input → scan → laporan HTML interaktif → export PDF**

```bash
# Smoke test
nightcrawler scan -t example.com --profile quick

# Scan default
nightcrawler scan -t corp.example.com --client "Acme Corp"

# Multi-target dari file, profile stealth
nightcrawler scan --targets-file scope.txt --profile stealth -o ./reports

# Pilih plugin tertentu
nightcrawler scan -t example.com --plugins dns,tls,headers,paths

# Gerbang CI: gagal build kalau ada temuan HIGH atau CRITICAL
nightcrawler scan -t $TARGET --profile compliance \
  --format ndjson,sarif --fail-on high -o ./out
```

---

## Migrasi dari v6.x

```bash
# 1. Pasang v7.0 berdampingan dengan v6.1 (nama binary berbeda)
# 2. Konversi konfigurasi v6.1 lama
nightcrawler config import-v6 /etc/nightcrawler/nightcrawler.conf \
  --output ~/.config/nightcrawler/config.yaml

# 3. Validasi
nightcrawler config validate

# 4. Dry-run untuk konfirmasi cakupan
nightcrawler scan --targets-file scope.txt --dry-run

# 5. Jalankan sebenarnya
nightcrawler scan --targets-file scope.txt
```

Panduan migrasi lengkap: [`docs/MIGRATION_V6_TO_V7.md`](docs/MIGRATION_V6_TO_V7.md).

---

## Keamanan & Etika

NIGHTCRAWLER memindai sistem yang menjadi target Anda. Jalankan **hanya** terhadap aset yang Anda miliki atau yang Anda punya **izin tertulis** untuk diuji. Penulis dan maintainer tidak bertanggung jawab atas penyalahgunaan.

Laporkan kerentanan sesuai panduan di [`SECURITY.md`](SECURITY.md). Mohon **jangan** membuka isu publik di GitHub untuk laporan keamanan.

---

## Lisensi

Apache 2.0. Lihat [`LICENSE`](LICENSE).

---

*Dibuat dengan saksama oleh 1607-NetEnginee / Cyberoutcast. Bila NIGHTCRAWLER membantu pekerjaan Anda, beri ⭐ di repo — gratis dan membantu visibilitas project.*
