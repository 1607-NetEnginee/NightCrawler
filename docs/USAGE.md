# Panduan Penggunaan NIGHTCRAWLER v7.0

> Dokumen ini menjelaskan cara menggunakan NIGHTCRAWLER setelah instalasi selesai.

---

## Daftar Isi

- [Verifikasi Instalasi](#verifikasi-instalasi)
- [Konsep Dasar](#konsep-dasar)
- [Perintah Pertama](#perintah-pertama)
- [Scan Command](#scan-command)
- [Profile Scan](#profile-scan)
- [Plugin](#plugin)
- [Output & Laporan](#output--laporan)
- [Multi Target](#multi-target)
- [Penggunaan di CI/CD](#penggunaan-di-cicd)
- [Tips & Trik](#tips--trik)

---

## Verifikasi Instalasi

Setelah install, pastikan binary berjalan:

```bash
nightcrawler version
nightcrawler --help
```

---

## Konsep Dasar

NIGHTCRAWLER bekerja dengan cara:

1. Kamu memberi **target** (domain atau URL)
2. Tool menjalankan **plugin** secara paralel terhadap target
3. Hasil ditulis sebagai **laporan** (HTML, NDJSON, SARIF, TXT)
4. Temuan dikategorikan berdasarkan **severity**: `critical`, `high`, `medium`, `low`, `info`

> ⚠️ **Etika:** Hanya scan target yang kamu miliki atau yang kamu punya izin tertulis untuk diuji.

---

## Perintah Pertama

### 1. Generate konfigurasi default

```bash
nightcrawler config init
```

File config akan dibuat di `~/.config/nightcrawler/config.yaml`.

### 2. Smoke test (scan cepat)

```bash
nightcrawler scan -t example.com --profile quick
```

### 3. Lihat hasilnya

Laporan disimpan di `~/nightcrawler-reports/` dalam format HTML dan NDJSON.

---

## Scan Command

Perintah utama adalah `nightcrawler scan`. Berikut flag yang tersedia:

| Flag | Shorthand | Default | Keterangan |
|---|---|---|---|
| `--target` | `-t` | — | Target domain/URL (bisa diulang) |
| `--targets-file` | — | — | File berisi daftar target, satu per baris |
| `--profile` | `-p` | `default` | Profile scan |
| `--plugins` | — | — | Plugin spesifik yang dijalankan |
| `--exclude` | — | — | Plugin yang dikecualikan |
| `--output` | `-o` | `~/nightcrawler-reports` | Direktori output laporan |
| `--format` | `-f` | `ndjson,html` | Format laporan |
| `--fail-on` | — | `high` | Severity minimum untuk exit non-zero |
| `--operator` | — | `$USER` | Nama operator (untuk audit trail) |
| `--client` | — | — | Nama klien (untuk laporan profesional) |
| `--dry-run` | — | `false` | Tampilkan rencana scan tanpa mengirim probe |
| `--timeout` | — | unlimited | Batas waktu keseluruhan scan |
| `--concurrency` | — | auto | Jumlah target yang diproses paralel |

### Contoh penggunaan:

```bash
# Scan satu target, profile default
nightcrawler scan -t corp.example.com

# Scan dengan nama klien (untuk laporan)
nightcrawler scan -t corp.example.com --client "PT Contoh Indonesia"

# Scan stealth (pelan, tidak mencolok)
nightcrawler scan -t corp.example.com --profile stealth

# Pilih plugin tertentu saja
nightcrawler scan -t corp.example.com --plugins dns,tls,headers

# Kecualikan plugin tertentu
nightcrawler scan -t corp.example.com --exclude sqli,xss

# Dry run — lihat rencana tanpa scan sungguhan
nightcrawler scan -t corp.example.com --dry-run

# Simpan laporan ke folder tertentu
nightcrawler scan -t corp.example.com -o ./hasil-scan

# Output dalam format SARIF (untuk integrasi IDE/CI)
nightcrawler scan -t corp.example.com -f sarif,html
```

---

## Profile Scan

Profile menentukan kecepatan, kedalaman, dan plugin yang dijalankan.

| Profile | Kecepatan | Kedalaman | Cocok untuk |
|---|---|---|---|
| `quick` | ⚡ Cepat | Dangkal | Cek cepat, smoke test |
| `stealth` | 🐢 Lambat | Sedang | Target sensitif, hindari deteksi |
| `default` | 🚶 Normal | Penuh | Penggunaan sehari-hari |
| `aggressive` | 🏃 Cepat | Dalam | Lab, target sendiri |
| `compliance` | 🚶 Normal | Penuh + SARIF | Audit, CI/CD gate |

```bash
nightcrawler scan -t target.com --profile stealth
nightcrawler scan -t target.com --profile aggressive
nightcrawler scan -t target.com --profile compliance
```

---

## Plugin

NIGHTCRAWLER memiliki 17 plugin bawaan.

### Lihat daftar semua plugin

```bash
nightcrawler plugin list
```

### Lihat detail satu plugin

```bash
nightcrawler plugin info dns
nightcrawler plugin info tls
nightcrawler plugin info sqli
```

### Daftar Plugin

| Nama | Kategori | Keterangan |
|---|---|---|
| `dns` | Recon | Enumerasi DNS, bruteforce subdomain |
| `tls` | Crypto | Verifikasi sertifikat, cipher lemah |
| `headers` | Hardening | Security headers (CSP, HSTS, dll) |
| `ports` | Network | Port scan dasar |
| `paths` | Discovery | File/path sensitif, termasuk path gov/edu Indonesia |
| `cms` | Fingerprint | Deteksi CMS (WordPress, Joomla, dll) |
| `methods` | Hardening | HTTP method berbahaya (PUT, DELETE, dll) |
| `cors` | Misconfiguration | CORS misconfiguration |
| `gambling` | Custom | Deteksi injeksi konten judi |
| `redirect` | Vuln | Open redirect |
| `disclosure` | Vuln | Info disclosure (error, stack trace, dll) |
| `timing` | Vuln | Timing-based detection |
| `xss` | Vuln | Cross-Site Scripting |
| `sqli` | Vuln | SQL Injection |
| `webshell` | Vuln | Deteksi webshell |
| `crtsh` | Recon | Passive recon via crt.sh |
| `nikto` | External | Nikto (opt-in, noisy) |

---

## Output & Laporan

### Format yang tersedia

```bash
# HTML (default, mudah dibaca browser)
nightcrawler scan -t target.com -f html

# NDJSON (untuk pipeline/SIEM)
nightcrawler scan -t target.com -f ndjson

# SARIF (untuk integrasi IDE, GitHub Code Scanning)
nightcrawler scan -t target.com -f sarif

# TXT (plain text)
nightcrawler scan -t target.com -f txt

# Kombinasi
nightcrawler scan -t target.com -f ndjson,html,sarif
```

### Lokasi laporan

Secara default laporan disimpan di:
```
~/nightcrawler-reports/<tanggal>_<target>_<scan_id>/
```

Ganti lokasi dengan flag `-o`:
```bash
nightcrawler scan -t target.com -o ./laporan-klien
```

---

## Multi Target

### Dari flag (beberapa target sekaligus)

```bash
nightcrawler scan -t target1.com -t target2.com -t target3.com
```

### Dari file

Buat file `targets.txt`:
```
target1.com
target2.com
https://api.target3.com
```

Lalu jalankan:
```bash
nightcrawler scan --targets-file targets.txt --profile stealth
```

---

## Penggunaan di CI/CD

NIGHTCRAWLER bisa dipakai sebagai **security gate** di pipeline CI:

```bash
# Gagalkan build kalau ada temuan HIGH atau CRITICAL
nightcrawler scan -t $TARGET \
  --profile compliance \
  --format ndjson,sarif \
  --fail-on high \
  -o ./scan-results
```

Exit codes:
- `0` — scan selesai, tidak ada temuan di atas threshold
- `1` — ada temuan di atas threshold `--fail-on`
- `2` — error konfigurasi atau jaringan

---

## Tips & Trik

### Proxy ke Burp Suite

```bash
# Edit config
nightcrawler config init
# Set http.proxy di config.yaml:
# http:
#   proxy: "http://127.0.0.1:8080"
```

### Pin versi installer

```bash
NC_VERSION=v7.0.0 curl -sSL \
  https://raw.githubusercontent.com/1607-NetEnginee/NightCrawler/main/scripts/install.sh | bash
```

### Skip verifikasi checksum (tidak disarankan)

```bash
NC_SKIP_VERIFY=1 curl -sSL \
  https://raw.githubusercontent.com/1607-NetEnginee/NightCrawler/main/scripts/install.sh | bash
```

### Install ke direktori custom

```bash
NC_INSTALL_DIR=/opt/tools curl -sSL \
  https://raw.githubusercontent.com/1607-NetEnginee/NightCrawler/main/scripts/install.sh | bash
```

---

*Dokumentasi ini bagian dari [NightCrawler v7.0](https://github.com/1607-NetEnginee/NightCrawler) oleh 1607-NetEnginee / Cyberoutcast.*
