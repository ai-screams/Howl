# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**Howl** — a Go statusline HUD for Claude Code. Reads JSON from stdin (event-driven: Claude Code re-runs the command on new messages, `/compact`, mode changes, a `refreshInterval` timer, and rate-limit or prompt-cache expiry, debounced at 300ms), computes 4 derived metrics (context%, cache efficiency, API wait ratio, cost/min) with feature toggles, and outputs ANSI-formatted lines. Zero external dependencies (stdlib only), CGO_ENABLED=0, ~10ms cold start.

Module: `github.com/ai-screams/howl`

## Commands

```bash
make build         # CGO_ENABLED=0 go build → build/howl
make install       # build + copy to ~/.claude/hud/howl
make unit-test     # go test ./... -v -cover -coverprofile=coverage.out
make test          # smoke test: pipes sample JSON into built binary
make lint          # golangci-lint run
make fmt           # go fmt ./...
make fmt-docs      # prettier on *.md *.yaml *.yml
make check         # fmt + fmt-docs + lint + unit-test
make setup         # configure .githooks + install prettier
make release-dry   # goreleaser snapshot build into dist/ (no publish)
make release-check # validate .goreleaser config
```

Run a single test:

```bash
go test ./internal -run TestComputeMetrics -v
go test ./cmd/howl -run TestE2E_VersionFlag -v
```

## Architecture

Some checkouts carry an untracked `AGENTS.md` tree covering the same directories at more length. It is excluded in `.git/info/exclude`, so it is per-clone and reaches nobody else — never treat it as the source of truth, and never assume a reader has it. Its copy of the preset table listed a removed metric and omitted four live toggles for months, which is the argument against a second copy of anything: when docs disagree, the code wins.

### Data Pipeline (main.go)

```
stdin JSON → json.Decode(StdinData) → LoadConfig → ComputeMetrics → GetGitInfo → UsageFromRateLimits → ParseTranscript → GetAccountInfo → ReadUpdateNotice → Render → stdout (ANSI lines, spaces → NBSP)
```

Every function after `ComputeMetrics` is **optional** — returns nil on failure, and Render gracefully omits that section. This is the core design principle: graceful degradation everywhere.

### internal/ Package — Single Flat Package

All business logic lives in `internal/` with no sub-packages. The dependency graph between files:

- **types.go** — `StdinData` struct matching Claude Code's JSON schema, `ModelTier` classification
- **metrics.go** — `Metrics` struct + `ComputeMetrics()`: context%, cache efficiency, API wait ratio, cost/min
- **constants.go** — Default threshold values (danger 85%, warning 70%, moderate 50%, session cost $5/$1, cache 80/50%, API wait 60/35%, cost velocity $0.50/$0.10/min, quota 10/25/50/75%). All 15 are configurable via `config.go` Thresholds
- **config.go** — `Config` + `FeatureToggles` + `Thresholds` (15 configurable color/behavior values) + 4 presets (full/minimal/developer/cost-focused). `LoadConfig()` reads `~/.claude/hud/config.json` with 4KB size guard. Features merge via `mergeFeatures(base, override)` — override can only enable, not disable. Thresholds merge via `mergeThresholds(base, override)` — only positive values override, validated via 3-step clamping
- **render.go** — `Render()` dispatches to `renderNormalMode` (2-4 lines) or `renderDangerMode` (2 dense lines) at configurable context threshold (default 85%). Line 2 supports priority ordering (max 5 metrics)
- **git.go** — `GetGitInfo()`: branch + dirty via subprocess with 1s timeout
- **usage.go** — `UsageFromRateLimits()`: converts the stdin `rate_limits` object into the render model. Pure function — no network, cache, or Keychain
- **transcript.go** — `ParseTranscript()`: tail-reads last 64KB/100 lines of JSONL, extracts top-5 tools + running agents
- **account.go** — `GetAccountInfo()`: reads `~/.claude.json` for email display
- **update.go** — `ReadUpdateNotice()`: reads `~/.claude/hud/.update-available`, written by the plugin's session-start hook. Never fetches — the render path stays offline
- **subagent.go** — `RenderSubagentRows()`: serves the `subagentStatusLine` setting via `howl --subagent`. Separate input schema (camelCase field names, unlike the main snake_case one); writes one `{"id","content"}` JSON line per agent-panel row it overrides

