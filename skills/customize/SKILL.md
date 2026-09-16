---
description: Customize Howl statusline with fine-grained metric toggles and priority ordering
disable-model-invocation: false
---

# Howl Customize

Advanced configuration for Howl statusline: choose a base preset, toggle individual metrics, and set display priority for Line 2.

## Configuration Structure

```json
{
  "preset": "developer",
  "features": {
    "quota": true
  },
  "priority": ["quota", "git"],
  "thresholds": {
    "context_danger": 90
  }
}
```

- **preset**: Base configuration (`full`, `minimal`, `developer`, `cost-focused`)
- **features**: Override specific metrics from the preset base (optional)
- **priority**: Reorder Line 2 metrics by importance (optional, max 5)
- **thresholds**: Override color/behavior breakpoints (optional, see `/howl:threshold`)

## Process

### Step 1: Choose Base Preset

**Use AskUserQuestion to present preset choices:**

- **Question**: "Which base preset would you like to start with?"
- **Header**: "Choose Base Preset"
- **Options** (4):
  - Label: **"full (default)"**  
    Description: "All 13 metrics - Complete visibility (2-4 lines)"
  - Label: **"minimal"**  
    Description: "Model + Context + Cost + Duration only (1 line)"
  - Label: **"developer"**  
    Description: "Coding focus: Account, Git, Changes, Cache, Vim (2 lines)"
  - Label: **"cost-focused"**  
    Description: "Budget tracking: Quota, API Wait, Cost Velocity (2 lines)"

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
| `output_tokens`    | Output tokens of the current response (`Out:1K`)                                                               | off        |
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

- **full**: the eleven marked "on in full"; every toggle marked off stays unchecked
- **minimal**: none
- **developer**: account, git, line_changes, cache_efficiency, vim_mode
- **cost-focused**: quota, api_wait_ratio, cost_velocity

**Features are additive-only.** Checking enables; unchecking does **not** disable,
because `mergeFeatures` can only turn a flag on. To drop something the preset
includes, start from `minimal` in Step 1 and check only what is wanted.

- Want `full` without git? Start from `minimal` and check everything except git.
- Want `developer` plus quota? Start from `developer` and check quota.

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

### Step 3: Set Display Priority (Line 2 Only)

**Use AskUserQuestion with multiSelect for priority:**

- **Question**: "Choose which metrics should appear first on Line 2 (max 5, ordered by selection)"
- **Header**: "Display Priority (Optional)"
- **Subtitle**: "Only Line 2 metrics can be prioritized. Selected order = display order."
- **Options** (4 checkboxes, only Line 2 metrics):
  1. **account** - Account email
  2. **git** - Git branch + status
  3. **line_changes** - Code additions/deletions
  4. **quota** - Usage quota visualization

**Constraints:**

- Max 5 selections
- Selection order determines display order
- Only show metrics that are **enabled** in the feature toggles from Step 2
- If user selects 0 metrics, omit `priority` from config.json

**Store selections as `priorityOrder` array (preserving order).**

### Step 4: Generate and Apply Configuration

**Build the config object:**

```json
{
  "preset": "<chosenPreset>",
  "features": {
    // Only include if different from preset base
    // Format: "metric_name": true/false
  },
  "priority": [
    // Only include if user selected 1+ metrics
    // Format: ["metric1", "metric2", ...]
  ]
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
Overrides: quota (enabled)
Priority: quota → git

Preview (example):
[Sonnet 4.5] | ████░░░░░░░░░░░░░░░░ 21% (210K/1M) | $32.7 | 2h46m
(2h)5h: 55%/42% :7d(3d6h) | user@example.com | main* | +2.7K/-120 | Cache:96% | I

Changes will apply on next refresh (~300ms).
```

## Examples

### Example 1: Preset + Feature Override

User wants `developer` preset but also wants quota visualization:

```json
{
  "preset": "developer",
  "features": {
    "quota": true
  }
}
```

### Example 2: Full Customization with Priority

User wants `full` preset but prioritizes git and quota on Line 2:

```json
{
  "preset": "full",
  "priority": ["git", "quota"]
}
```

### Example 3: Minimal + Selective Additions

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

### Example 4: Cost-focused with Custom Priority

User wants `cost-focused` and reorders Line 2:

```json
{
  "preset": "cost-focused",
  "priority": ["quota"]
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
- Priority only accepts **Line 2 metrics** (others ignored)
- Priority is capped at **5 metrics maximum**
- Duplicate entries in priority are removed
- Config file size limited to 4KB (DoS protection)

### Line Placement Rules

- **Line 1**: Model badge, context bar, cost, duration (always shown)
- **Line 2**: account, git, line_changes, quota (prioritizable)
- **Line 3**: tools, agents (only in `full` preset or danger mode)
- **Line 4**: cache_efficiency, api_wait_ratio, cost_velocity, vim_mode, agent_name (only in `full` or danger mode)
- **Optional** (default off): effort, thinking, session_name, pull_request, worktree

### Refresh Rate

Configuration changes apply on the next statusline refresh (~300ms). No restart needed.

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
☑ account, git, line_changes, cache_efficiency, vim_mode
☐ quota, tools, agents, api_wait_ratio, cost_velocity, agent_name, effort, thinking, session_name, pull_request, worktree
> User also checks: quota

[Step 3] Priority for Line 2 (max 5):
> User selects: quota, git (in that order)

Applying configuration...
✅ Config applied: developer + quota, priority: quota → git
Preview: (2h)5h: 55%/42% :7d(3d6h) | user@example.com | ...

Changes will apply in ~300ms.
```
