package internal

import "testing"

func TestClassifyModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		model Model
		want  ModelTier
	}{
		{
			name:  "opus via DisplayName",
			model: Model{DisplayName: "Claude Opus 4"},
			want:  TierOpus,
		},
		{
			name:  "sonnet via DisplayName",
			model: Model{DisplayName: "Claude Sonnet 4.5"},
			want:  TierSonnet,
		},
		{
			name:  "haiku via DisplayName",
			model: Model{DisplayName: "Claude Haiku 4.5"},
			want:  TierHaiku,
		},
		{
			name:  "unknown model",
			model: Model{DisplayName: "GPT-4"},
			want:  TierUnknown,
		},
		{
			name:  "empty DisplayName falls back to ID with opus",
			model: Model{ID: "claude-opus-4", DisplayName: ""},
			want:  TierOpus,
		},
		{
			name:  "empty DisplayName falls back to ID with sonnet",
			model: Model{ID: "claude-sonnet-3-5", DisplayName: ""},
			want:  TierSonnet,
		},
		{
			name:  "empty DisplayName falls back to ID with haiku",
			model: Model{ID: "claude-haiku-3", DisplayName: ""},
			want:  TierHaiku,
		},
		{
			name:  "both empty returns TierUnknown",
			model: Model{ID: "", DisplayName: ""},
			want:  TierUnknown,
		},
		{
			name:  "case insensitive OPUS uppercase",
			model: Model{DisplayName: "Claude OPUS 4"},
			want:  TierOpus,
		},
		{
			name:  "case insensitive opus lowercase",
			model: Model{DisplayName: "claude opus 4"},
			want:  TierOpus,
		},
		{
			name:  "bedrock-style ID with sonnet",
			model: Model{ID: "anthropic.claude-sonnet-3-5-v2:0"},
			want:  TierSonnet,
		},
		{
			name:  "mixed case in ID",
			model: Model{ID: "Claude-HAIKU-3"},
			want:  TierHaiku,
		},
		{
			name:  "opus takes precedence over sonnet",
			model: Model{DisplayName: "Claude Opus Sonnet 4"},
			want:  TierOpus,
		},
		{
			name:  "sonnet takes precedence over haiku",
			model: Model{DisplayName: "Claude Sonnet Haiku 4"},
			want:  TierSonnet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyModel(tt.model)
			if got != tt.want {
				t.Errorf("classifyModel(%v) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
