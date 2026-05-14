// Package validator is the false-positive engine. It contains the
// five validators that v6.1 evolved over five releases of bug-fix
// iteration, ported from Bash to Go and exposed via the
// api.Validator interface so plugins can use them uniformly.
//
// The five validators (mapping to v6.1 source lines):
//
//   - Catchall / soft-404         (was: detect_catchall, line 406)
//     Request a random slug; record body hash; treat any later 200
//     response with matching hash as a soft-404.
//
//   - Content validation          (was: validate_path_content, line 447)
//     Per-signature content checks (e.g. .env must look like KEY=VAL,
//     .git/HEAD must start with "ref:", actuator/env must have
//     "activeProfiles"). This is the most valuable migration target —
//     it is the asset that differentiates NIGHTCRAWLER from generic
//     scanners.
//
//   - Tech profile detection      (was: detect_tech_profile, line 519)
//     Quick HEAD-and-fetch fingerprint of the target's framework
//     (Laravel / WordPress / Spring / generic-PHP / unknown). Used by
//     downstream plugins to filter signatures.
//
//   - Path relevance              (was: is_path_relevant, line 548)
//     Tech-aware filter that skips e.g. /wp-config.php on a Laravel
//     site. Reduces noise dramatically.
//
//   - Smart IP differential       (was: is_suspicious_ip_diff, line 579)
//     Don't flag mail.target.com → google.com as anomalous; mail
//     subdomains pointing to known relay ranges (Google/Cloudflare/
//     AWS Mail) are expected.
//
//   - Gambling density            (was: count_gambling_density, line 615)
//     Strip nav/footer/header/script/style before computing keyword
//     density. Prevents false-positive from breadcrumb navigation.
//
// These functions are pure (no I/O beyond what the plugin already
// performed) and safe for concurrent use.
package validator

// TODO(v7.0):
//   - Engine struct with config, sigDB, cache
//   - IsCatchall(target, body) bool                      (catchall.go)
//   - ValidateContent(sig, body) bool                    (content.go)
//   - TechProfile(target) string                         (techprofile.go)
//   - IsPathRelevant(path, tech) bool                    (pathrelevance.go)
//   - IsSuspiciousIPDiff(sub, ip) bool                   (ipdiff.go)
//   - GamblingDensity(body) float64                      (density.go)
