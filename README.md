# Howl

> *"Your AI screams — Howl listens."*

A blazing-fast, feature-rich statusline HUD for [Claude Code](https://code.claude.com) written in Go. Provides real-time visibility into your AI coding session with intelligent metrics, usage tracking, and adaptive layouts.

[![Language](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev)
[![Size](https://img.shields.io/badge/Binary-5.2MB-blue)]()
[![Speed](https://img.shields.io/badge/Cold%20Start-~10ms-green)]()

---

## Features

### 📊 **Intelligent Metrics**
- **Cache Efficiency** — Track prompt cache utilization (90%+ = optimal)
- **API Wait Ratio** — See how much time spent waiting for AI responses
- **Response Speed** — Real-time tokens/second output rate
- **Cost Velocity** — Monitor spending rate ($/minute)

### 🎯 **Essential Status**
- **Model Tier Badge** — Color-coded Opus (gold) / Sonnet (cyan) / Haiku (green)
- **Context Health Bar** — Visual 20-char bar with 4-tier gradient
- **Token Absolutes** — See exact usage (210K/1000K) not just percentages
- **Usage Quota** — Live 5h/7d limits with reset countdowns

### 🔧 **Workflow Awareness**
- **Git Integration** — Branch name + dirty status (`main*`)
- **Code Changes** — Track lines added/removed with color coding
- **Tool Usage** — Top 5 most-used tools (Read, Bash, Edit...)
- **Active Agents** — See running subagents in real-time
- **Vim Mode** — N/I/V indicators for modal editing

### 🎨 **Adaptive Layouts**
- **Normal Mode** (< 85% context) — Clean 3-4 line display
- **Danger Mode** (85%+ context) — Expanded view with token breakdown
- **Smart Grouping** — Logical organization of related metrics

---

## Installation

### Prerequisites
- Go 1.23+ (for building from source)
- macOS with Keychain access (for OAuth quota tracking)
- Claude Code CLI installed

### Quick Install

```bash
# Clone the repository
git clone https://github.com/ai-scream/howl.git
cd howl

# Build and install
make install

# The binary will be installed to ~/.claude/hud/howl
```

### Configure Claude Code

Add to your `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "/Users/YOUR_USERNAME/.claude/hud/howl"
  }
}
```

Restart Claude Code to see Howl in action.

---

## Usage

Howl runs automatically as a subprocess every ~300ms. No manual interaction needed.

### Example Output

**Normal Session (21% context):**
```
[Sonnet 4.5 (1M)] | ████░░░░░░░░░░░░░░░░ 21% (210K/1000K) | $32.7 | 2h46m
main* | +2.7K/-120 | 50tok/s | (2h)5h: 55%/42% :7d(3d6h)
Read(9) Bash(8) TaskCreate(4) mcp__context7(3)
Cache:96% | Wait:41% | Cost:$0.19/m | I
```

**Danger Mode (87% context):**
```
🔴 [Opus 4.6] | ████████████████░░ 87% (174K/200K) | $15.7 | 1h23m
hud/main* | +850/-45 | In:30K Out:3K Cache:135K | 11tok/s | (1h)5h: 25%/18% :7d(2d)
▶code-writer | Cache:79% | Wait:24% | $11.3/h | I | @code-wri
```

### Metrics Explained

| Metric | Meaning | Color Coding |
|--------|---------|--------------|
| **Cache:96%** | Prompt cache efficiency (% of input from cache) | Green (80%+), Yellow (50-80%), Red (<50%) |
| **Wait:41%** | Time spent waiting for API responses | Green (<35%), Yellow (35-60%), Red (60%+) |
| **Cost:$0.19/m** | API spending rate per minute | Green (<$0.10), Yellow ($0.10-0.50), Red ($0.50+) |
| **50tok/s** | Output token generation speed | Green (60+), Yellow (30-60), Orange (<30) |
| **(2h)5h: 55%** | 5-hour quota: 55% remaining, resets in 2 hours | Gradient based on % remaining |
| **:7d(3d6h)** | 7-day quota: 42% remaining, resets in 3d6h | Gradient based on % remaining |

---

## Architecture

### Data Flow

```
Claude Code (every ~300ms)
    │
    ├─ Pipes JSON to stdin
    │
    ▼
┌─────────────────────────────────────┐
│  Howl Binary (Go)                   │
│                                     │
│  1. Parse stdin JSON                │
│  2. Compute derived metrics         │
│  3. Fetch git status (1s timeout)   │
│  4. Get OAuth quota (60s cache)     │
│  5. Parse transcript (last 100 ln)  │
│  6. Render ANSI output              │
│  7. Output to stdout                │
└─────────────────────────────────────┘
    │
    ▼
Claude Code Statusline Display
```

### Project Structure

```
howl/
├── cmd/
│   └── howl/
│       └── main.go          # Entry point, orchestration
├── internal/
│   ├── types.go             # StdinData structs
│   ├── metrics.go           # Derived calculations
│   ├── render.go            # ANSI output generation
│   ├── git.go               # Git subprocess calls
│   ├── usage.go             # OAuth quota API
│   └── transcript.go        # JSONL parsing
├── docs/                    # Design documents
├── Makefile                 # Build automation
└── go.mod                   # Go module definition
```

### Key Modules

- **types.go** — StdinData schema matching Claude Code's JSON output
- **metrics.go** — Cache efficiency, API ratio, cost velocity calculations
- **render.go** — ANSI color codes, adaptive layouts (normal/danger)
- **git.go** — Branch detection with graceful timeout
- **usage.go** — Anthropic OAuth API client with session-scoped caching
- **transcript.go** — Tool usage extraction from conversation history

---

## Performance

| Scenario | Execution Time | Notes |
|----------|----------------|-------|
| stdin-only | ~10ms | JSON parse + render |
| + git status | ~50ms | 1s timeout, graceful |
| + transcript | ~100ms | Last 100 lines only |
| + OAuth quota | ~3s first, ~10ms cached | 60s cache TTL |

**Optimizations:**
- Compiled Go binary (no interpreter startup)
- Session-scoped caching for external API calls
- Tail-only transcript parsing (vs full file scan)
- 1-second timeout on git operations

---

## Development

### Build from Source

```bash
# Install dependencies (none! stdlib only)
go mod download

# Build
make build

# Run tests
make test

# Install to ~/.claude/hud/
make install
```

### Project Commands

```bash
make build    # Compile to build/howl
make install  # Copy to ~/.claude/hud/howl
make clean    # Remove build artifacts
make test     # Run with sample JSON input
```

### Adding New Metrics

1. Add field to `Metrics` struct in `internal/metrics.go`
2. Implement calculation function
3. Call in `ComputeMetrics()`
4. Add render function in `internal/render.go`
5. Integrate into layout (normal/danger modes)

Example:
```go
// metrics.go
type Metrics struct {
    // ...
    NewMetric *int
}

func calcNewMetric(d *StdinData) *int {
    // calculation logic
}

// render.go
func renderNewMetric(val int) string {
    return fmt.Sprintf("%s%d%s", color, val, Reset)
}
```

---

## Configuration

### OAuth Credentials

Howl automatically reads OAuth tokens from macOS Keychain:
- Service: `Claude Code-credentials`
- Extracted field: `claudeAiOauth.accessToken`

No manual configuration needed if Claude Code is authenticated.

### Cache Locations

- **Usage quota cache:** `/tmp/howl-{session_id}/usage.json` (60s TTL)
- **Session-scoped:** Each Claude Code session has isolated cache

---

## Troubleshooting

### Quota shows `?`
- OAuth API unavailable or credentials expired
- Check: `security find-generic-password -s "Claude Code-credentials" -w`
- Fallback: Quota display is optional, other metrics still work

### Git branch not showing
- Not a git repository
- Git timeout (1s) exceeded
- Solution: Initialize git or ignore (graceful degradation)

### Tools line empty
- Transcript file not accessible
- Session just started (no tools used yet)
- Solution: Wait for tool usage or check transcript path

### Performance slower than expected
- Large transcript file (>10MB)
- Network latency for OAuth API
- Solution: Transcript parses last 100 lines only, quota cached for 60s

---

## Why Howl?

Howl was created to solve specific pain points with existing Claude Code statusline tools:

### Problems with claude-hud
- ❌ **Cross-session bugs** — Global cache shared between sessions
- ❌ **OAuth API blocked** — Missing `anthropic-beta` header
- ❌ **Limited metrics** — No cache efficiency or wait ratio
- ❌ **Node.js dependency** — 70ms cold start overhead

### Howl Solutions
- ✅ **Session isolation** — Cache per `session_id`
- ✅ **OAuth working** — Correct API headers discovered
- ✅ **Rich metrics** — 13 distinct indicators
- ✅ **Go performance** — 10ms cold start, 2MB binary

---

## Roadmap

- [ ] Configuration file support (`~/.claude/hud/config.json`)
- [ ] Custom color schemes
- [ ] Plugin system for custom metrics
- [ ] Multi-platform support (Linux, Windows)

---

## Contributing

This is a personal tool for the AiScream project. Feedback and bug reports welcome!

---

## License

MIT License — see LICENSE file for details.

---

## Credits

**Project:** [ai-scream/howl](https://github.com/ai-scream/howl)
**Author:** hanyul
**Inspired by:** [claude-hud](https://github.com/jarrodwatts/claude-hud) by Jarrod Watts

Built with ❤️ and Claude Code.
