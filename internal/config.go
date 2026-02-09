package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const maxConfigSize = 4096 // 4KB limit

// validLine2Metrics defines the allowed metrics for Priority (Line 2 only).
var validLine2Metrics = map[string]bool{
	"account":        true,
	"git":            true,
	"line_changes":   true,
	"response_speed": true,
	"quota":          true,
}

// Config represents user's statusline configuration.
type Config struct {
	Preset   string         `json:"preset"`
	Features FeatureToggles `json:"features"` // v1.1: override preset base
	Priority []string       `json:"priority"` // v1.1: Line 2 metric order (max 5)
}

// FeatureToggles controls which metrics are displayed.
type FeatureToggles struct {
	Account         bool `json:"account"`
	Git             bool `json:"git"`
	LineChanges     bool `json:"line_changes"`
	ResponseSpeed   bool `json:"response_speed"`
	Quota           bool `json:"quota"`
	Tools           bool `json:"tools"`
	Agents          bool `json:"agents"`
	CacheEfficiency bool `json:"cache_efficiency"`
	APIWaitRatio    bool `json:"api_wait_ratio"`
	CostVelocity    bool `json:"cost_velocity"`
	VimMode         bool `json:"vim_mode"`
	AgentName       bool `json:"agent_name"`
	// TokenBreakdown bool `json:"token_breakdown"` // Reserved for danger mode only (v1.1+)
}

var presets = map[string]FeatureToggles{
	"full": {
		Account:         true,
		Git:             true,
		LineChanges:     true,
		ResponseSpeed:   true,
		Quota:           true,
		Tools:           true,
		Agents:          true,
		CacheEfficiency: true,
		APIWaitRatio:    true,
		CostVelocity:    true,
		VimMode:         true,
		AgentName:       true,
	},
	"minimal": {}, // all false
	"developer": {
		Account:         true,
		Git:             true,
		LineChanges:     true,
		ResponseSpeed:   true,
		CacheEfficiency: true,
		VimMode:         true,
	},
	"cost-focused": {
		Quota:        true,
		APIWaitRatio: true,
		CostVelocity: true,
	},
}

// DefaultConfig returns full preset (current behavior).
// mergeFeatures merges override into base. If override field is true, it overrides base.
// If override field is false, base value is preserved (no reflection, explicit for all 12 fields).
func mergeFeatures(base, override FeatureToggles) FeatureToggles {
	result := base
	if override.Account {
		result.Account = true
	}
	if override.Git {
		result.Git = true
	}
	if override.LineChanges {
		result.LineChanges = true
	}
	if override.ResponseSpeed {
		result.ResponseSpeed = true
	}
	if override.Quota {
		result.Quota = true
	}
	if override.Tools {
		result.Tools = true
	}
	if override.Agents {
		result.Agents = true
	}
	if override.CacheEfficiency {
		result.CacheEfficiency = true
	}
	if override.APIWaitRatio {
		result.APIWaitRatio = true
	}
	if override.CostVelocity {
		result.CostVelocity = true
	}
	if override.VimMode {
		result.VimMode = true
	}
	if override.AgentName {
		result.AgentName = true
	}
	return result
}

// validatePriority normalizes, deduplicates, and validates priority list.
// Returns only Line 2 metrics (max 5), lowercased and deduplicated.
func validatePriority(input []string) []string {
	if len(input) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	result := make([]string, 0, 5)

	for _, metric := range input {
		// Normalize: lowercase + trim
		normalized := strings.ToLower(strings.TrimSpace(metric))
		if normalized == "" {
			continue
		}

		// Skip duplicates
		if seen[normalized] {
			continue
		}

		// Validate: only Line 2 metrics
		if !validLine2Metrics[normalized] {
			continue
		}

		seen[normalized] = true
		result = append(result, normalized)

		// Max 5 items
		if len(result) >= 5 {
			break
		}
	}

	return result
}

func DefaultConfig() Config {
	return PresetConfig("full")
}

// PresetConfig returns a Config with the specified preset.
// Used for testing. If preset is unknown, returns full.
func PresetConfig(name string) Config {
	name = strings.ToLower(strings.TrimSpace(name))
	if features, ok := presets[name]; ok {
		return Config{Preset: name, Features: features}
	}
	return Config{Preset: "full", Features: presets["full"]}
}

// LoadConfig loads configuration from ~/.claude/hud/config.json.
// On any error (file not found, parse error, invalid preset), returns DefaultConfig().
func LoadConfig() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultConfig()
	}

	path := filepath.Join(home, ".claude", "hud", "config.json")

	// Guard: check file size before reading
	stat, err := os.Stat(path)
	if err != nil {
		return DefaultConfig() // File not found
	}
	if stat.Size() > maxConfigSize {
		return DefaultConfig() // Too large, DoS protection
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig()
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig() // Malformed JSON
	}

	// Normalize preset name: lowercase + trim whitespace
	cfg.Preset = strings.ToLower(strings.TrimSpace(cfg.Preset))
	if cfg.Preset == "" {
		cfg.Preset = "full"
	}

	// Get preset base
	base, ok := presets[cfg.Preset]
	if !ok {
		// Unknown preset -> fallback to full silently
		cfg.Preset = "full"
		base = presets["full"]
	}

	// v1.1: merge features override into preset base
	cfg.Features = mergeFeatures(base, cfg.Features)

	// v1.1: validate and normalize priority
	cfg.Priority = validatePriority(cfg.Priority)

	return cfg
}
