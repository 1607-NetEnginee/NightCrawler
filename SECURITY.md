# Security Policy

## Reporting a Vulnerability

The NIGHTCRAWLER project takes security issues seriously. We appreciate responsible disclosure.

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please use one of the following channels:

- **GitHub Security Advisory:** open a private security advisory in this repository
- **Email:** `rootmask597@proton.me`

### What to include

Please include the following in your report (as much as you can):

- A description of the issue and its impact
- Step-by-step reproduction
- Affected version(s)
- Any proof-of-concept code or screenshots
- Suggested mitigation, if any

### Response timeline

| When | What happens |
|---|---|
| Within 48 hours | Acknowledgement of receipt |
| Within 7 days | Initial assessment and severity classification |
| Within 30 days | Patch developed and tested for confirmed issues |
| Within 90 days | Public disclosure (coordinated with reporter) |

We follow a 90-day disclosure timeline by default, with extensions possible for issues that require significant infrastructure changes.

### Scope

In scope:
- The NIGHTCRAWLER binary and its plugins
- The official Docker images at `ghcr.io/1607-netengineee/nightcrawler:*`
- The installer script at `scripts/install.sh`
- The signature pack distribution mechanism
- The release-signing supply chain

Out of scope (please do not report):
- Vulnerabilities in target systems discovered by NIGHTCRAWLER (those are findings, not vulnerabilities in the tool)
- Issues that require root access or local code execution on the operator's machine to exploit
- Spam, phishing, or social engineering of project maintainers
- Self-XSS in HTML reports rendered from operator-controlled content

### Hall of Fame

Security researchers who responsibly disclose vulnerabilities will be credited (with their permission) in the release notes of the fix, and listed below.

_(No entries yet — be the first.)_



---

## Supported versions

| Version  | Supported          |
|----------|--------------------|
| v7.x     | :white_check_mark: |
| v6.x     | :warning: critical fixes only, until 2026-12-31 |
| < v6.0   | :x: end of life    |
