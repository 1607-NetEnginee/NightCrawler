// Package output is the multi-writer for finding events.
//
// NDJSON is the canonical format — one finding per line, schema URI
// pinned at "nightcrawler.io/v1/finding". Every other format
// (HTML, TXT, SARIF, PDF) is generated *from* the NDJSON stream, not
// from in-memory findings. This is the philosophy from §18 of the
// design document and is what makes `nightcrawler report render
// --input old.ndjson` work cleanly.
//
// Writers implement a single-method interface:
//
//	type Writer interface {
//	    Write(api.Finding) error
//	    Close() error
//	}
//
// A multi-writer wraps an arbitrary number of sinks; failures in any
// one sink are logged but never block the others.
package output

// TODO(v7.0):
//   - Writer interface
//   - NDJSONWriter (canonical)
//   - TXTWriter (v6.1-compatibility mode)
//   - HTMLWriter (template-backed, embed.FS)
//   - SARIFWriter (SARIF 2.1.0 for GitHub code-scanning)
//   - PDFWriter (chromedp-backed, P1)
//   - MultiWriter
//   - notify/: Telegram, Slack, Discord, syslog sinks
