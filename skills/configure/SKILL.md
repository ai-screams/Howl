---
description: Configure Howl statusline display preset with interactive guidance and preview
disable-model-invocation: false
---

# Howl Configure

Help the user choose a display preset for their Howl statusline HUD.

## Available Presets

| Preset           | Lines | Best For            | Displays                                |
| ---------------- | ----- | ------------------- | --------------------------------------- |
| **full**         | 2-4   | Complete visibility | All metrics (default)                   |
| **minimal**      | 1     | Clean workspace     | Model + Context + Cost + Duration       |
| **developer**    | 2     | Coding focus        | + Account + Git + Changes + Cache + Vim |
| **cost-focused** | 2     | Budget tracking     | + Quota + API Wait + Cost Velocity      |

## Process

1. **Use the AskUserQuestion tool to present preset choices as an interactive UI:**

   - Question: "Which Howl statusline preset would you like to use?"
   - Header: "Display Preset"
   - Options (4):
     - Label: "full (default)" | Description: "Complete visibility - All metrics (2-4 lines)"
     - Label: "minimal" | Description: "Clean workspace - Model + Context + Cost + Duration only (1 line)"
     - Label: "developer" | Description: "Coding focus - full minus API wait, cost velocity and agent name (2 lines)"
     - Label: "cost-focused" | Description: "Budget tracking - Account, Output tokens, Quota, API wait, Cost velocity (2 lines)"

2. **Show a preview** of their chosen preset:

   **minimal:**

   ```
   [Sonnet 4.5] | ████░░░░░░░░░░░░░░░░ 21% (210K/1M) | $32.7 | 2h46m
   ```

   **developer:**

   ```
   [Sonnet 4.5] | ████░░░░░░░░░░░░░░░░ 21% (210K/1M) | $32.7 | 2h46m
   user@example.com | main* | +2.7K/-120 | Cache:96% | I
   ```

   **cost-focused:**

   ```
   [Sonnet 4.5] | ████░░░░░░░░░░░░░░░░ 21% (210K/1M) | $32.7 | 2h46m
   $0.19/m | Wait:41% | (2h)5h: 55%/42% :7d(3d6h)
   ```

   **full:**

   ```
   [Sonnet 4.5] | ████░░░░░░░░░░░░░░░░ 21% (210K/1M) | $32.7 | 2h46m
   user@example.com | main* | +2.7K/-120 | (2h)5h: 55%/42% :7d(3d6h)
   Read(9) Bash(8) TaskCreate(4) | ▶researcher,tester
   Cache:96% | Wait:41% | Cost:$0.19/m | I | @code-wri
   ```

3. **Apply the configuration:**

   ```bash
   mkdir -p ~/.claude/hud
   cat > ~/.claude/hud/config.json << 'EOF'
   {
     "preset": "CHOSEN_PRESET"
   }
   EOF
   ```

4. **Confirm** the change:
   "✅ Preset set to **CHOSEN_PRESET**. Changes will appear in the next statusline refresh (~300ms)."

## Reset to Default

To reset to full (default):

```bash
rm ~/.claude/hud/config.json
```

Or set explicitly:

```bash
echo '{"preset":"full"}' > ~/.claude/hud/config.json
```

## Current Configuration

To see your current preset:

```bash
cat ~/.claude/hud/config.json
```

If the file doesn't exist, you're using `full` (default).

## Important Notes

### Danger Mode Override

**When context usage reaches the danger threshold (default 85%), Howl automatically switches to full information mode regardless of your chosen preset.** This ensures you have complete visibility during critical situations when you're approaching context limits.

This behavior cannot be disabled - it's a safety feature. The trigger point can be adjusted via `/howl:threshold` or the `context_danger` field in `~/.claude/hud/config.json`.

### What Gets Hidden in Each Preset

- **minimal**: Hides account, git, line changes, quota, tools, agents, cache, API wait ratio, cost velocity, vim mode, agent name, token breakdown
- **developer**: Hides quota, tools, agents, API wait ratio, cost velocity, agent name, token breakdown
- **cost-focused**: Hides account, git, line changes, tools, agents, cache, vim mode, agent name, token breakdown

### Refresh Rate

Configuration changes apply on the next statusline refresh, which occurs approximately every 300ms. You don't need to restart Claude Code.

## Example Dialogue

```
User: I want a cleaner statusline
Agent: I can help configure that! What's your priority?
- Minimal (just essentials: model, context, cost, time)
- Developer focus (+ git, changes, cache)
- Cost tracking (+ quota, API wait, velocity)

User: Minimal please
Agent: Here's what minimal will look like:
[Sonnet 4.5] | ████░░░░░░░░░░░░░░░░ 21% (210K/1M) | $32.7 | 2h46m

Applying configuration...
✅ Preset set to minimal. Changes will appear in ~300ms.

Note: At the danger threshold (default 85%), Howl will temporarily show full info for safety.
You can adjust this via /howl:threshold.
```
