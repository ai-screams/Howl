---
description: This skill should be used when the user asks to "install howl", "set up the statusline", "configure howl", "enable the HUD", or "install the statusline binary". It downloads the Howl binary and configures Claude Code to use it.
disable-model-invocation: true
---

# Howl Setup

Install and configure the Howl statusline HUD for Claude Code.

## What This Does

1. Downloads the latest Howl binary from GitHub Releases, verifying its SHA256
2. Installs it to `~/.claude/hud/howl`
3. Points `statusLine` in `~/.claude/settings.json` at it
4. Tells the user to restart Claude Code

Step 3 is why this skill exists. A plugin cannot set `statusLine` itself —
Claude Code allows only the `agent` and `subagentStatusLine` keys in a plugin's
own settings — so the main status line has to be written into the user's
settings by something running as the user. That is this script.

The agent panel needs no such step: the plugin ships `subagentStatusLine`
directly, so per-subagent rows start working as soon as the binary exists.

## Installation

Run the install script:

```bash
bash "${CLAUDE_PLUGIN_ROOT}/scripts/install.sh"
```

It refuses to overwrite a `statusLine` that points at something other than
Howl. If the user already runs a different status line, report that and let
them decide rather than working around it.

## Verify Installation

```bash
~/.claude/hud/howl --version
echo '{}' | ~/.claude/hud/howl
```

The first prints the installed version. The second should print a status line
built from empty input — that exercises the whole render path, so if it prints
anything at all the binary works.

If the binary is missing or not executable, check `~/.claude/hud/howl` and run
`chmod +x ~/.claude/hud/howl`.

## Post-Install

Tell the user:

- The binary is at `~/.claude/hud/howl`, and `statusLine` now points at it
- **Restart Claude Code** — the status line does not appear until then
- Display is configurable: `/howl:configure` for a preset, `/howl:customize`
  for individual metrics, `/howl:threshold` for the colour thresholds

## Staying Up To Date

Three separate things move, and they update differently. Explain whichever the
user asks about rather than reciting all three.

| What                             | How it updates                                                         |
| -------------------------------- | ---------------------------------------------------------------------- |
| Plugin content (skills, scripts) | Claude Code, from the marketplace — **only if auto-update is on**      |
| The binary                       | This plugin's `SessionStart` hook, whenever the plugin version changes |
| Knowing an update exists         | A badge on the status line, from a once-a-day check                    |

**Auto-update is off by default for this marketplace.** Claude Code enables it
for its own marketplaces, not for third-party ones. Without it the plugin stays
on whatever version was current at install time, forever. Turn it on once:

```
/plugin  →  Marketplaces  →  ai-screams-howl  →  Enable auto-update
```

Once it is on, Claude Code refreshes after session start (with a delay of up to
ten minutes), and the hook then pulls the matching binary in the background.

To update right now instead of waiting:

```bash
claude plugin marketplace update ai-screams-howl
claude plugin update howl@ai-screams-howl
```

Or re-run this skill, which downloads the latest release directly.

### The update badge

Claude Code has no "update available" indicator — its plugin list does not
carry the notion at all — so Howl shows its own. The `SessionStart` hook asks
GitHub for the latest release at most once a day, in the background, and writes
the answer to `~/.claude/hud/.update-available`. The status line reads that file
and shows `↑1.11.0` on the first line.

The check never runs from the status line itself: that command runs every few
seconds, and a network call there would wreck it.

To turn the badge off, set `hide_update_notice` in `~/.claude/hud/config.json`.
That also stops the daily check.

```json
{ "features": { "hide_update_notice": true } }
```

## Troubleshooting

If the install fails:

- Check internet connectivity (needs access to github.com)
- Verify `curl` is available
- Try manual download from: https://github.com/ai-screams/howl/releases/latest
- Supported platforms: macOS (arm64/amd64), Linux (arm64/amd64)

If the statusline does not appear after restart:

- Check `~/.claude/settings.json` contains the `statusLine` field
- Verify the binary path is correct for this system
- Run `~/.claude/hud/howl --version` to confirm the binary works