### Test Conventions

~98% coverage. Every source file except `constants.go` has a `_test.go` pair plus `integration_test.go` for full pipeline tests and `schema_test.go`, which decodes a real captured payload so a wrong `json:"…"` tag cannot pass unnoticed.

- **git_test.go** creates real git repos in `t.TempDir()`
- **fuzz_test.go** fuzzes `visibleLen`, `fitParts`, `RenderSubagentRow`, and `ReadUpdateNotice` — the escape-sequence and truncation paths where byte/rune and index mistakes hide
- **cmd/howl/main_test.go** does E2E binary execution: builds the binary, pipes JSON via `exec.Command`
- **integration_test.go** tests the full JSON → Unmarshal → ComputeMetrics → Render pipeline
- **Re-introduce the bug to prove a new test catches it.** Two regression tests here passed with their bug still present: one chose multi-byte inputs that were also long in runes, so a byte-vs-rune slice worked by accident (the trigger needs byte length over the bound and rune length under it); the other stripped ANSI before asserting, which erased the injected escape it was looking for.

## Commit Conventions

**Conventional Commits required** — enforced by `.githooks/commit-msg`. Auto-release (svu) uses these prefixes to determine semver bumps:

- `feat:` → minor version bump (triggers release)
- `fix:` → patch version bump (triggers release)
- `docs:`, `chore:`, `test:`, `ci:`, `style:`, `refactor:` → no version bump
- `chore(deps):` → no bump (Dependabot prefix)

The auto-release pipeline: PR merge to main → svu calculates next version → syncs `.claude-plugin/plugin.json` version → pushes a git tag → release.yaml starts from its `push: tags` trigger → GoReleaser builds 4 binaries. Direct pushes to main (docs, ci, chore) do **not** trigger version bumps.

Three things this pipeline depends on, all easy to break:

- **Never write a CI skip marker into a commit message, not even inside backticks while explaining one.** GitHub scans the entire message and suppresses every workflow for that push. A fix to this pipeline once skipped its own verification run that way. `.githooks/commit-msg` now rejects such messages; write "skip-ci" or "the CI skip marker" in prose instead.
- **The sync commit must not carry a skip marker.** The tag points at it, and GitHub evaluates skip directives against the head commit of a push — marking it skip suppresses the tag push as well, so nothing starts release.yaml.
- **svu decides the bump from each commit's subject line, not its body.** The PR title becomes the squash subject, so it must carry the prefix the release needs: a `chore:`-titled PR that adds a feature in its body ships no release. A `BREAKING CHANGE:` footer in the body still forces a major bump; a `feat!:` line in the body does not. The workflow passes `--tag.mode current` so the current version comes only from tags reachable from `main`.

`workflow_dispatch` on release.yaml stays available for re-running a release by hand: `gh workflow run release.yaml -f tag=vX.Y.Z`.

## Pre-commit Hooks

Active via `git config core.hooksPath .githooks`. Runs: go format, prettier (md/yaml), go mod tidy, golangci-lint. ~2s. Tests run in CI only.

The hook exits before any check when golangci-lint or prettier is missing from both `PATH` and `$(go env GOPATH)/bin`, so no commit is possible at all. `make setup` installs only prettier; install golangci-lint separately (CI pins the version in `quality-lint.yaml`).

If prettier fails on SECURITY.md or CHANGELOG.md tables, run `make fmt-docs` to auto-fix.

## Key Patterns

