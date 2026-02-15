package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSanitizeSessionID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal id", "abc-123-def", "abc-123-def"},
		{"path traversal", "../../../etc/passwd", "passwd"},
		{"absolute path", "/absolute/path/id", "id"},
		{"dot", ".", "."},
		{"double dot", "..", ".."},
		{"empty", "", "."},
		{"nested traversal", "foo/../bar", "bar"},
		{"just slash", "/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeSessionID(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeSessionID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestQuotaColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		remaining float64
		want      string
	}{
		{"critical below 10", 5.0, boldRed},
		{"critical at boundary", 9.99, boldRed},
		{"low at 10", 10.0, red},
		{"low below 25", 24.9, red},
		{"medium at 25", 25.0, orange},
		{"medium below 50", 49.9, orange},
		{"high at 50", 50.0, yellow},
		{"high below 75", 74.9, yellow},
		{"healthy at 75", 75.0, green},
		{"healthy at 100", 100.0, green},
		{"zero critical", 0.0, boldRed},
		{"edge case 10.01", 10.01, red},
		{"edge case 25.01", 25.01, orange},
		{"edge case 50.01", 50.01, yellow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := quotaColor(tt.remaining, DefaultThresholds())
			if got != tt.want {
				t.Errorf("quotaColor(%v) = %q, want %q", tt.remaining, got, tt.want)
			}
		})
	}
}

func TestFormatTimeUntil(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 2, 7, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		now    time.Time
		target time.Time
		want   string
	}{
		{
			name:   "zero target",
			now:    baseTime,
			target: time.Time{},
			want:   "?",
		},
		{
			name:   "expired past",
			now:    baseTime,
			target: baseTime.Add(-1 * time.Hour),
			want:   "0d00h",
		},
		{
			name:   "expired exact",
			now:    baseTime,
			target: baseTime,
			want:   "0d00h",
		},
		{
			name:   "3 hours future",
			now:    baseTime,
			target: baseTime.Add(3 * time.Hour),
			want:   "0d03h",
		},
		{
			name:   "25 hours future",
			now:    baseTime,
			target: baseTime.Add(25 * time.Hour),
			want:   "1d01h",
		},
		{
			name:   "48 hours future",
			now:    baseTime,
			target: baseTime.Add(48 * time.Hour),
			want:   "2d00h",
		},
		{
			name:   "1 minute future",
			now:    baseTime,
			target: baseTime.Add(1 * time.Minute),
			want:   "0d00h",
		},
		{
			name:   "23 hours 59 minutes",
			now:    baseTime,
			target: baseTime.Add(23*time.Hour + 59*time.Minute),
			want:   "0d23h",
		},
		{
			name:   "7 days exact",
			now:    baseTime,
			target: baseTime.Add(7 * 24 * time.Hour),
			want:   "7d00h",
		},
		{
			name:   "3 days 12 hours",
			now:    baseTime,
			target: baseTime.Add(3*24*time.Hour + 12*time.Hour),
			want:   "3d12h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatTimeUntil(tt.now, tt.target)
			if got != tt.want {
				t.Errorf("formatTimeUntil() = %q, want %q", got, tt.want)
			}
		})
	}
}

// saveAndRestore saves package-level vars and restores them via t.Cleanup.
func saveAndRestore(t *testing.T) {
	t.Helper()
	origURL := usageAPIURL
	origTokenFunc := getOAuthTokenFunc
	t.Cleanup(func() {
		usageAPIURL = origURL
		getOAuthTokenFunc = origTokenFunc
	})
}

// Group 1: Cache Functions

func TestCacheDir(t *testing.T) {
	sessionID := "test-session-123"
	got := cacheDir(sessionID)
	if !strings.Contains(got, "howl-") {
		t.Errorf("cacheDir(%q) = %q, expected to contain 'howl-' prefix", sessionID, got)
	}
	if !strings.Contains(got, "test-session-123") {
		t.Errorf("cacheDir(%q) = %q, expected to contain session ID", sessionID, got)
	}
}

func TestCachePath(t *testing.T) {
	sessionID := "test-session-456"
	got := cachePath(sessionID)
	if !strings.HasSuffix(got, "usage.json") {
		t.Errorf("cachePath(%q) = %q, expected to end with 'usage.json'", sessionID, got)
	}
	if !strings.Contains(got, "howl-") {
		t.Errorf("cachePath(%q) = %q, expected to contain 'howl-' prefix", sessionID, got)
	}
}

func TestLoadCachedUsage_NoFile(t *testing.T) {
	sessionID := "nonexistent-session-" + t.Name()
	got := loadCachedUsage(sessionID)
	if got != nil {
		t.Errorf("loadCachedUsage(%q) = %+v, want nil for nonexistent file", sessionID, got)
	}
}

