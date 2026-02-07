package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

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

const (
	usageAPIURL = "https://api.anthropic.com/api/oauth/usage"
	cacheTTL    = UsageCacheTTL
	apiTimeout  = UsageAPITimeout
)

// getUsage fetches OAuth usage data with session-scoped caching.
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
	token := getOAuthToken()
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
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
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
	cmd := exec.Command("security", "find-generic-password",
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

func cacheDir(sessionID string) string {
	return filepath.Join(os.TempDir(), "howl-"+sessionID)
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
	os.MkdirAll(dir, 0755)
	data, err := json.Marshal(usage)
	if err != nil {
		return
	}
	os.WriteFile(cachePath(sessionID), data, 0644)
}

// renderQuota formats 5h/7d remaining percentage for display
// Format: (2h)5h: 55%/41% :7d(3d5h)
func renderQuota(u *UsageData) string {
	// Color based on remaining percentage (lower = more urgent)
	color5 := quotaColor(u.RemainingPercent5h)
	color7 := quotaColor(u.RemainingPercent7d)

	// Calculate time until Reset
	now := time.Now()
	until5h := formatTimeUntil(now, u.ResetsAt5h)
	until7d := formatTimeUntil(now, u.ResetsAt7d)

	return fmt.Sprintf("%s(%s)5h:%s %s%.0f%%%s/%s%.0f%%%s %s:7d(%s)%s",
		dim, until5h, Reset,
		color5, u.RemainingPercent5h, Reset,
		color7, u.RemainingPercent7d, Reset,
		grey, until7d, Reset)
}

func formatTimeUntil(now, target time.Time) string {
	if target.IsZero() {
		return "?"
	}
	diff := target.Sub(now)
	if diff < 0 {
		return "0"
	}

	hours := int(diff.Hours())
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	days := hours / 24
	remainHours := hours % 24
	if remainHours == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd%dh", days, remainHours)
}

func quotaColor(remaining float64) string {
	switch {
	case remaining < QuotaCritical:
		return boldRed
	case remaining < QuotaLow:
		return red
	case remaining < QuotaMedium:
		return orange
	case remaining < QuotaHigh:
		return yellow
	default:
		return green
	}
}
