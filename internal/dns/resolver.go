// Package dns owns the DNS resolver shared across plugins. Built on
// github.com/miekg/dns rather than net.Resolver so we can:
//
//   - Use multiple resolvers in parallel (system + 1.1.1.1 + 8.8.8.8).
//   - Distinguish NXDOMAIN from network failures cleanly.
//   - Cache results across plugins to amortize subdomain bruteforce.
//   - Honor the global concurrency cap from config.
//
// The crt.sh passive enumeration source lives at crtsh.go and is the
// migration of v6.1's `curl https://crt.sh/?q=...&output=json | jq`
// pipeline, refactored to native JSON parsing and a configurable HTTP
// timeout.
package dns

// TODO(v7.0):
//   - New(*config.DNSConfig) Resolver
//   - LookupA/CNAME/TXT/MX/NS with retry and per-resolver fallback
//   - Bruteforce(ctx, base, wordlist) (<-chan SubdomainHit, error)
//   - CrtSh(ctx, base) ([]SubdomainHit, error)
//   - Result cache with TTL
