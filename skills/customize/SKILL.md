---
description: Customize Howl statusline with fine-grained metric toggles on top of a preset
disable-model-invocation: false
---

# Howl Customize

Advanced configuration for Howl statusline: choose a base preset, then toggle individual metrics on top of it. Segment order within a line is fixed in the binary; there is no ordering setting.

## Configuration Structure

```json
{
  "preset": "developer",
  "features": {
    "quota": true
  },
  "thresholds": {
    "context_danger": 90
  }
}
```

- **preset**: Base configuration (`full`, `minimal`, `developer`, `cost-focused`)
- **features**: Override specific metrics from the preset base (optional)
- **thresholds**: Override color/behavior breakpoints (optional, see `/howl:threshold`)

## Process

### Step 1: Choose Base Preset

**Use AskUserQuestion to present preset choices:**

- **Question**: "Which base preset would you like to start with?"
- **Header**: "Choose Base Preset"
- **Options** (4):
  - Label: **"full (default)"**  
    Description: "All 12 preset toggles - Complete visibility (up to 4 lines)"
  - Label: **"minimal"**  
    Description: "Model + Context + Cost + Duration, plus the version line (2 lines)"
  - Label: **"developer"**  
    Description: "Coding focus: full minus API wait, cost velocity and agent name (up to 4 lines)"
  - Label: **"cost-focused"**  
    Description: "Budget tracking: Account, Output tokens, Quota, API wait, Cost velocity (up to 3 lines)"

**Store the user's selection as `chosenPreset`.**

### Step 2: Toggle Individual Metrics

There are 23 display toggles. `AskUserQuestion` allows at most four questions
per call and four options per question, so ask in themed groups rather than one
long list — a single 23-item checkbox is not something the tool can render.

Use `multiSelect: true` on every group, and pre-check what the chosen preset
already enables.

**First call — the four groups people change most:**

| Group            | Options                                                    |
| ---------------- | ---------------------------------------------------------- |
| Workspace        | `git`, `repo`, `worktree`, `added_dirs`                    |
| Session          | `account`, `session_name`, `vim_mode`, `output_style`      |
| Cost and quota   | `quota`, `cost_velocity`, `api_wait_ratio`, `line_changes` |
| Tools and agents | `tools`, `agents`, `agent_name`, `pull_request`            |

**Second call — model and context:**

| Group       | Options                            |
| ----------- | ---------------------------------- |
| Cache       | `cache_efficiency`, `prompt_cache` |
| Model state | `effort`, `thinking`, `fast_mode`  |
| Context     | `output_tokens`, `exceeds_200k`    |

What each one shows:

| Toggle             | Shows                                                                                                          | Default    |
| ------------------ | -------------------------------------------------------------------------------------------------------------- | ---------- |
| `account`          | Account email                                                                                                  | on in full |
| `git`              | Git branch and dirty marker                                                                                    | on in full |
| `line_changes`     | Lines added and removed                                                                                        | on in full |
| `quota`            | 5h and 7d quota bars                                                                                           | on in full |
| `tools`            | Top tool call counts                                                                                           | on in full |
| `agents`           | Running agent names                                                                                            | on in full |
| `cache_efficiency` | Cache hit rate of the **last API call** (`Cache:99%`)                                                          | on in full |
| `api_wait_ratio`   | Share of session time spent waiting on the API                                                                 | on in full |
| `cost_velocity`    | Cost per minute                                                                                                | on in full |
| `vim_mode`         | Vim mode (`Insert`, `V-Line`, …)                                                                               | on in full |
| `agent_name`       | Active agent (`@executor`)                                                                                     | on in full |
| `output_tokens`    | Output tokens of the current response (`Out:1K`)                                                               | on in full |
| `effort`           | Reasoning effort (`E:xhigh`)                                                                                   | off        |
| `thinking`         | Extended thinking indicator (`Think`)                                                                          | off        |
| `session_name`     | Session name, truncated                                                                                        | off        |
| `pull_request`     | Linked PR or MR (`MR#23 approved`), clickable where supported                                                  | off        |
| `worktree`         | Active worktree (`wt:name`)                                                                                    | off        |
| `prompt_cache`     | **Session-wide** cache: hit ratio, time until the cache goes cold, rebuilds and their cause                    | off        |
| `fast_mode`        | Fast mode indicator                                                                                            | off        |
| `exceeds_200k`     | Warns past 200k tokens — a fixed threshold, so it can fire while the context bar still reads low on a 1M model | off        |
| `output_style`     | Active output style, unless it is the default                                                                  | off        |
| `repo`             | Repository from the origin remote (`owner/name`)                                                               | off        |
| `added_dirs`       | Count of directories added with `/add-dir`                                                                     | off        |

`cache_efficiency` and `prompt_cache` answer different questions — the last call
versus the whole session — so neither replaces the other. Offer both.

**Pre-check based on `chosenPreset`:**

These are the exact sets in `internal/config.go`; read them there rather than
trusting a summary.

- **full** (12): account, git, line_changes, output_tokens, quota, tools, agents, cache_efficiency, api_wait_ratio, cost_velocity, vim_mode, agent_name
- **minimal** (0): none
- **developer** (9): account, git, line_changes, output_tokens, quota, tools, agents, cache_efficiency, vim_mode
- **cost-focused** (5): account, output_tokens, quota, api_wait_ratio, cost_velocity

