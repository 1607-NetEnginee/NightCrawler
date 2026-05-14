// Package http owns the stealth-aware HTTP client used by every
// scanner plugin. A single *http.Client is shared across the process,
// configured with:
//
//   - Connection pooling tuned for many-target concurrent workloads.
//   - A custom RoundTripper chain (transport.go) that injects:
//   - rotating User-Agent (from the configured pool)
//   - rotating Referer (from the configured pool)
//   - Chrome-style Sec-Fetch-* headers
//   - jittered delay between requests to a given host
//   - adaptive backoff on 429/503 and Retry-After
//   - Body size limit via io.LimitReader (default 5 MiB).
//   - TLS verification on by default; --insecure requires explicit
//     opt-in elsewhere.
//
// This package replaces the v6.1 stealth_curl() bash function and the
// thousands of curl subprocess invocations it produced.
package http

// TODO(v7.0):
//   - New(*config.HTTPConfig) (*http.Client, error)
//   - transport.go: chained RoundTripper
//   - stealth.go: UA + Referer pools, jitter
//   - ratelimit.go: adaptive token bucket with 429/503 awareness
//   - pool.go: connection pool tuning (MaxIdleConnsPerHost, etc)
