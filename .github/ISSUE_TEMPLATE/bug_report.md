---
name: Bug report
about: Something is broken or behaves unexpectedly
labels: ["bug", "triage"]
---

## What happened?

<!-- A clear and concise description of the bug. -->

## What did you expect?

<!-- Describe the behavior you expected. -->

## Reproduction

Steps to reproduce the behavior:

```bash
nightcrawler scan -t <target> --profile <profile> ...
```

Include:

- The full command you ran
- The target type (public domain, private lab, fixture)
- Any relevant configuration

## Environment

- `nightcrawler version`:
- OS + arch (e.g. Ubuntu 22.04 / amd64):
- Go version (if you built from source):
- Container? (Docker image tag, K8s, etc.):

## Logs / output

<details>
<summary>Click to expand</summary>

```
paste relevant log lines or stderr here
```

</details>

## Additional context

<!-- Anything else that would help triage. -->
