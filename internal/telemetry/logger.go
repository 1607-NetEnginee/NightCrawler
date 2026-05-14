// Package telemetry wires structured logging, optional OpenTelemetry
// tracing, and optional Prometheus metrics.
//
// Logging is built on stdlib log/slog. Two handlers:
//
//   - "tint" : color/pretty for interactive TTY (uses lmittmann/tint)
//   - "json" : structured for production / CI / log shippers
//
// Tracing and metrics are off by default and opt-in via config —
// shipping with them always-on would surprise operators with new
// network egress.
//
// Secret redaction is implemented as a slog.Handler middleware that
// strips well-known patterns from log attributes before they hit any
// sink: "password=", "token=", "api_key=", Bearer tokens, base64
// blobs longer than 32 chars in "evidence" attributes.
package telemetry

// TODO(v7.0):
//   - NewLogger(*config.TelemetryConfig) *slog.Logger
//   - redactHandler wrapping any slog.Handler
//   - InitTracing(ctx, *config.TracingConfig) (shutdown func, error)
//   - InitMetrics(*config.MetricsConfig) (http.Handler, error)
