# Security Policy

## Supported Versions

We actively support the latest release only. Please upgrade to the most recent version before reporting security issues.

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0.0 | :x:                |

---

## Reporting a Vulnerability

**🚨 Please DO NOT open public issues for security vulnerabilities.**

Public disclosure puts all users at risk. Report security issues privately through one of these channels:

### 1. GitHub Security Advisory (Recommended)

→ [Report a vulnerability](https://github.com/ai-screams/Howl/security/advisories/new)

This is the preferred method as it allows for coordinated disclosure and automatic CVE assignment.

### 2. Email (Alternative)

If you're unable to use GitHub Security Advisories, contact the maintainer directly through the email listed in the GitHub profile.

---

## What to Include in Your Report

Help us understand and fix the issue quickly by including:

- **Description**: Clear explanation of the vulnerability
- **Affected versions**: Which versions are impacted
- **Reproduction steps**: Detailed steps to reproduce the issue
- **Impact assessment**: What an attacker could achieve
- **Suggested fix**: If you have a patch or mitigation idea (optional)

---

## Security Scope

Howl is a statusline HUD for Claude Code. Security considerations include:

### In Scope

- ✅ Binary integrity and checksum verification
- ✅ Install script (`scripts/install.sh`) code injection risks
- ✅ OAuth token handling and storage
- ✅ Settings file manipulation safety
- ✅ Input validation from Claude Code JSON
- ✅ Malicious metric calculation causing DoS

### Out of Scope

- ❌ Claude Code itself (report to Anthropic)
- ❌ Third-party dependencies (we use Go stdlib only)
- ❌ User's local system security
- ❌ Network-level attacks (MitM on GitHub downloads)

---

## Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial assessment**: Within 3-5 business days
- **Fix timeline**:
  - Critical (remote code execution, token theft): 48-72 hours
  - High (privilege escalation, data exposure): 7 days
  - Medium (DoS, information disclosure): 14 days
  - Low (edge cases, theoretical issues): 30 days

---

## Disclosure Policy

We follow **coordinated disclosure**:

1. You report the issue privately
2. We confirm and develop a fix
3. We release a patched version
4. We publish a security advisory
5. You receive credit (if desired)

We will not disclose your identity without permission.

---

## Security Best Practices for Users

- ✅ Download binaries only from [official GitHub Releases](https://github.com/ai-screams/Howl/releases)
- ✅ Verify SHA256 checksums before installation
- ✅ Review `scripts/install.sh` before running
- ✅ Keep Howl updated to the latest version
- ✅ Report suspicious behavior immediately

---

## Past Security Advisories

None yet. This project has not had any security vulnerabilities disclosed.

---

**Last updated:** 2026-02-09
