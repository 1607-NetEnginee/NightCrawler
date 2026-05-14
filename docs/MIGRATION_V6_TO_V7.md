# Migrasi dari v6.x ke v7.0

> Panduan lengkap migrasi dari NIGHTCRAWLER v6.1 (Bash) ke v7.0 (Go).

---

## Perbedaan Utama v6.x vs v7.0

| Aspek | v6.x | v7.0 |
|---|---|---|
| Bahasa | Bash | Go |
| Binary | Script + dependencies | Single static binary |
| Concurrency | Sequential | 3-layer parallel |
| Config | `.conf` (shell vars) | YAML |
| Output | TXT | NDJSON, HTML, SARIF, TXT |
| Plugin | Hardcoded | Modular DAG |

---

## Langkah Migrasi

### 1. Install v7.0 berdampingan

```bash
# v7.0 binary bernama 'nightcrawler' (sama dengan v6.1)
# Install ke direktori berbeda dulu untuk testing
NC_INSTALL_DIR=/usr/local/bin/nc7 curl -sSL \
  https://raw.githubusercontent.com/1607-NetEnginee/NightCrawler/main/scripts/install.sh | bash
```

### 2. Konversi konfigurasi v6.1

```bash
nightcrawler config import-v6 /etc/nightcrawler/nightcrawler.conf \
  --output ~/.config/nightcrawler/config.yaml
```

### 3. Validasi konfigurasi baru

```bash
nightcrawler config validate
```

### 4. Dry-run untuk konfirmasi cakupan

```bash
nightcrawler scan --targets-file scope.txt --dry-run
```

### 5. Scan sungguhan

```bash
nightcrawler scan --targets-file scope.txt
```

---

## Mapping Flag v6.x ke v7.0

| v6.x | v7.0 |
|---|---|
| `-t TARGET` | `--target TARGET` atau `-t TARGET` |
| `--mode stealth` | `--profile stealth` |
| `--out DIR` | `--output DIR` atau `-o DIR` |
| `--plugins "dns,tls"` | `--plugins dns,tls` |

---

*Dokumentasi ini bagian dari [NightCrawler v7.0](https://github.com/1607-NetEnginee/NightCrawler) oleh 1607-NetEnginee / Cyberoutcast.*
