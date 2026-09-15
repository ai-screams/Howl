package internal

import (
	"strings"
	"testing"
)

// visibleLen must never panic and never exceed the rune count, whatever
// truncated or malformed escape sequence it is handed.
func FuzzVisibleLen(f *testing.F) {
	seeds := []string{
		"", "plain", "\033", "\033[", "\033[3", "\033[31m", "\033]", "\033]8",
		"\033]8;;", "\033]8;;http://x\a", "\033]8;;http://x\033", "\033]8;;http://x\033\\",
		"\033]8;;\033", "\033\\", "\033x", "\033[38;5;245m", "a\033]8;;u\aB\033]8;;\ab",
		"\033]8;;\033[31m\a", "한글\033[31m텍스트", "\033]8;;http://m.example\aPR\033]8;;\a",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := visibleLen(s) // must not panic
		if got < 0 {
			t.Fatalf("visibleLen(%q) = %d, negative", s, got)
		}
		if n := len([]rune(s)); got > n {
			t.Fatalf("visibleLen(%q) = %d, exceeds rune count %d", s, got, n)
		}
	})
}

// fitParts must terminate and must never return something wider than the
// budget unless it is down to a single part.
func FuzzFitParts(f *testing.F) {
	f.Add("a|b|c", 10)
	f.Add("", 0)
	f.Add("\033[31mx\033[0m|\033]8;;u\ay\033]8;;\a", 5)
	f.Fuzz(func(t *testing.T, joined string, width int) {
		parts := strings.Split(joined, "|")
		got := fitParts(parts, width)
		if len(got) > len(parts) {
			t.Fatalf("fitParts grew the slice: %d -> %d", len(parts), len(got))
		}
		if width > 0 && len(got) > 1 && visibleLen(joinParts(got)) > width {
			t.Fatalf("fitParts(%q, %d) = %q, still over budget", parts, width, got)
		}
	})
}

// RenderSubagentRow truncates a name and a description. Both bounds are rune
// counts, so arbitrary multi-byte input must neither panic nor emit NUL or
// replacement runes.
func FuzzRenderSubagentRow(f *testing.F) {
	f.Add("explore", "find the loader", "running", 80)
	f.Add("가나다라마바사아자차카타파하", "설정 로더를 찾아서 분석해 주세요", "running", 80)
	f.Add("🔍🔍🔍", "", "", 0)
	f.Add("", "", "", -1)
	f.Fuzz(func(t *testing.T, name, desc, status string, width int) {
		got := RenderSubagentRow(SubagentTask{Name: name, Description: desc, Status: status}, width, DefaultThresholds())
		if strings.ContainsRune(got, 0) {
			t.Fatalf("NUL rune in row: %q (name=%q desc=%q)", got, name, desc)
		}
		if width > 0 && got != "" && visibleLen(got) > width {
			t.Fatalf("row %q is %d columns, over the %d budget", got, visibleLen(got), width)
		}
	})
}
