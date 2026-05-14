// Package report builds rendered scan reports from the canonical
// NDJSON event stream. The package is split into:
//
//   - builder.go : assembles a Report value from NDJSON input
//   - render/    : format-specific renderers (html, pdf, txt, sarif)
//   - template/  : embed.FS bundle of HTML templates, CSS, and JS
//
// Report rendering is decoupled from scanning so an operator can
// re-render historical scans with a newer template, or generate a
// SARIF for code-scanning integration after the fact.
package report

// TODO(v7.0):
//   - type Report struct { /* scan metadata + grouped findings */ }
//   - func Build(ctx, ndjsonReader) (*Report, error)
//   - func RenderHTML(*Report, w io.Writer, locale string) error
//   - func RenderSARIF(*Report, w io.Writer) error
//   - func RenderTXT(*Report, w io.Writer, locale string) error
//   - func RenderPDF(*Report, w io.Writer) error  // chromedp, P1
