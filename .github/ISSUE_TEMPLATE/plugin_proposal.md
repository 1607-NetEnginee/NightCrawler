---
name: Plugin proposal
about: Propose a new built-in or community plugin
labels: ["plugin-proposal", "triage"]
---

## Plugin name

`<short-name>` (e.g. `cloudflare-bypass`, `s3-enum`, `kubernetes-discovery`)

## One-line description

<!-- What does this plugin detect, recon, or fingerprint? -->

## Category

- [ ] Recon
- [ ] Fingerprint
- [ ] Vulnerability check
- [ ] Configuration / posture
- [ ] Content / compromise indicator
- [ ] Integration (third-party tool wrapper)

## Use case

<!-- Concrete scenario. "When auditing X, an operator currently has to
manually do Y; this plugin would automate Y to produce findings of
type Z." -->

## Manifest sketch

```yaml
apiVersion: nightcrawler.io/v1
kind: Plugin
metadata:
  name: <plugin-name>
spec:
  description: "..."
  category: ...
  profile: default        # stealth | default | aggressive
  tags: []
  dependsOn: []
  cwe: []
  references: []
```

## Output

<!-- What does a finding from this plugin look like? Title, severity
mapping, evidence, mitigation ID. -->

## False positive risk

<!-- HOW are you going to keep false positives low? What validators
will you use? This is the most important section — proposals without
a credible answer here typically don't get accepted. -->

## Noise profile

- [ ] Stealth-safe (no payloads, no aggressive probing)
- [ ] Default-safe (mild probing, would not trigger most WAFs)
- [ ] Noisy (active payloads, likely to be flagged; needs opt-in)

## Dependencies

- [ ] Pure Go, no external tools
- [ ] Wraps an external tool: `___`
- [ ] Calls a third-party API: `___`

## Are you willing to implement this?

- [ ] Yes, I plan to send a PR
- [ ] Yes, but I need pairing/guidance
- [ ] No, requesting that someone else implement
