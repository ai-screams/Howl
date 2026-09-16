#!/usr/bin/env bash
#
# sync-binary.sh — the plugin's SessionStart hook.
#
# Three jobs, all of which must stay cheap and must never block session start:
#
#   1. Nudge, once, when the plugin is installed but the statusline never was.
#      Claude Code cannot configure `statusLine` from a plugin — only `agent`
#      and `subagentStatusLine` are allowed there — so a user who installs the
#      plugin and stops has a working set of skills and an empty status bar,
#      with nothing telling them why.
#   2. Keep the binary in step with the plugin. Claude Code updates the plugin
#      *content* from the marketplace; the binary is a separately-downloaded
#      release artifact and would otherwise drift.
#   3. Notice a newer release. Third-party marketplaces do not auto-update by
#      default, so a user who never turned it on would otherwise never learn
#      that a new version exists. Checked here, at most once a day, in the
#      background — never from the status line itself, which runs every few
#      seconds and is expected to finish in about ten milliseconds.
set -u

root="${CLAUDE_PLUGIN_ROOT:-}"
[ -n "$root" ] || exit 0

manifest="$root/.claude-plugin/plugin.json"
[ -f "$manifest" ] || exit 0

hud="$HOME/.claude/hud"
binary="$hud/howl"
notice="$hud/.update-available"

# Job 1: not set up yet. Say so and stop — never write to the user's settings
# uninvited. This is the only path that prints anything; a configured install
# is silent, so the message costs nothing once it has been acted on.
if [ ! -x "$binary" ]; then
  echo "Howl: the plugin is installed but the statusline binary is not."
  echo "Howl: run /howl:setup to download it and configure the status line."
  exit 0
fi

# CLAUDE_PLUGIN_DATA is the persistent per-plugin state dir (survives updates).
# Fall back to a stable location on older Claude Code that doesn't set it.
data="${CLAUDE_PLUGIN_DATA:-$hud/.plugin-state}"
mkdir -p "$data" 2>/dev/null || exit 0
saved="$data/plugin.json"

# Job 2: the plugin moved, so the binary should follow. Record the manifest only
# on success, so a failed or offline run retries on the next session start.
if ! { [ -f "$saved" ] && cmp -s "$manifest" "$saved"; }; then
  nohup env HOWL_ROOT="$root" HOWL_MANIFEST="$manifest" HOWL_SAVED="$saved" HOWL_NOTICE="$notice" bash -c '
    if bash "$HOWL_ROOT/scripts/install.sh" >/dev/null 2>&1; then
      cp "$HOWL_MANIFEST" "$HOWL_SAVED" 2>/dev/null || true
      # The install just took the latest release, so any pending notice is spent.
      rm -f "$HOWL_NOTICE" 2>/dev/null || true
    fi
  ' >/dev/null 2>&1 &
  exit 0
fi

# Job 3: in step with the plugin, so ask GitHub — at most once a day — whether
# the plugin itself is behind. Skipped when the user has opted out of the badge,
# so hiding it also stops the network call.
config="$hud/config.json"
[ -n "${HOWL_NO_UPDATE_CHECK:-}" ] && exit 0
grep -q '"hide_update_notice"[[:space:]]*:[[:space:]]*true' "$config" 2>/dev/null && exit 0

stamp="$data/last-update-check"
if [ -f "$stamp" ] && [ -z "$(find "$stamp" -mmin +1440 2>/dev/null)" ]; then
  exit 0 # checked within the last day
fi

nohup env HOWL_BINARY="$binary" HOWL_NOTICE="$notice" HOWL_STAMP="$stamp" bash -c '
  # Stamp first: a failed check should still wait a day before retrying, or an
  # offline machine would call out on every single session start.
  : > "$HOWL_STAMP"

  latest=$(curl -fsSL --max-time 10 \
    "https://api.github.com/repos/ai-screams/howl/releases/latest" 2>/dev/null \
    | sed -n "s/.*\"tag_name\"[[:space:]]*:[[:space:]]*\"v\{0,1\}\([^\"]*\)\".*/\1/p" \
    | head -1)
  [ -n "$latest" ] || exit 0

  current=$("$HOWL_BINARY" --version 2>/dev/null | sed -n "s/^howl[[:space:]]*v\{0,1\}\([^[:space:]]*\).*/\1/p")
  [ -n "$current" ] || exit 0

  if [ "$latest" = "$current" ]; then
    rm -f "$HOWL_NOTICE" 2>/dev/null || true
  else
    printf "%s\n" "$latest" > "$HOWL_NOTICE" 2>/dev/null || true
  fi
' >/dev/null 2>&1 &

exit 0
