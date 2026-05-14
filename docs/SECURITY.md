# Security Model NIGHTCRAWLER v7.0

> Dokumen ini menjelaskan model keamanan, threat model, dan panduan hardening untuk operator NIGHTCRAWLER.

---

## Prinsip Keamanan

- **Non-root by default** — binary tidak memerlukan hak root untuk berjalan
- **TLS verification aktif** — koneksi ke target selalu diverifikasi kecuali dimatikan manual
- **Rate limiter adaptif** — menghormati `Retry-After` dan backoff otomatis saat 429/503
- **No telemetry** — tidak ada data yang dikirim ke server eksternal tanpa persetujuan operator
- **Audit trail** — setiap scan dicatat dengan scan_id, operator, timestamp

---

## Hardening Rekomendasi

### Jalankan sebagai user non-root

```bash
# Buat user khusus untuk nightcrawler
sudo useradd -r -s /bin/false nightcrawler
sudo -u nightcrawler nightcrawler scan -t target.com
```

### Verifikasi binary setelah download

```bash
# Verifikasi SHA-256
grep "nightcrawler_7.0.0_linux_amd64.tar.gz" checksums.txt | sha256sum -c -
```

### Batasi akses jaringan

Gunakan firewall untuk membatasi koneksi keluar dari host nightcrawler hanya ke target yang diizinkan.

---

## Threat Model

| Ancaman | Mitigasi |
|---|---|
| Binary palsu | Verifikasi SHA-256 checksums.txt |
| MITM saat download | HTTPS + checksum verification |
| Scan target tanpa izin | Tanggung jawab operator (lihat etika penggunaan) |
| Kebocoran data laporan | Enkripsi output (fitur P1+), akses direktori laporan dibatasi |
| Rate flooding target | Rate limiter adaptif, profile stealth |

---

## Laporkan Kerentanan

Laporkan kerentanan keamanan ke: `rootmask597@proton.me`

**Jangan** buka public GitHub issue untuk laporan keamanan.

---

*Dokumentasi ini bagian dari [NightCrawler v7.0](https://github.com/1607-NetEnginee/NightCrawler) oleh 1607-NetEnginee / Cyberoutcast.*
