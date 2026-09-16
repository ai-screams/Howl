package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadUpdateNoticeAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string // "" means: do not create the file at all
		current string
		want    string // "" means: expect no notice
	}{
		{"no file at all", "", "1.10.2", ""},
		{"newer version", "1.11.0", "1.10.2", "1.11.0"},
		{"v prefix is normalised", "v1.11.0\n", "1.10.2", "1.11.0"},
		{"v prefix on the running version too", "1.11.0", "v1.10.2", "1.11.0"},
		{"stale file naming the running version", "1.10.2", "1.10.2", ""},
		{"stale file with prefixes on both sides", "v1.10.2\n", "v1.10.2", ""},
		{"blank file", "\n", "1.10.2", ""},
		{"whitespace only", "   \t  \n", "1.10.2", ""},
		{"only the first line is read", "1.11.0\nignored\n", "1.10.2", "1.11.0"},
		{"control characters are stripped", "1.11.0\033[2J", "1.10.2", "1.11.0"},
		{"escape-only content", "\033\x00", "1.10.2", ""},
		{"not a version at all", "howl", "1.10.2", ""},
		{"an error message", "curl: (6) could not resolve host", "1.10.2", ""},
		{"leading junk before a version", "error 1.11.0", "1.10.2", ""},
		{"prerelease suffix is kept", "1.11.0-rc.1", "1.10.2", "1.11.0-rc.1"},
		{"build metadata is kept", "1.11.0+build.5", "1.10.2", "1.11.0+build.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), updateNoticeFile)
			if tt.content != "" {
				if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
					t.Fatalf("writing fixture: %v", err)
				}
			}

			got := readUpdateNoticeAt(path, tt.current)
			if tt.want == "" {
				if got != nil {
					t.Errorf("readUpdateNoticeAt() = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("readUpdateNoticeAt() = nil, want version %q", tt.want)
			}
			if got.Version != tt.want {
				t.Errorf("version = %q, want %q", got.Version, tt.want)
			}
		})
	}
}

// The file is written by a shell script, so a runaway writer must not turn into
// a status line full of garbage.
func TestReadUpdateNoticeAt_CapsWhatItReads(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), updateNoticeFile)
	huge := strings.Repeat("9", 10_000)
	if err := os.WriteFile(path, []byte(huge), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	got := readUpdateNoticeAt(path, "1.10.2")
	if got == nil {
		t.Fatal("expected a notice")
	}
	if len(got.Version) > maxUpdateNoticeBytes {
		t.Errorf("version is %d bytes, over the %d cap", len(got.Version), maxUpdateNoticeBytes)
	}
}

func TestReadUpdateNoticeAt_UnreadablePathIsNotAnError(t *testing.T) {
	t.Parallel()

	// A directory where a file is expected: Open succeeds on some systems and
	// the read fails, so this exercises the nil-return-on-failure contract.
	dir := filepath.Join(t.TempDir(), updateNoticeFile)
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatalf("creating fixture: %v", err)
	}
	if got := readUpdateNoticeAt(dir, "1.10.2"); got != nil {
		t.Errorf("readUpdateNoticeAt(directory) = %+v, want nil", got)
	}
}

func TestRenderUpdateNotice(t *testing.T) {
	t.Parallel()

	if got := renderUpdateNotice(nil); got != "" {
		t.Errorf("renderUpdateNotice(nil) = %q, want empty", got)
	}
	if got := stripANSI(renderUpdateNotice(&UpdateNotice{Version: "1.11.0"})); got != "↑1.11.0" {
		t.Errorf("renderUpdateNotice() = %q, want %q", got, "↑1.11.0")
	}
	// The version came off disk; it must not carry escapes into the terminal.
	got := renderUpdateNotice(&UpdateNotice{Version: "1.11.0\033[2J"})
	if err := onlyHowlsOwnEscapes(got); err != "" {
		t.Errorf("%s: %q", err, got)
	}
}

// The badge must appear without configuration — a notice nobody enabled tells
// nobody anything — and must still be suppressible.
func TestRender_UpdateNoticeIsOnByDefaultAndCanBeHidden(t *testing.T) {
	t.Parallel()

	data := &StdinData{
		Model:         Model{DisplayName: "Opus"},
		Workspace:     Workspace{CurrentDir: "/tmp/p"},
		ContextWindow: ContextWindow{UsedPercentage: floatPtr(30), ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 60000},
	}
	notice := &UpdateNotice{Version: "1.11.0"}

	t.Run("shown by default", func(t *testing.T) {
		t.Parallel()
		out := stripANSI(strings.Join(Render(RenderContext{
			Data: data, Metrics: ComputeMetrics(data), Update: notice, Config: PresetConfig("full"),
		}), "\n"))
		if !strings.Contains(out, "↑1.11.0") {
			t.Errorf("update badge missing from default output:\n%s", out)
		}
	})

	t.Run("absent when there is no update", func(t *testing.T) {
		t.Parallel()
		out := stripANSI(strings.Join(Render(RenderContext{
			Data: data, Metrics: ComputeMetrics(data), Config: PresetConfig("full"),
		}), "\n"))
		if strings.Contains(out, "↑") {
			t.Errorf("badge rendered with no notice:\n%s", out)
		}
	})

	t.Run("hidden when opted out", func(t *testing.T) {
		t.Parallel()
		cfg := PresetConfig("full")
		cfg.Features.HideUpdateNotice = true
		out := stripANSI(strings.Join(Render(RenderContext{
			Data: data, Metrics: ComputeMetrics(data), Update: notice, Config: cfg,
		}), "\n"))
		if strings.Contains(out, "1.11.0") {
			t.Errorf("badge rendered although hide_update_notice was set:\n%s", out)
		}
	})

	t.Run("left out of danger mode", func(t *testing.T) {
		t.Parallel()
		danger := *data
		danger.ContextWindow.UsedPercentage = floatPtr(95)
		out := stripANSI(strings.Join(Render(RenderContext{
			Data: &danger, Metrics: ComputeMetrics(&danger), Update: notice, Config: PresetConfig("full"),
		}), "\n"))
		if strings.Contains(out, "1.11.0") {
			t.Errorf("badge rendered in danger mode:\n%s", out)
		}
	})
}

// The opt-out is inverted precisely because mergeFeatures cannot turn a flag
// off; this pins that the inverted flag still survives a merge.
func TestMergeFeatures_HideUpdateNotice(t *testing.T) {
	t.Parallel()

	if got := mergeFeatures(FeatureToggles{}, FeatureToggles{HideUpdateNotice: true}); !got.HideUpdateNotice {
		t.Error("mergeFeatures dropped the hide_update_notice override")
	}
	if got := mergeFeatures(FeatureToggles{HideUpdateNotice: true}, FeatureToggles{}); !got.HideUpdateNotice {
		t.Error("an empty override cleared hide_update_notice")
	}
	if got := mergeFeatures(FeatureToggles{}, FeatureToggles{}); got.HideUpdateNotice {
		t.Error("hide_update_notice defaulted to hidden; the notice should show")
	}
}
