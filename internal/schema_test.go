package internal

import (
	"encoding/json"
	"os"
	"testing"
)

// Every other test builds StdinData in Go, so a wrong `json:"…"` tag would
// decode to a zero value, render nothing, and leave the suite green. This
// decodes a real Claude Code 2.1.272 payload — captured from stdin, then
// scrubbed of session-local identifiers — so the tags themselves are covered.
//
// It also pins the null-vs-zero behaviour that motivated the pointer types:
// `last_miss_at` and `last_miss_cause` arrive as JSON null and must stay nil
// rather than collapsing into a zero that reads as "no misses, at time 0".
func TestDecodeRealPayload(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("testdata/statusline_payload.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	var d StdinData
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("decoding the real payload failed: %v", err)
	}

	t.Run("prompt_cache", func(t *testing.T) {
		pc := d.PromptCache
		if pc == nil {
			t.Fatal("prompt_cache decoded as nil — check the tag on StdinData.PromptCache")
		}
		if !pc.Warm {
			t.Error("warm decoded false, fixture says true")
		}
		if !pc.CachingObserved {
			t.Error("caching_observed decoded false, fixture says true")
		}
		if pc.TTL != "1h" {
			t.Errorf("ttl = %q, want \"1h\"", pc.TTL)
		}
		if pc.ExpiresAt == nil || *pc.ExpiresAt != 1789455504 {
			t.Errorf("expires_at = %v, want 1789455504", pc.ExpiresAt)
		}
		if pc.HitRatio == nil {
			t.Fatal("hit_ratio decoded as nil")
		}
		if *pc.HitRatio < 0.97 || *pc.HitRatio > 0.98 {
			t.Errorf("hit_ratio = %v, want ~0.9716 (a fraction, not a percentage)", *pc.HitRatio)
		}
		if pc.Requests != 40 {
			t.Errorf("requests = %d, want 40", pc.Requests)
		}
		if pc.CacheWriteTokens != 105598 {
			t.Errorf("cache_write_tokens = %d, want 105598", pc.CacheWriteTokens)
		}
		if pc.RecacheTokensIfCold == nil || *pc.RecacheTokensIfCold != 138106 {
			t.Errorf("recache_tokens_if_cold = %v, want 138106", pc.RecacheTokensIfCold)
		}
		// Documented as null until the session's first miss. A non-pointer type
		// would turn "never missed" into "missed at the epoch".
		if pc.LastMissAt != nil {
			t.Errorf("last_miss_at = %v, want nil for a session with no misses", *pc.LastMissAt)
		}
		if pc.LastMissCause != nil {
			t.Errorf("last_miss_cause = %+v, want nil for a session with no misses", *pc.LastMissCause)
		}
		if len(pc.MissCauses) != 0 {
			t.Errorf("miss_causes = %v, want empty", pc.MissCauses)
		}
	})

	t.Run("fast_mode", func(t *testing.T) {
		if d.FastMode {
			t.Error("fast_mode decoded true, fixture says false")
		}
	})

	t.Run("pr", func(t *testing.T) {
		if d.PR == nil {
			t.Fatal("pr decoded as nil")
		}
		if d.PR.Number != 23 {
			t.Errorf("pr.number = %d, want 23", d.PR.Number)
		}
		if d.PR.Kind != "mr" {
			t.Errorf("pr.kind = %q, want \"mr\" — without it a merge request renders as PR#", d.PR.Kind)
		}
		if d.PR.URL == "" {
			t.Error("pr.url decoded empty, so the badge would carry no hyperlink")
		}
		// The whole point of the field: a GitLab MR must not render as PR#.
		if got := stripANSI(renderPR(d.PR)); got != "MR#23 approved" {
			t.Errorf("renderPR() = %q, want \"MR#23 approved\"", got)
		}
	})

	t.Run("context window and exact token count", func(t *testing.T) {
		cw := d.ContextWindow
		if cw.TotalInputTokens != 138106 {
			t.Errorf("total_input_tokens = %d, want 138106", cw.TotalInputTokens)
		}
		if cw.UsedPercentage == nil || int(*cw.UsedPercentage) != 14 {
			t.Errorf("used_percentage = %v, want 14", cw.UsedPercentage)
		}
		// The rounded percentage would derive 140,000 from a 1M window.
		if got := usedContextTokens(cw, 14); got != 138106 {
			t.Errorf("usedContextTokens() = %d, want the exact 138106", got)
		}
	})

	t.Run("rate limits", func(t *testing.T) {
		if d.RateLimits == nil || d.RateLimits.FiveHour == nil || d.RateLimits.SevenDay == nil {
			t.Fatalf("rate_limits decoded incompletely: %+v", d.RateLimits)
		}
		if d.RateLimits.FiveHour.UsedPercentage != 33 {
			t.Errorf("five_hour.used_percentage = %v, want 33", d.RateLimits.FiveHour.UsedPercentage)
		}
	})

	t.Run("workspace repo", func(t *testing.T) {
		if d.Workspace.Repo == nil || d.Workspace.Repo.Name != "howl" {
			t.Errorf("workspace.repo = %+v, want name \"howl\"", d.Workspace.Repo)
		}
	})
}
