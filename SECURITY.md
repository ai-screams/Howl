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

> [Report a vulnerability](https://github.com/ai-screams/howl/security/advisories/new)

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

| Command                                       | Purpose          | Timeout | Mitigation                                            |
| --------------------------------------------- | ---------------- | ------- | ----------------------------------------------------- |
| `git rev-parse --abbrev-ref HEAD`             | Branch detection | 1s      | `exec.CommandContext` with args separation (no shell) |
| `git status --porcelain --untracked-files=no` | Dirty status     | 1s      | `exec.CommandContext` with args separation (no shell) |

### Credential Handling

**The binary handles no credentials.** Quota comes from the `rate_limits` object
Claude Code already puts on stdin, so there is no token to read, hold or send.

Earlier versions read an OAuth token from the macOS Keychain and called
`api.anthropic.com` for the same numbers. That path is gone: `internal/usage.go`
is now a pure function over stdin, with no network, no cache and no Keychain.

### Network Egress

**The binary makes no network connections.** Nothing in `internal/` or
`cmd/` imports `net/http`; the status line renders entirely from stdin, local
config and the filesystem.

Two shell scripts do reach the network, and only these:

| Script                   | Host                           | When                                        | Sends                             |
| ------------------------ | ------------------------------ | ------------------------------------------- | --------------------------------- |
| `scripts/install.sh`     | `api.github.com`, `github.com` | On `/howl:setup` or a plugin version change | Nothing beyond the request itself |
| `scripts/sync-binary.sh` | `api.github.com`               | At most once a day, in the background       | Nothing beyond the request itself |

Both are unauthenticated GETs against public release metadata. No telemetry, no
analytics, no user data, no stdin content. `install.sh` verifies the SHA256 of
what it downloads against the published `checksums.txt` and refuses to install
on a mismatch. Setting `HOWL_NO_UPDATE_CHECK`, or `hide_update_notice` in
config, stops the daily check.

### File System Access

| Path                              | Operation | Permissions | Content                                                                                                                     |
| --------------------------------- | --------- | ----------- | --------------------------------------------------------------------------------------------------------------------------- |
| `~/.claude/hud/.update-available` | Read      | —           | A version string, written by the session-start hook (parsed as untrusted: first line, 64-byte cap, must start with a digit) |
| `~/.claude/hud/config.json`       | Read      | —           | User config (4KB size limit enforced)                                                                                       |
| `~/.claude.json`                  | Read      | —           | Account info (email, display name)                                                                                          |
| Transcript JSONL                  | Read      | —           | Last 64KB only via tail optimization                                                                                        |

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
- Stdin JSON input validation and size limits
- Transcript path handling (non-regular paths such as a FIFO are refused)
- ANSI escape sequence injection via strings the user does not control (session name, repository name from the git remote, transcript tool and agent names, subagent task text, directory names) — all pass through `sanitizeText` before rendering
- The update check: the release version it writes, and the file the status line reads it from
- Config and account file parsing exploits (oversized files, malformed JSON)
- Git subprocess working directory controlled via stdin JSON (`project_dir`/`cwd`)
- CI/CD pipeline injection vectors (workflow commands, release integrity)
- Settings file (`~/.claude/settings.json`) manipulation safety

### Out of Scope

- Claude Code itself (report to [Anthropic](https://anthropic.com/security))
- Third-party dependencies (we use Go stdlib only — zero external deps)
- User's local system security beyond Howl's file access
- Man-in-the-middle attacks on HTTPS connections (mitigated by TLS, and by checksum verification for downloaded binaries)

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

- Download binaries only from [official GitHub Releases](https://github.com/ai-screams/howl/releases)
- Verify SHA256 checksums before installation
- Review `scripts/install.sh` before running
- Keep Howl updated to the latest version
- Report suspicious behavior immediately

---

## Past Security Advisories

None yet. This project has not had any security vulnerabilities disclosed.

---

**Last updated:** 2026-02-10
