// Package config owns the Viper-backed configuration plumbing for
// NIGHTCRAWLER v7.0.
//
// Layered resolution order (lowest to highest precedence):
//
//  1. Compiled-in defaults from defaults.go.
//  2. /etc/nightcrawler/config.yaml (system-wide).
//  3. $XDG_CONFIG_HOME/nightcrawler/config.yaml (per-user).
//  4. NIGHTCRAWLER_* environment variables.
//  5. CLI flags.
//
// Validation is delegated to validate.go which JSON-Schema-checks the
// merged config before the orchestrator sees it.
//
// The v6.x bash config translator lives at migrate.go; it parses the
// legacy /etc/nightcrawler/nightcrawler.conf shell file and emits the
// equivalent v7 YAML so existing operators have a one-command upgrade
// path. See `nightcrawler config import-v6`.
//
// This package is intentionally thin: Viper does the heavy lifting and
// we expose typed accessors to the rest of the codebase so the Viper
// dependency does not leak past this boundary.
package config

// TODO(v7.0):
//   - Load: read+merge defaults/system/user/env/flags, return *Config.
//   - Save: write canonical YAML back to disk (round-trip safe).
//   - Watch: optional fsnotify-based reload (P1+).
//   - ImportV6: parse v6.x bash config, return v7 Config.
