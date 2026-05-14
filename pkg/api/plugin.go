// Package api defines the public Plugin API and shared types for the
// NIGHTCRAWLER v7.0 framework. Third-party plugins import this package
// to implement the Plugin interface. This is the stability surface of
// the project: breaking changes here require a MAJOR version bump.
//
// Author: 1607-NetEnginee
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Severity ranks a finding's importance. The ordering matters: do not
// reorder constants without bumping MAJOR — they are serialized as
// strings in NDJSON output.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Score returns the numeric weight used by the aggregator to compute
// the overall risk score for a target (0..100).
func (s Severity) Score() float64 {
	switch s {
	case SeverityCritical:
		return 25
	case SeverityHigh:
		return 10
	case SeverityMedium:
		return 4
	case SeverityLow:
		return 1
	default:
		return 0
	}
}

// Category groups plugins for the UI and for profile selection.
type Category string

const (
	CategoryRecon       Category = "recon"
	CategoryFingerprint Category = "fingerprint"
	CategoryVuln        Category = "vuln"
	CategoryConfig      Category = "config"
	CategoryContent     Category = "content"
)

// Profile selects how aggressively a plugin should probe.
type Profile string

const (
	ProfileStealth    Profile = "stealth"
	ProfileDefault    Profile = "default"
	ProfileAggressive Profile = "aggressive"
)

// Manifest is plugin self-description. Used by the registry, the CLI
// `plugin info` command, and the scheduler (DependsOn drives the DAG).
type Manifest struct {
	Name         string
	Version      string
	Author       string
	Description  string
	Category     Category
	Profile      Profile
	Tags         []string
	DependsOn    []string
	OutputFields []string
	CWE          []string
	References   []Reference
}

// Reference is an external pointer (CWE, CVE, OWASP, vendor advisory).
type Reference struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
}

// Target describes a single scan subject after resolution.
type Target struct {
	Domain string
	URL    string
	IPs    []string
	Port   int
	Scheme string

	// Cache exposes per-target data shared between plugins, e.g.
	// catchall fingerprint, tech profile, DNS records. Plugins read
	// upstream-plugin output here without re-running probes.
	Cache TargetCache
}

// TargetCache is implemented by core; plugins only consume it.
type TargetCache interface {
	Get(key string) (any, bool)
	Set(key string, value any)
}

// Evidence is a fragment of data that supports a Finding.
type Evidence struct {
	Type string `json:"type"`           // response_excerpt, header, screenshot
	Data string `json:"data,omitempty"` // redacted if SecretLike == true
}

// Mitigation carries bilingual remediation guidance. Stored in the
// signature YAML and looked up by ID.
type Mitigation struct {
	ID      string   `json:"id"`
	TitleID string   `json:"title_id"`
	TitleEN string   `json:"title_en"`
	StepsID []string `json:"steps_id"`
	StepsEN []string `json:"steps_en"`
}

// Validation traces the false-positive filters that approved a finding.
// Ported from v6.1 detect_catchall(), validate_path_content(),
// is_path_relevant(), is_suspicious_ip_diff(), count_gambling_density().
type Validation struct {
	CatchallFiltered  bool   `json:"catchall_filtered"`
	ContentValidated  bool   `json:"content_validated"`
	TechFilterApplied string `json:"tech_filter_applied,omitempty"`
	Notes             string `json:"notes,omitempty"`
}

// Finding is the canonical unit emitted by plugins. Serialized as a
// single NDJSON line in the output stream.
type Finding struct {
	Schema        string      `json:"schema"`
	ID            string      `json:"id"`
	ScanID        string      `json:"scan_id"`
	Timestamp     time.Time   `json:"timestamp"`
	Plugin        string      `json:"plugin"`
	PluginVersion string      `json:"plugin_version"`
	Level         Severity    `json:"level"`
	Title         string      `json:"title"`
	Description   string      `json:"description,omitempty"`
	Resource      Resource    `json:"resource"`
	Target        TargetRef   `json:"target"`
	Evidence      []Evidence  `json:"evidence,omitempty"`
	Validation    Validation  `json:"validation,omitempty"`
	Mitigation    *Mitigation `json:"mitigation,omitempty"`
	References    []Reference `json:"references,omitempty"`
	Tags          []string    `json:"tags,omitempty"`
	RiskScore     float64     `json:"risk_score,omitempty"`
}

// Resource is the URL/asset where the finding was observed.
type Resource struct {
	URL        string `json:"url,omitempty"`
	Method     string `json:"method,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       int    `json:"port,omitempty"`
}

// TargetRef is a lightweight identifier for the target on a Finding.
type TargetRef struct {
	Domain string `json:"domain"`
	IP     string `json:"ip,omitempty"`
}

// Emitter is the callback plugins use to publish findings. The core
// wires this to the event bus channel; back-pressure is handled there.
type Emitter func(Finding)

// Deps is the dependency bundle injected into Plugin.Init. Keep this
// minimal — extending it is a breaking change.
type Deps struct {
	HTTP      *http.Client
	DNS       DNSResolver
	Logger    *slog.Logger
	Validator Validator
	Config    PluginConfig
	ScanID    string
}

// PluginConfig is a typed accessor over the plugin's slice of the
// global YAML config.
type PluginConfig interface {
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetDuration(key string) time.Duration
	GetStringSlice(key string) []string
}

// Validator is the cross-plugin false-positive engine (catch-all,
// tech-profile, path relevance, IP differential, density check).
type Validator interface {
	IsCatchall(target Target, body []byte) bool
	TechProfile(target Target) string
	IsPathRelevant(path, tech string) bool
	IsSuspiciousIPDiff(subdomain, ip string) bool
	GamblingDensity(body []byte) float64
}

// DNSResolver abstracts the DNS engine for plugins.
type DNSResolver interface {
	LookupA(ctx context.Context, host string) ([]string, error)
	LookupCNAME(ctx context.Context, host string) (string, error)
	LookupTXT(ctx context.Context, host string) ([]string, error)
	LookupMX(ctx context.Context, host string) ([]string, error)
	LookupNS(ctx context.Context, host string) ([]string, error)
}

// Plugin is the contract every scanner implements. Stateless across
// targets is preferred but not required; if state is kept it must be
// goroutine-safe because Run may be called concurrently for different
// targets.
type Plugin interface {
	Manifest() Manifest
	Init(ctx context.Context, deps Deps) error
	Run(ctx context.Context, target Target, emit Emitter) error
}