- **Nil-return = skip**: All optional data sources (git, usage, transcript, account) return nil on failure. Render checks nil before including.
- **Pointer fields for optional metrics**: `Metrics` uses `*int` / `*float64` — nil means "not enough data to compute."
- **Config merging**: `mergeFeatures(base, override)` is additive-only. Override `true` enables; `false` preserves base value. `mergeThresholds(base, override)` uses same pattern — only positive values override. No reflection — explicit per-field.
- **NBSP output**: All spaces in final output are replaced with `\u00A0` (non-breaking space) because Claude Code strips regular spaces from statusline.
- **Sanitize before rendering**: every externally-sourced string — `session_name`, `workspace.repo`, transcript tool and agent names, subagent task text, a directory basename — goes through `sanitizeText` before it reaches output. Unsanitized, a crafted value reached OSC 52 and wrote the terminal clipboard. `sanitize_test.go` fails any renderer that skips it.

## Plugin Distribution

The repo is both the marketplace (`.claude-plugin/marketplace.json`) and the plugin it serves. Two platform constraints shape the whole install flow:

- **A plugin cannot set `statusLine`.** Only `agent` and `subagentStatusLine` are allowed in a plugin's own `settings.json`, which is why `/howl:setup` exists to write the user's settings. `subagentStatusLine` ships from the plugin and needs no setup.
- **Auto-update is off by default for third-party marketplaces.** Without the `/plugin` → Marketplaces toggle, an install stays on its original version forever. The status line's own update badge exists to cover users who never find it.

`scripts/sync-binary.sh` runs at SessionStart and must stay a no-op with no network when in sync. Preset contents live in `presets` in `internal/config.go` — doc copies of them drifted for months; read the source.

## Product Page

`site/` is the product page at https://ai-scream.ai/Howl/ — one static `index.html` with inline CSS/JS and self-hosted fonts, no build step. `pages.yaml` deploys it on pushes to `main` that touch `site/**`. The path is case-sensitive: spell it `Howl`.

The hero's HUD demo is a JavaScript copy of `renderNormalMode`/`renderDangerMode`. Its preset toggles and thresholds sit in the page's `howl-config` JSON block, and `internal/site_test.go` fails when that block differs from `config.go`, when any `http://`, `https://` or `//host` appears outside an outbound `<a>`, the canonical link or an `og:` meta (so no remote stylesheet, script, image, font, frame, media file or `@import` can slip in), when its script calls `fetch`, `XMLHttpRequest`, `WebSocket`, `EventSource`, `sendBeacon` or `import()`, or when a local link is broken. A change to which segments a line carries is not caught by the test — mirror it in `site/index.html` by hand.

## CI/CD

10 workflows across `.github/workflows/`. All GitHub Actions SHA-pinned. Dependabot updates actions weekly.

- **Dependabot does not see tools a workflow installs itself** — golangci-lint (`version:` in quality-lint.yaml), gitleaks (curl + sha256 in security-secrets.yaml), svu (wget + sha256 in auto-release.yaml), govulncheck (`go install …@vX` in security-scan.yaml). Bump them by hand; take the sha256 from the release's checksums file, not from a hash you computed alone.
- **A neutral or skipping `CodeQL`/`gitleaks` check on a PR is not a failure.** Code scanning reports "configurations not found" when main's results came from default setup or from the release/weekly workflow categories that the PR did not upload. It does not block merging.
- **auto-release runs only on PR merge**, so a change to it is first exercised by the next merge. Check that run's `Current: vX, Next: vY` log line.
- **auto-release.yaml** and **release.yaml** have **separate concurrency groups** (`auto-release` vs `release`) — this is critical. Sharing a group causes the tag-triggered Release to be skipped.
- **release-build.yaml** runs GoReleaser v2; `release.yaml` calls it with `secrets: inherit` to pass `GITHUB_TOKEN`.
- Auto-release uses a GitHub App token (not GITHUB_TOKEN) because Actions tokens can't trigger other workflows.
