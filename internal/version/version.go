// Package version exposes build-time metadata. Values are overridden
// at link time via -ldflags. See Makefile and .goreleaser.yaml.
package version

import "runtime/debug"

var (
	// Version is the semantic version of this build, e.g. "7.0.0".
	// Set via: -X github.com/1607-NetEnginee/NightCrawler/internal/version.Version=...
	Version = "dev"

	// Commit is the short Git SHA.
	Commit = "unknown"

	// BuildDate is the RFC3339 build timestamp.
	BuildDate = "unknown"
)

// Full returns a human-readable version string used by --version.
func Full() string {
	v := Version
	if Commit != "unknown" {
		v += "+" + Commit
	}
	return v
}

// GoVersion returns the Go toolchain used to build this binary.
func GoVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.GoVersion
	}
	return "unknown"
}
