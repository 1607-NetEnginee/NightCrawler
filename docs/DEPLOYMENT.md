# Deployment NIGHTCRAWLER v7.0

> Panduan deployment untuk berbagai environment.

---

## Instalasi Lokal (Debian/Ubuntu)

```bash
curl -sSL https://raw.githubusercontent.com/1607-NetEnginee/NightCrawler/main/scripts/install.sh | bash
```

## Build dari Source

```bash
git clone https://github.com/1607-NetEnginee/NightCrawler.git
cd NightCrawler
make build
sudo install -m 0755 bin/nightcrawler /usr/local/bin/nightcrawler
```

## VPS / Server

```bash
# Download binary langsung
curl -sSL https://github.com/1607-NetEnginee/NightCrawler/releases/download/v7.0.0/nightcrawler_7.0.0_linux_amd64.tar.gz | tar -xz
sudo install -m 0755 nightcrawler /usr/local/bin/nightcrawler
```

## CI/CD (GitHub Actions)

```yaml
- name: Security Scan
  run: |
    curl -sSL https://raw.githubusercontent.com/1607-NetEnginee/NightCrawler/main/scripts/install.sh | bash
    nightcrawler scan -t ${{ env.TARGET }} --profile compliance --fail-on high -f sarif -o ./results

- name: Upload SARIF
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: ./results/nightcrawler.sarif
```

## Docker (Coming Soon)

Docker image akan tersedia di versi mendatang setelah buildx dikonfigurasi.

```bash
# Belum tersedia di v7.0.0
# docker pull ghcr.io/1607-netengineee/nightcrawler:latest
```

---

*Dokumentasi ini bagian dari [NightCrawler v7.0](https://github.com/1607-NetEnginee/NightCrawler) oleh 1607-NetEnginee / Cyberoutcast.*