func TestLoadCachedUsage_BadJSON(t *testing.T) {
	sessionID := "bad-json-session-" + t.Name()
	dir := cacheDir(sessionID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	path := cachePath(sessionID)
	if err := os.WriteFile(path, []byte("not valid json {{{"), 0600); err != nil {
		t.Fatalf("Failed to write bad JSON: %v", err)
	}

	got := loadCachedUsage(sessionID)
	if got != nil {
		t.Errorf("loadCachedUsage(%q) = %+v, want nil for invalid JSON", sessionID, got)
	}
}

func TestCacheRoundTrip(t *testing.T) {
	sessionID := "roundtrip-session-" + t.Name()
	dir := cacheDir(sessionID)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	baseTime := time.Date(2026, 2, 10, 10, 30, 0, 0, time.UTC)
	original := &UsageData{
		RemainingPercent5h: 42.5,
		RemainingPercent7d: 67.8,
		ResetsAt5h:         baseTime.Add(2 * time.Hour),
		ResetsAt7d:         baseTime.Add(4 * 24 * time.Hour),
		FetchedAt:          baseTime.Unix(),
	}

	saveCachedUsage(sessionID, original)
	loaded := loadCachedUsage(sessionID)

	if loaded == nil {
		t.Fatal("loadCachedUsage() returned nil after save")
	}

	if loaded.RemainingPercent5h != original.RemainingPercent5h {
		t.Errorf("RemainingPercent5h mismatch: got %v, want %v", loaded.RemainingPercent5h, original.RemainingPercent5h)
	}
	if loaded.RemainingPercent7d != original.RemainingPercent7d {
		t.Errorf("RemainingPercent7d mismatch: got %v, want %v", loaded.RemainingPercent7d, original.RemainingPercent7d)
	}
	if !loaded.ResetsAt5h.Equal(original.ResetsAt5h) {
		t.Errorf("ResetsAt5h mismatch: got %v, want %v", loaded.ResetsAt5h, original.ResetsAt5h)
	}
	if !loaded.ResetsAt7d.Equal(original.ResetsAt7d) {
		t.Errorf("ResetsAt7d mismatch: got %v, want %v", loaded.ResetsAt7d, original.ResetsAt7d)
	}
	if loaded.FetchedAt != original.FetchedAt {
		t.Errorf("FetchedAt mismatch: got %v, want %v", loaded.FetchedAt, original.FetchedAt)
	}
}

// Group 2: GetUsage Edge Cases

func TestGetUsage_EmptySession(t *testing.T) {
	got := GetUsage("")
	if got != nil {
		t.Errorf("GetUsage(\"\") = %+v, want nil for empty session", got)
	}
}

func TestGetUsage_CacheHit(t *testing.T) {
	sessionID := "cache-hit-session-" + t.Name()
	dir := cacheDir(sessionID)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	// Pre-populate fresh cache
	baseTime := time.Now()
	cached := &UsageData{
		RemainingPercent5h: 55.0,
		RemainingPercent7d: 75.0,
		ResetsAt5h:         baseTime.Add(1 * time.Hour),
		ResetsAt7d:         baseTime.Add(3 * 24 * time.Hour),
		FetchedAt:          baseTime.Unix(), // fresh cache
	}
	saveCachedUsage(sessionID, cached)

	// Should return cached data without HTTP
	got := GetUsage(sessionID)
	if got == nil {
		t.Fatal("GetUsage() returned nil, expected cached data")
	}

	if got.RemainingPercent5h != 55.0 {
		t.Errorf("GetUsage() RemainingPercent5h = %v, want 55.0 from cache", got.RemainingPercent5h)
	}
	if got.RemainingPercent7d != 75.0 {
		t.Errorf("GetUsage() RemainingPercent7d = %v, want 75.0 from cache", got.RemainingPercent7d)
	}
}

func TestGetUsage_CacheExpired(t *testing.T) {
	saveAndRestore(t)

	sessionID := "cache-expired-session-" + t.Name()
	dir := cacheDir(sessionID)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	// Pre-populate expired cache (61 seconds ago, TTL is 60s)
	expiredTime := time.Now().Add(-61 * time.Second)
	expired := &UsageData{
		RemainingPercent5h: 55.0,
		RemainingPercent7d: 75.0,
		ResetsAt5h:         time.Now().Add(1 * time.Hour),
		ResetsAt7d:         time.Now().Add(3 * 24 * time.Hour),
		FetchedAt:          expiredTime.Unix(),
	}
	saveCachedUsage(sessionID, expired)

	// Mock token func to return empty (no OAuth token available)
	getOAuthTokenFunc = func() string { return "" }

	got := GetUsage(sessionID)
	if got != nil {
		t.Errorf("GetUsage() = %+v, want nil when cache expired and no token", got)
	}
}

// Group 3: HTTP Path

func TestGetUsage_HTTPSuccess(t *testing.T) {
	saveAndRestore(t)

	sessionID := "http-success-session-" + t.Name()
	dir := cacheDir(sessionID)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	// Mock server
	var receivedAuthHeader string
	var receivedBetaHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		receivedBetaHeader = r.Header.Get("anthropic-beta")

		response := usageAPIResponse{
			FiveHour: struct {
				Utilization float64 `json:"utilization"`
				ResetsAt    string  `json:"resets_at"`
			}{
				Utilization: 30.0, // API returns used percentage
				ResetsAt:    "2026-02-10T15:00:00Z",
			},
			SevenDay: struct {
				Utilization float64 `json:"utilization"`
				ResetsAt    string  `json:"resets_at"`
			}{
				Utilization: 20.0,
				ResetsAt:    "2026-02-15T12:00:00Z",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	usageAPIURL = server.URL
	getOAuthTokenFunc = func() string { return "test-token" }

	got := GetUsage(sessionID)

	// Verify request headers
	if receivedAuthHeader != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want %q", receivedAuthHeader, "Bearer test-token")
	}
	if receivedBetaHeader != "oauth-2025-04-20" {
		t.Errorf("anthropic-beta header = %q, want %q", receivedBetaHeader, "oauth-2025-04-20")
	}

	// Verify utilization→remaining conversion
	if got == nil {
		t.Fatal("GetUsage() returned nil, expected valid data")
	}
	if got.RemainingPercent5h != 70.0 { // 100 - 30
		t.Errorf("RemainingPercent5h = %v, want 70.0 (100 - 30)", got.RemainingPercent5h)
	}
	if got.RemainingPercent7d != 80.0 { // 100 - 20
		t.Errorf("RemainingPercent7d = %v, want 80.0 (100 - 20)", got.RemainingPercent7d)
	}

	// Verify caching
	cached := loadCachedUsage(sessionID)
	if cached == nil {
		t.Error("Result was not cached after fetch")
	} else if cached.RemainingPercent5h != 70.0 {
		t.Errorf("Cached RemainingPercent5h = %v, want 70.0", cached.RemainingPercent5h)
	}
}

func TestGetUsage_NoToken(t *testing.T) {
	saveAndRestore(t)

	sessionID := "no-token-session-" + t.Name()
	getOAuthTokenFunc = func() string { return "" }

	got := GetUsage(sessionID)
	if got != nil {
		t.Errorf("GetUsage() = %+v, want nil when no token available", got)
	}
}

func TestGetUsage_HTTPError(t *testing.T) {
	saveAndRestore(t)

	sessionID := "http-error-session-" + t.Name()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	usageAPIURL = server.URL
	getOAuthTokenFunc = func() string { return "test-token" }

	got := GetUsage(sessionID)
	if got != nil {
		t.Errorf("GetUsage() = %+v, want nil on HTTP 500", got)
	}
}

func TestGetUsage_BadResponseJSON(t *testing.T) {
	saveAndRestore(t)

	sessionID := "bad-response-json-session-" + t.Name()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json {{{"))
	}))
	defer server.Close()

	usageAPIURL = server.URL
	getOAuthTokenFunc = func() string { return "test-token" }

	got := GetUsage(sessionID)
	if got != nil {
		t.Errorf("GetUsage() = %+v, want nil on invalid JSON response", got)
	}
}

func TestGetUsage_HTTP403(t *testing.T) {
	saveAndRestore(t)

	sessionID := "http-403-session-" + t.Name()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	usageAPIURL = server.URL
	getOAuthTokenFunc = func() string { return "test-token" }

	got := GetUsage(sessionID)
	if got != nil {
		t.Errorf("GetUsage() = %+v, want nil on HTTP 403", got)
	}
}

func TestGetUsage_PartialAPIResponse(t *testing.T) {
	saveAndRestore(t)

	sessionID := "partial-response-session-" + t.Name()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Missing resets_at fields
		response := map[string]interface{}{
			"five_hour": map[string]interface{}{
				"utilization": 25.0,
			},
			"seven_day": map[string]interface{}{
				"utilization": 15.0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	usageAPIURL = server.URL
	getOAuthTokenFunc = func() string { return "test-token" }

	got := GetUsage(sessionID)
	if got == nil {
		t.Fatal("GetUsage() returned nil, expected partial data")
	}

	// Verify utilization conversion works
	if got.RemainingPercent5h != 75.0 { // 100 - 25
		t.Errorf("RemainingPercent5h = %v, want 75.0", got.RemainingPercent5h)
	}

	// Verify zero times for missing resets_at
	if !got.ResetsAt5h.IsZero() {
		t.Errorf("ResetsAt5h = %v, want zero time for missing field", got.ResetsAt5h)
	}
	if !got.ResetsAt7d.IsZero() {
		t.Errorf("ResetsAt7d = %v, want zero time for missing field", got.ResetsAt7d)
	}
}
