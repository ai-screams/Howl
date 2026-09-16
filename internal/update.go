package internal

import (
	"os"
	"path/filepath"
	"strings"
)

// updateNoticeFile is where the plugin's SessionStart hook records a newer
// released version. Howl only ever reads it: checking GitHub from the render
// path would put a network call in a command that runs every few seconds and
// is expected to finish in about ten milliseconds.
const updateNoticeFile = ".update-available"

// maxUpdateNoticeBytes caps what is read from that file. A version string is a
// handful of bytes; anything larger is not one.
const maxUpdateNoticeBytes = 64

// UpdateNotice reports a newer released version of Howl. nil means there is
// nothing to say — no check has run, the check found nothing, or the file
// names the version already running.
type UpdateNotice struct {
	Version string
}

// ReadUpdateNotice reads the notice the session-start hook left behind.
//
// current is the running binary's version; a notice naming it is stale (the
// update already happened) and is ignored, so a missed cleanup cannot pin a
// permanent badge to the status line.
func ReadUpdateNotice(current string) *UpdateNotice {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return readUpdateNoticeAt(filepath.Join(home, ".claude", "hud", updateNoticeFile), current)
}

func readUpdateNoticeAt(path, current string) *UpdateNotice {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, maxUpdateNoticeBytes)
	n, _ := f.Read(buf)
	if n <= 0 {
		return nil
	}

	// The file is written by a shell script, so treat it as untrusted text.
	// A version is a constrained token, so rather than only stripping control
	// characters — which would leave the inert remains of an escape sequence
	// such as "[2J" in the string — keep the leading run of characters a
	// version can actually contain and drop the rest.
	line, _, _ := strings.Cut(string(buf[:n]), "\n")
	version := versionPrefix(strings.TrimPrefix(strings.TrimSpace(line), "v"))
	// A version starts with a number. Requiring that rejects whatever else a
	// broken writer might leave behind — an error message, a tool's name — in
	// the one place where a bad value would be shown on every render.
	if version == "" || version[0] < '0' || version[0] > '9' {
		return nil
	}
	if version == strings.TrimPrefix(strings.TrimSpace(current), "v") {
		return nil // already running it
	}
	return &UpdateNotice{Version: version}
}

// versionPrefix returns the leading run of characters that can appear in a
// semantic version, so trailing junk cannot reach the status line.
func versionPrefix(s string) string {
	for i, r := range s {
		switch {
		case r >= '0' && r <= '9',
			r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r == '.', r == '-', r == '+':
		default:
			return s[:i]
		}
	}
	return s
}
