# Security Policy

## Supported Versions

Only the latest release is supported. Please upgrade before reporting security issues.

| Version    | Supported          |
| ---------- | ------------------ |
| Latest     | :white_check_mark: |
| All others | :x:                |

---

## Reporting a Vulnerability

**Please do NOT report security vulnerabilities through public GitHub issues.**

### 1. GitHub Security Advisory (Recommended)

> [Report a vulnerability](https://github.com/ai-screams/Howl/security/advisories/new)

This allows coordinated disclosure with automatic CVE assignment.

### 2. Email (Alternative)

Send to **hanyul.ryu@hanyul.xyz** with description, steps to reproduce, and affected versions.

---

## What to Include in Your Report

- **Description**: Clear explanation of the vulnerability
- **Affected versions**: Which versions are impacted
- **Reproduction steps**: Detailed steps to reproduce the issue
- **Impact assessment**: What an attacker could achieve
- **Suggested fix**: If you have a patch or mitigation idea (optional)

---

## Security Architecture

### Trust Model

Howl receives JSON from stdin piped by Claude Code (trusted caller). All stdin fields are treated as untrusted for defense-in-depth.

### Subprocess Inventory

| Command                                                                   | Purpose          | Timeout | Mitigation                                            |
| ------------------------------------------------------------------------- | ---------------- | ------- | ----------------------------------------------------- |
| `git rev-parse --abbrev-ref HEAD`                                         | Branch detection | 1s      | `exec.CommandContext` with args separation (no shell) |
| `git status --porcelain --untracked-files=no`                             | Dirty status     | 1s      | `exec.CommandContext` with args separation (no shell) |
| `/usr/bin/security find-generic-password -s "Claude Code-credentials" -w` | OAuth token read | 3s      | Absolute path, macOS only                             |

### Credential Handling

- OAuth token fetched from **macOS Keychain** via `/usr/bin/security` CLI (read-only)
- Token held in **process memory only** — never written to disk, never logged
- Sent over **HTTPS** to `api.anthropic.com/api/oauth/usage` with `Authorization: Bearer` header
- Token lifetime bounded by process lifetime (no persistent caching of credentials)

### Network Egress

Single outbound connection: `https://api.anthropic.com/api/oauth/usage`

- **Sent**: Authorization header only
- **Not sent**: No telemetry, no analytics, no user data, no stdin content
- **Received**: Usage quota percentages (5h/7d remaining)

### File System Access

| Path                               | Operation  | Permissions         | Content                                                |
| ---------------------------------- | ---------- | ------------------- | ------------------------------------------------------ |
| `/tmp/howl-{sessionID}/usage.json` | Read/Write | 0700 dir, 0600 file | Usage percentages and timestamps only (no credentials) |
| `~/.claude/hud/config.json`        | Read       | —                   | User config (4KB size limit enforced)                  |
| `~/.claude.json`                   | Read       | —                   | Account info (email, display name)                     |
| Transcript JSONL                   | Read       | —                   | Last 64KB only via tail optimization                   |

### Supply Chain

- Binaries built via GoReleaser in GitHub Actions
- SHA256 checksums published alongside binaries
- All CI actions SHA-pinned with version comments
- Checksums are self-attesting (same pipeline) — GPG/cosign signing not yet implemented

---

## Security Scope

### In Scope

- Binary integrity and checksum verification
- Install script (`scripts/install.sh`) injection risks
- OAuth token read access via macOS Keychain (`security` CLI)
- Stdin JSON input validation and size limits
- Path traversal in cache directory (`/tmp/howl-*`) and transcript path
- ANSI escape sequence injection via user-controlled strings (model name, git branch, agent name, tool names)
- Config and account file parsing exploits (oversized files, malformed JSON)
- Git subprocess working directory controlled via stdin JSON (`project_dir`/`cwd`)
- CI/CD pipeline injection vectors (workflow commands, release integrity)
- Settings file (`~/.claude/settings.json`) manipulation safety

### Out of Scope

- Claude Code itself (report to [Anthropic](https://anthropic.com/security))
- Third-party dependencies (we use Go stdlib only — zero external deps)
- User's local system security beyond Howl's file access
- Man-in-the-middle attacks on HTTPS connections (mitigated by TLS)

---

## Response Timeline

This is a single-maintainer project. Timelines are best-effort targets.

- **Acknowledgment**: Within 48 hours
- **Initial assessment**: Within 5 business days
- **Fix timeline**:
  - Critical (remote code execution, token theft): 7 days (best effort: 48-72h)
  - High (privilege escalation, data exposure): 14 days
  - Medium (DoS, information disclosure): 30 days
  - Low (edge cases, theoretical issues): 60 days or next release

If you do not receive acknowledgment within 48 hours, please follow up.

---

## Disclosure Policy

We follow **coordinated disclosure** with a **90-day embargo**:

1. You report the issue privately
2. We acknowledge within 48 hours
3. We confirm and develop a fix
4. We release a patched version
5. We publish a security advisory (within 90 days of report)
6. You receive credit (if desired)

We will not disclose your identity without permission.

---

## Security Best Practices for Users

- Download binaries only from [official GitHub Releases](https://github.com/ai-screams/Howl/releases)
- Verify SHA256 checksums before installation
- Review `scripts/install.sh` before running
- Keep Howl updated to the latest version
- Report suspicious behavior immediately

---

## Past Security Advisories

None yet. This project has not had any security vulnerabilities disclosed.

---

**Last updated:** 2026-02-10
