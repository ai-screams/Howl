# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**Howl** — a Go statusline HUD for Claude Code. Reads JSON from stdin (piped every ~300ms), computes 5 derived metrics (context%, cache efficiency, API wait ratio, cost/min, tokens/sec) with 12 feature toggles, and outputs ANSI-formatted lines. Zero external dependencies (stdlib only), CGO_ENABLED=0, ~10ms cold start.

Module: `github.com/ai-screams/howl`

## Commands

```bash
make build     # CGO_ENABLED=0 go build → build/howl
make install   # build + copy to ~/.claude/hud/howl
make unit-test # go test ./... -v -cover -coverprofile=coverage.out
make test      # smoke test: pipes sample JSON into built binary
make lint      # golangci-lint run
make fmt       # go fmt ./...
make fmt-docs  # prettier on *.md *.yaml *.yml
make check     # fmt + fmt-docs + lint + unit-test
make setup     # configure .githooks + install prettier
```

Run a single test:

```bash
go test ./internal -run TestComputeMetrics -v
go test ./cmd/howl -run TestVersionFlag -v
```

## Architecture

### Data Pipeline (main.go)

```
stdin JSON → json.Decode(StdinData) → LoadConfig → ComputeMetrics → GetGitInfo → GetUsage → ParseTranscript → GetAccountInfo → Render → stdout (ANSI lines, spaces → NBSP)
```

Every function after `ComputeMetrics` is **optional** — returns nil on failure, and Render gracefully omits that section. This is the core design principle: graceful degradation everywhere.

### internal/ Package — Single Flat Package

All business logic lives in `internal/` with no sub-packages. The dependency graph between files:

- **types.go** — `StdinData` struct matching Claude Code's JSON schema, `ModelTier` classification
- **metrics.go** — `Metrics` struct + `ComputeMetrics()`: context%, cache efficiency, API wait ratio, cost/min, tokens/sec
- **constants.go** — Default threshold values (danger 85%, warning 70%, moderate 50%, session cost $5/$1, cache 80/50%, API wait 60/35%, speed 60/30 tok/s, cost velocity $0.50/$0.10/min, quota 10/25/50/75%). All configurable via `config.go` Thresholds
- **config.go** — `Config` + `FeatureToggles` + `Thresholds` (17 configurable color/behavior values) + 4 presets (full/minimal/developer/cost-focused). `LoadConfig()` reads `~/.claude/hud/config.json` with 4KB size guard. Features merge via `mergeFeatures(base, override)` — override can only enable, not disable. Thresholds merge via `mergeThresholds(base, override)` — only positive values override, validated via 3-step clamping
- **render.go** — `Render()` dispatches to `renderNormalMode` (2-4 lines) or `renderDangerMode` (2 dense lines) at configurable context threshold (default 85%). Line 2 supports priority ordering (max 5 metrics)
- **git.go** — `GetGitInfo()`: branch + dirty via subprocess with 1s timeout
- **usage.go** — `GetUsage()`: OAuth quota from `api.anthropic.com/api/oauth/usage`, 60s session-scoped cache in `/tmp/howl-{sessionID}/`. Token from macOS Keychain via `/usr/bin/security`
- **transcript.go** — `ParseTranscript()`: tail-reads last 64KB/100 lines of JSONL, extracts top-5 tools + running agents
- **account.go** — `GetAccountInfo()`: reads `~/.claude.json` for email display

### Test Conventions

94.7% coverage. Every source file has a `_test.go` pair plus `integration_test.go` for full pipeline tests.

- **git_test.go** creates real git repos in `t.TempDir()`
- **usage_test.go** uses package-level var injection (`usageAPIURL`, `getOAuthTokenFunc`) for test substitution — no interfaces
- **cmd/howl/main_test.go** does E2E binary execution: builds the binary, pipes JSON via `exec.Command`
- **integration_test.go** tests the full JSON → Unmarshal → ComputeMetrics → Render pipeline

## Commit Conventions

**Conventional Commits required** — enforced by `.githooks/commit-msg`. Auto-release (svu) uses these prefixes to determine semver bumps:

- `feat:` → minor version bump (triggers release)
- `fix:` → patch version bump (triggers release)
- `docs:`, `chore:`, `test:`, `ci:`, `style:`, `refactor:` → no version bump
- `chore(deps):` → no bump (Dependabot prefix)

The auto-release pipeline: PR merge to main → svu calculates next version → syncs `.claude-plugin/plugin.json` version → creates git tag → dispatches Release workflow → GoReleaser builds 4 binaries. Direct pushes to main (docs, ci, chore) do **not** trigger version bumps.

## Pre-commit Hooks

Active via `git config core.hooksPath .githooks`. Runs: go format, prettier (md/yaml), go mod tidy, golangci-lint. ~3.8s. Tests run in CI only.

If prettier fails on SECURITY.md or CHANGELOG.md tables, run `make fmt-docs` to auto-fix.

## Key Patterns

- **Nil-return = skip**: All optional data sources (git, usage, transcript, account) return nil on failure. Render checks nil before including.
- **Pointer fields for optional metrics**: `Metrics` uses `*int` / `*float64` — nil means "not enough data to compute."
- **Config merging**: `mergeFeatures(base, override)` is additive-only. Override `true` enables; `false` preserves base value. `mergeThresholds(base, override)` uses same pattern — only positive values override. No reflection — explicit per-field.
- **NBSP output**: All spaces in final output are replaced with `\u00A0` (non-breaking space) because Claude Code strips regular spaces from statusline.
- **Session isolation**: Usage cache keyed by `session_id` in `/tmp/howl-{sanitized_id}/`. Each Claude Code session gets its own cache.

## CI/CD

9 workflows across `.github/workflows/`. All GitHub Actions SHA-pinned. Dependabot updates actions weekly.

- **auto-release.yaml** and **release.yaml** have **separate concurrency groups** (`auto-release` vs `release`) — this is critical. Sharing a group causes the tag-triggered Release to be skipped.
- **release-build.yaml** uses GoReleaser v2 with `secrets: inherit` for the `GITHUB_TOKEN`.
- Auto-release uses a GitHub App token (not GITHUB_TOKEN) because Actions tokens can't trigger other workflows.