**Features are additive-only.** Checking enables; unchecking does **not** disable,
because `mergeFeatures` can only turn a flag on. To drop something the preset
includes, start from `minimal` in Step 1 and check only what is wanted.

- Want `full` without git? Start from `minimal` and check everything except git.
- Want `developer` plus the API wait ratio? Start from `developer` and check `api_wait_ratio`.

**The update badge is the exception.** `hide_update_notice` is an opt-**out**: it
is on by default, and setting it to `true` turns the badge off. It exists in this
inverted shape precisely because an additive merge could never switch off a
default-on flag. Do not put it in the metric groups — offer it only if the user
asks to stop seeing update notices, and explain that it also stops the daily
version check.

**Always displayed, not toggleable:** model badge, context bar, session cost, duration.

If the resulting selection matches the preset exactly, omit `features` from
config.json entirely — a preset name alone is easier to read later.

**Store selections as `selectedFeatures` array.**

### Step 3: Generate and Apply Configuration

**Build the config object:**

```json
{
  "preset": "<chosenPreset>",
  "features": {
    // Only include if different from preset base
    // Format: "metric_name": true
  }
}
```

**Apply configuration:**

```bash
mkdir -p ~/.claude/hud
cat > ~/.claude/hud/config.json << 'EOF'
{JSON_CONTENT_HERE}
EOF
```

**Show a configuration summary:**

```
✅ Configuration Applied

Preset: developer
Overrides: api_wait_ratio (enabled)

Preview (example):
[Opus 4.6] | user@example.com | main* | Out:1K | $185.4 | 91h12m
████░░░░░░  41% ( 82K/200K) | █████████░  94% (4h36m/5h) | ███████░░░  79% (2d20h/7d)
Δ+3.4K/-1.3K | Cache:99%(W:0K/R:82K) | Insert | v2.1.272
Bash(5) Read(3) Edit(1)

Changes apply on the next refresh: the next event, or the `refreshInterval` timer when one is configured (the installer defaults it to 10 seconds).
```

## Examples

### Example 1: Preset + Feature Override

User wants `developer` preset but also wants the API wait ratio (not in `developer`):

```json
{
  "preset": "developer",
  "features": {
    "api_wait_ratio": true
  }
}
```

### Example 2: Minimal + Selective Additions

User wants `minimal` but adds git and cache:

```json
{
  "preset": "minimal",
  "features": {
    "git": true,
    "cache_efficiency": true
  }
}
```

## Reset to Default

To reset to `full` preset with no overrides:

```bash
rm ~/.claude/hud/config.json
```

Or set explicitly:

```bash
echo '{"preset":"full"}' > ~/.claude/hud/config.json
```

## Current Configuration

To view current config:

```bash
cat ~/.claude/hud/config.json
```

## Important Notes

### Danger Mode Override

**When context usage reaches the danger threshold (default 85%), Howl automatically switches to full information mode regardless of your configuration.** This ensures complete visibility during critical situations.

This override cannot be disabled - it's a safety feature. The trigger point can be adjusted via `/howl:threshold` or the `context_danger` field in config.json.

### Configuration Validation

- Invalid preset names fall back to `full`
- Feature toggles only accept known metrics (others ignored silently)
- Any other key, `priority` included, is ignored silently — the binary reads only `preset`, `features` and `thresholds`
- Config file size limited to 4KB (DoS protection)

### Line Placement Rules

Fixed in `renderNormalMode` in `internal/render.go`; read it there rather than trusting a summary. In outline:

- **Line 1**: model badge, then account, git, output tokens, cost, duration. When `quota` is off, the context bar moves into this line.
- **Line 2** (only with `quota`): context bar, 5h quota bar, 7d quota bar.
- **Line 3**: line changes, cache, API wait, cost velocity, vim mode, the optional fields, then the Claude Code version (always).
- **Line 4**: tools and running agents.

Danger mode ignores the toggles and always prints its own two lines.

### Refresh Rate

Configuration changes apply on the next statusline refresh — the next event, or the `refreshInterval` timer when one is configured (the installer defaults it to 10 seconds). No restart needed.

### Quick Switch Between Presets

If user just wants to switch presets without customization, recommend using `/howl:configure` instead - it's faster for simple preset changes.

### Color Thresholds

To customize when colors change (e.g., danger mode trigger, cost warning levels), use `/howl:threshold` instead. This skill focuses on **which** metrics are displayed; `/howl:threshold` controls **when** they change color.

## Example Dialogue

```
User: I want more control over what's shown
Agent: I can help customize that! Let's walk through it.

[Step 1] Which base preset?
> developer

[Step 2] Customize metrics (pre-checked based on developer):
☑ account, git, line_changes, output_tokens, quota, tools, agents, cache_efficiency, vim_mode
☐ api_wait_ratio, cost_velocity, agent_name, effort, thinking, session_name, pull_request, worktree, prompt_cache, fast_mode, exceeds_200k, output_style, repo, added_dirs
> User also checks: api_wait_ratio

Applying configuration...
✅ Config applied: developer + api_wait_ratio
Preview: [Opus 4.6] | user@example.com | main* | ...

Changes apply on the next refresh (the next event, or the `refreshInterval` timer when one is configured (the installer defaults it to 10 seconds)).
```
