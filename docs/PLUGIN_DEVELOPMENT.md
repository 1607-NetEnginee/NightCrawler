# Pengembangan Plugin NIGHTCRAWLER v7.0

> Panduan membuat plugin eksternal untuk NIGHTCRAWLER v7.0.

---

## Struktur Plugin

Setiap plugin mengimplementasikan interface `Plugin` dari `pkg/api`:

```go
package myplugin

import (
    "context"
    "github.com/1607-NetEnginee/NightCrawler/pkg/api"
)

type MyPlugin struct{}

func (p *MyPlugin) Manifest() api.Manifest {
    return api.Manifest{
        Name:        "myplugin",
        Version:     "1.0.0",
        Author:      "NamaKamu",
        Category:    "recon",
        Description: "Deskripsi plugin kamu",
        Tags:        []string{"passive", "recon"},
    }
}

func (p *MyPlugin) Run(ctx context.Context, req api.ScanRequest) ([]api.Finding, error) {
    // Implementasi scan di sini
    findings := []api.Finding{}
    
    // Contoh temuan
    findings = append(findings, api.Finding{
        Plugin:   "myplugin",
        Severity: api.SeverityInfo,
        Title:    "Contoh Temuan",
        Detail:   "Deskripsi detail temuan",
        Target:   req.Target,
    })
    
    return findings, nil
}
```

---

## Severity Levels

```go
api.SeverityCritical  // CVSS 9.0-10.0
api.SeverityHigh      // CVSS 7.0-8.9
api.SeverityMedium    // CVSS 4.0-6.9
api.SeverityLow       // CVSS 0.1-3.9
api.SeverityInfo      // Informational
```

---

## Registrasi Plugin

Daftarkan plugin kamu di `cmd/nightcrawler/builtins.go`:

```go
import "github.com/1607-NetEnginee/NightCrawler/internal/plugins/myplugin"

// Di dalam fungsi registerBuiltins():
registry.Register(&myplugin.MyPlugin{})
```

---

## Menjalankan Plugin

```bash
# List plugin termasuk yang baru
nightcrawler plugin list

# Lihat info plugin
nightcrawler plugin info myplugin

# Jalankan hanya plugin kamu
nightcrawler scan -t target.com --plugins myplugin
```

---

*Dokumentasi ini bagian dari [NightCrawler v7.0](https://github.com/1607-NetEnginee/NightCrawler) oleh 1607-NetEnginee / Cyberoutcast.*
