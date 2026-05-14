// Package findings is the aggregator. It owns the consumer side of
// the event bus: a single goroutine reads api.Finding values off the
// findings channel, deduplicates them, calculates the running risk
// score, and forwards them to the output layer.
//
// Deduplication: a SHA-256 hash of (plugin, level, resource.url,
// title) is computed; repeats within the same scan are suppressed,
// which protects against pathological scenarios like a misbehaving
// plugin emitting the same finding in a loop.
//
// Risk scoring: severity-weighted sum capped at 100. Weights:
//
//	critical=25, high=10, medium=4, low=1, info=0.
//
// A target with one critical and three highs scores 55.
package findings

// TODO(v7.0):
//   - Aggregator struct
//   - Consume(ctx, <-chan api.Finding) → drives writers
//   - Dedupe (hash + Bloom filter for memory bound)
//   - Classifier: CWE/CVE/OWASP enrichment from references YAML
//   - Risk scorer with target-level rollup
