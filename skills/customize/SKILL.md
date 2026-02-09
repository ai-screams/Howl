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
  "priority": ["quota", "git"]
}
```

- **preset**: Base configuration (`full`, `minimal`, `developer`, `cost-focused`)
- **features**: Override specific metrics from the preset base (optional)
- **priority**: Reorder Line 2 metrics by importance (optional, max 5)

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
    Description: "Coding focus: Account, Git, Changes, Speed, Cache, Vim (2 lines)"
  - Label: **"cost-focused"**  
    Description: "Budget tracking: Quota, API Wait, Cost Velocity (2 lines)"

**Store the user's selection as `chosenPreset`.**

### Step 2: Toggle Individual Metrics

**Use AskUserQuestion with multiSelect to show metric toggles:**

- **Question**: "Select which metrics to display (pre-checked = enabled in your preset)"
- **Header**: "Customize Metrics"
- **Options** (12 checkboxes):
  1. **account** - Account email
  2. **git** - Git branch + status
  3. **line_changes** - Code additions/deletions
  4. **response_speed** - Tokens per second
  5. **quota** - Usage quota visualization
  6. **tools** - Tool call counts
  7. **agents** - Active agent indicators
  8. **cache_efficiency** - Cache hit percentage
  9. **api_wait_ratio** - API wait time ratio
  10. **cost_velocity** - Cost per minute
  11. **vim_mode** - Vim mode indicator
  12. **agent_name** - Current agent name

**Pre-check based on `chosenPreset`:**

- **full**: All 12 checked
- **minimal**: None checked
- **developer**: account, git, line_changes, response_speed, cache_efficiency, vim_mode
- **cost-focused**: quota, api_wait_ratio, cost_velocity

**Important: Features are Additive-Only**

- ✅ **Checking** a metric enables it (adds to preset)
- ❌ **Unchecking** does NOT disable it (preset base is preserved)
- To disable features from `full`, start with `minimal` and check only what you want

**Example:**

- Want `full` without git? → Use Step 1: `minimal`, Step 2: check all except git
- Want `developer` + quota? → Use Step 1: `developer`, Step 2: check quota

**Important Notes:**

- Model badge, context bar, cost, and duration are **always displayed** (cannot be toggled)
- If user selects the same set as the preset base, omit `features` from config.json (cleaner)
- If user changes any toggles, record differences in `features` object

**Store selections as `selectedFeatures` array.**

### Step 3: Set Display Priority (Line 2 Only)

**Use AskUserQuestion with multiSelect for priority:**

- **Question**: "Choose which metrics should appear first on Line 2 (max 5, ordered by selection)"
- **Header**: "Display Priority (Optional)"
- **Subtitle**: "Only Line 2 metrics can be prioritized. Selected order = display order."
- **Options** (5 checkboxes, only Line 2 metrics):
  1. **account** - Account email
  2. **git** - Git branch + status
  3. **line_changes** - Code additions/deletions
  4. **response_speed** - Tokens per second
  5. **quota** - Usage quota visualization

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
(2h)5h: 55%/42% :7d(3d6h) | user@example.com | main* | +2.7K/-120 | 50tok/s | Cache:96% | I

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

**When context usage reaches 85% or higher, Howl automatically switches to full information mode regardless of your configuration.** This ensures complete visibility during critical situations.

This override cannot be disabled - it's a safety feature to prevent context exhaustion.

### Configuration Validation

- Invalid preset names fall back to `full`
- Feature toggles only accept known metrics (others ignored silently)
- Priority only accepts **Line 2 metrics** (others ignored)
- Priority is capped at **5 metrics maximum**
- Duplicate entries in priority are removed
- Config file size limited to 4KB (DoS protection)

### Line Placement Rules

- **Line 1**: Model badge, context bar, cost, duration (always shown)
- **Line 2**: account, git, line_changes, response_speed, quota (prioritizable)
- **Line 3**: tools, agents (only in `full` preset or danger mode)
- **Line 4**: cache_efficiency, api_wait_ratio, cost_velocity, vim_mode, agent_name (only in `full` or danger mode)

### Refresh Rate

Configuration changes apply on the next statusline refresh (~300ms). No restart needed.

### Quick Switch Between Presets

If user just wants to switch presets without customization, recommend using `/howl:configure` instead - it's faster for simple preset changes.

## Example Dialogue

```
User: I want more control over what's shown
Agent: I can help customize that! Let's walk through it.

[Step 1] Which base preset?
> developer

[Step 2] Customize metrics (pre-checked based on developer):
☑ account, git, line_changes, response_speed, cache_efficiency, vim_mode
☐ quota, tools, agents, api_wait_ratio, cost_velocity, agent_name
> User also checks: quota

[Step 3] Priority for Line 2 (max 5):
> User selects: quota, git (in that order)

Applying configuration...
✅ Config applied: developer + quota, priority: quota → git
Preview: (2h)5h: 55%/42% :7d(3d6h) | user@example.com | ...

Changes will apply in ~300ms.
```
