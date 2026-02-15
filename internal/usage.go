package internal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// UsageData represents the OAuth API usage quota with 5-hour and 7-day limits.
type UsageData struct {
	RemainingPercent5h float64   `json:"remaining_percent_5h"`
	RemainingPercent7d float64   `json:"remaining_percent_7d"`
	ResetsAt5h         time.Time `json:"resets_at_5h"`
	ResetsAt7d         time.Time `json:"resets_at_7d"`
	FetchedAt          int64     `json:"fetched_at"`
}

type usageAPIResponse struct {
	FiveHour struct {
		Utilization float64 `json:"utilization"`
		ResetsAt    string  `json:"resets_at"`
	} `json:"five_hour"`
	SevenDay struct {
		Utilization float64 `json:"utilization"`
		ResetsAt    string  `json:"resets_at"`
	} `json:"seven_day"`
}

// usageAPIURL is the OAuth usage API endpoint. Package-level var for test substitution.
var usageAPIURL = "https://api.anthropic.com/api/oauth/usage"

// getOAuthTokenFunc returns the OAuth token. Package-level var for test substitution.
var getOAuthTokenFunc = getOAuthToken

const (
	cacheTTL   = UsageCacheTTL
	apiTimeout = UsageAPITimeout
)

// GetUsage fetches OAuth usage data with session-scoped caching.
// Returns nil on any failure — usage quota is optional.
func GetUsage(sessionID string) *UsageData {
	if sessionID == "" {
		return nil
	}

	// Try cache first
	if cached := loadCachedUsage(sessionID); cached != nil {
		if time.Since(time.Unix(cached.FetchedAt, 0)) < cacheTTL {
			return cached
		}
	}

	// Fetch fresh data
	token := getOAuthTokenFunc()
	if token == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", usageAPIURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("User-Agent", "howl/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
	if err != nil {
		return nil
	}

	var apiResp usageAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil
	}

	// Parse reset times
	resetTime5h, _ := time.Parse(time.RFC3339, apiResp.FiveHour.ResetsAt)
	resetTime7d, _ := time.Parse(time.RFC3339, apiResp.SevenDay.ResetsAt)

	// Convert utilization to remaining percentage
	// API returns utilization (used %), we want remaining %
	usage := &UsageData{
		RemainingPercent5h: 100 - apiResp.FiveHour.Utilization,
		RemainingPercent7d: 100 - apiResp.SevenDay.Utilization,
		ResetsAt5h:         resetTime5h,
		ResetsAt7d:         resetTime7d,
		FetchedAt:          time.Now().Unix(),
	}

	// Cache it
	saveCachedUsage(sessionID, usage)
	return usage
}

func getOAuthToken() string {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/security", "find-generic-password",
		"-s", "Claude Code-credentials",
		"-w")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse JSON structure: {"claudeAiOauth":{"accessToken":"..."}}
	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(out, &creds); err != nil {
		return ""
	}
	return creds.ClaudeAiOauth.AccessToken
}

func sanitizeSessionID(sessionID string) string {
	return filepath.Base(sessionID)
}

func cacheDir(sessionID string) string {
	return filepath.Join(os.TempDir(), "howl-"+sanitizeSessionID(sessionID))
}

func cachePath(sessionID string) string {
	return filepath.Join(cacheDir(sessionID), "usage.json")
}

func loadCachedUsage(sessionID string) *UsageData {
	path := cachePath(sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var usage UsageData
	if err := json.Unmarshal(data, &usage); err != nil {
		return nil
	}
	return &usage
}

func saveCachedUsage(sessionID string, usage *UsageData) {
	dir := cacheDir(sessionID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return
	}
	data, err := json.Marshal(usage)
	if err != nil {
		return
	}
	_ = os.WriteFile(cachePath(sessionID), data, 0600)
}
