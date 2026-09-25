---
description: Customize Howl color thresholds and behavior triggers interactively
disable-model-invocation: false
---

# Howl Threshold

Customize when Howl changes colors and switches modes. All 15 threshold values are configurable — they control when metrics turn green/yellow/orange/red and when danger mode activates.

## Threshold Groups

| Group             | Thresholds                                                  | Defaults                     | Effect                                      |
| ----------------- | ----------------------------------------------------------- | ---------------------------- | ------------------------------------------- |
| **Context**       | `context_danger`, `context_warning`, `context_moderate`     | 85%, 70%, 50%                | Danger mode trigger, warning/moderate color |
| **Session Cost**  | `session_cost_high`, `session_cost_medium`                  | $5.00, $1.00                 | Cost display color                          |
| **Cache**         | `cache_excellent`, `cache_good`                             | 80%, 50%                     | Cache efficiency color                      |
| **API Wait**      | `wait_high`, `wait_medium`                                  | 60%, 35%                     | API wait ratio color                        |
| **Cost Velocity** | `cost_velocity_high`, `cost_velocity_medium`                | $0.50, $0.10/min             | Cost velocity color                         |
| **Quota**         | `quota_critical`, `quota_low`, `quota_medium`, `quota_high` | 10%, 25%, 50%, 75% remaining | Quota color bands                           |

## Configuration Structure

```json
{
  "preset": "full",
  "thresholds": {
    "context_danger": 90,
    "context_warning": 75,
    "quota_critical": 5
  }
}
```

Only specified thresholds override defaults. Omitted fields keep default values.

## Process

### Step 1: Choose Action

**Use AskUserQuestion:**

- **Question**: "What would you like to do with Howl thresholds?"
- **Header**: "Thresholds"
- **Options** (4):
  - Label: **"Customize"** | Description: "Change specific threshold values"
  - Label: **"View Current"** | Description: "Show all current threshold values"
  - Label: **"Reset All"** | Description: "Remove all custom thresholds (restore defaults)"
  - Label: **"Quick Adjust"** | Description: "Change just the danger mode trigger point"

**If "View Current":**

1. Read `~/.claude/hud/config.json` (if exists)
2. Display all 15 thresholds in a table, marking custom values with `*`
3. Done.

**If "Reset All":**

1. Read existing config.json
2. Remove the `thresholds` field, preserving `preset`, `features`, and `priority`
3. Write back
4. Confirm: "Thresholds reset to defaults. Changes apply on the next refresh (at most the 10-second `refreshInterval`)."
5. Done.

**If "Quick Adjust":**

1. Ask: "At what context percentage should danger mode activate? (current: 85%)"
2. Validate: must be 50-100
3. Write `context_danger` to config.json (preserving other fields)
4. Confirm with before/after comparison
5. Done.

**If "Customize", continue to Step 2.**

### Step 2: Select Threshold Group

**Use AskUserQuestion:**

- **Question**: "Which threshold group would you like to customize?"
- **Header**: "Select Group"
- **Options** (4):
  - Label: **"Context & Danger"** | Description: "When danger mode activates (85%) and warning shows (70%)"
  - Label: **"Performance"** | Description: "Cache (80/50%), API Wait (60/35%)"
  - Label: **"Cost"** | Description: "Session cost ($5/$1), Cost velocity ($0.50/$0.10/min)"
  - Label: **"Quota"** | Description: "Quota color bands (10/25/50/75% remaining)"

### Step 3: Set Values

Based on selected group, ask for specific values.

**Context & Danger:**

- Ask: "Set danger mode trigger (current default: 85%, must be 50-100):"
- Ask: "Set warning indicator (current default: 70%, must be less than danger):"

**Performance:**

- Ask: "Set cache thresholds — Excellent (green, default 80%) and Good (yellow, default 50%):"
- Ask: "Set API wait thresholds — High (red, default 60%) and Medium (yellow, default 35%):"

**Cost:**

- Ask: "Set session cost thresholds — High (red, default $5.00) and Medium (yellow, default $1.00):"
- Ask: "Set cost velocity thresholds — High (red, default $0.50/min) and Medium (yellow, default $0.10/min):"

**Quota:**

- Ask: "Set quota color bands (% remaining) — Critical (bold red, default 10), Low (red, default 25), Medium (orange, default 50), High (yellow, default 75):"

**Validation Rules:**

- All values must be positive numbers
- For paired thresholds: high > low (e.g., danger > warning, fast > moderate)
- For quota: critical < low < medium < high
- Reject invalid values with explanation

### Step 4: Apply Configuration

1. Read existing `~/.claude/hud/config.json` (or start with `{}`)
2. Merge new thresholds into existing config (preserve `preset`, `features`, `priority`)
3. Write back:

```bash
mkdir -p ~/.claude/hud
cat > ~/.claude/hud/config.json << 'EOF'
{JSON_WITH_MERGED_THRESHOLDS}
EOF
```

4. Show confirmation with before/after comparison:

```
Thresholds Updated

  context_danger: 85 → 90
  context_warning: 70 → 75

All other thresholds remain at defaults.
Changes apply on the next refresh (at most the 10-second `refreshInterval`).
```

5. Ask: "Would you like to customize another group?"

## Examples

### Example 1: Raise Danger Threshold

User wants danger mode at 90% instead of 85%:

```json
{
  "preset": "full",
  "thresholds": {
    "context_danger": 90
  }
}
```

### Example 2: Stricter Cache Thresholds

User wants higher standards for cache efficiency:

```json
{
  "preset": "developer",
  "thresholds": {
    "cache_excellent": 90,
    "cache_good": 70
  }
}
```

### Example 3: Relaxed Cost Thresholds

User on a team plan with higher budget:

```json
{
  "preset": "cost-focused",
  "thresholds": {
    "session_cost_high": 20.0,
    "session_cost_medium": 5.0,
    "cost_velocity_high": 2.0,
    "cost_velocity_medium": 0.5
  }
}
```

### Example 4: Aggressive Quota Alerts

User wants earlier quota warnings:

```json
{
  "preset": "full",
  "thresholds": {
    "quota_critical": 20,
    "quota_low": 40,
    "quota_medium": 60,
    "quota_high": 85
  }
}
```

## Important Notes

### Merging Behavior

- Custom thresholds are **merged** with defaults — only specified values change
- Zero or negative values are ignored (treated as "use default")
- Existing `preset`, `features`, and `priority` fields are always preserved

### Danger Mode

- The `context_danger` threshold controls when Howl switches to compact 2-line danger mode
- Setting it higher (e.g., 95%) gives you more room before the mode switch
- Setting it lower (e.g., 75%) triggers earlier warnings
- **Cannot be disabled entirely** — it's a safety feature

### Config File

- Location: `~/.claude/hud/config.json`
- Size limit: 4KB
- Invalid JSON falls back to all defaults silently

### Refresh Rate

Changes apply on the next statusline refresh — the next event, or the `refreshInterval` timer, 10 seconds as `/howl:setup` sets it. No restart needed.

## Example Dialogue

```
User: I want danger mode to trigger later
Agent: Currently danger mode triggers at 85% context usage.

What should the new trigger point be? (50-100)
> 92

And the warning indicator? (currently 70%, must be less than 92)
> 80

Applying...

✅ Thresholds Updated
  context_danger: 85 → 92
  context_warning: 70 → 80

Danger mode will now activate at 92% context usage.
Changes apply on the next refresh (at most the 10-second `refreshInterval`).
```
