package internal

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// The product page (site/index.html) carries a copy of the preset toggles and
// default thresholds so its HUD demo can lay out lines the way render.go does.
// These tests fail when that copy drifts from config.go, when the page picks
// up a request to another host, or when it links to a file that is not there.

const sitePath = "../site/index.html"

func readSite(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(sitePath)
	if err != nil {
		t.Fatalf("read %s: %v", sitePath, err)
	}
	return string(b)
}

var siteConfigRe = regexp.MustCompile(`(?s)<script type="application/json" id="howl-config">(.*?)</script>`)

// What fails this: flip any toggle or change any number in the page's
// howl-config block, add or drop a preset, or misspell a key.
func TestSiteConfigMatchesCode(t *testing.T) {
	m := siteConfigRe.FindStringSubmatch(readSite(t))
	if m == nil {
		t.Fatal(`site/index.html has no <script type="application/json" id="howl-config"> block`)
	}
	var got struct {
		Presets    map[string]FeatureToggles `json:"presets"`
		Thresholds Thresholds                `json:"thresholds"`
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(m[1])))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("howl-config: %v", err)
	}
	if !reflect.DeepEqual(got.Presets, presets) {
		t.Errorf("site presets differ from config.go presets\n site: %+v\n code: %+v", got.Presets, presets)
	}
	if got.Thresholds != DefaultThresholds() {
		t.Errorf("site thresholds differ from DefaultThresholds()\n site: %+v\n code: %+v", got.Thresholds, DefaultThresholds())
	}
}

var (
	linkTagRe = regexp.MustCompile(`<link\b[^>]*>`)
	relRe     = regexp.MustCompile(`\brel="([^"]*)"`)
	hrefRe    = regexp.MustCompile(`\bhref="([^"]*)"`)
	remoteRes = []*regexp.Regexp{
		regexp.MustCompile(`<script\b[^>]*\bsrc="https?://`),
		regexp.MustCompile(`<img\b[^>]*\bsrc="https?://`),
		regexp.MustCompile(`url\(\s*["']?https?://`),
		regexp.MustCompile(`<(?:iframe|frame|embed|object|source|video|audio|track)\b[^>]*\b(?:src|data)="https?://`),
		regexp.MustCompile(`@import\s+(?:url\()?["']?https?://`),
	}
	// What fails this: add fetch('https://…'), new XMLHttpRequest(), new WebSocket(…),
	// new EventSource(…), navigator.sendBeacon(…) or a dynamic import() to the page's script.
	networkCallRe = regexp.MustCompile(`\bfetch\s*\(|XMLHttpRequest|WebSocket\s*\(|EventSource\s*\(|sendBeacon\s*\(|\bimport\s*\(`)
)

// What fails this: put the Google Fonts <link rel="stylesheet"> back, load a
// script, image, frame or media file from any host, @import a remote
// stylesheet, or add a fetch/XMLHttpRequest/WebSocket/EventSource/sendBeacon/
// import() call to the script.
func TestSiteHasNoExternalRequests(t *testing.T) {
	html := readSite(t)
	for _, tag := range linkTagRe.FindAllString(html, -1) {
		rel, href := relRe.FindStringSubmatch(tag), hrefRe.FindStringSubmatch(tag)
		if rel == nil || href == nil || rel[1] == "canonical" {
			continue
		}
		if strings.HasPrefix(href[1], "http://") || strings.HasPrefix(href[1], "https://") {
			t.Errorf("external %s: %s", rel[1], tag)
		}
	}
	for _, re := range remoteRes {
		if loc := re.FindStringIndex(html); loc != nil {
			end := min(loc[1]+60, len(html))
			t.Errorf("external request: %q", html[loc[0]:end])
		}
	}
	if loc := networkCallRe.FindStringIndex(html); loc != nil {
		end := min(loc[1]+60, len(html))
		t.Errorf("network call in site/index.html: %q", html[loc[0]:end])
	}
}

var localRefRe = regexp.MustCompile(`(?:href|src)="([^"#:]+)"|url\(\s*["']?([^"')#:]+)["']?\s*\)`)

// What fails this: rename a font or icon file without updating the page, or
// point a link at a path that does not exist under site/.
func TestSiteLocalLinksResolve(t *testing.T) {
	html := readSite(t)
	seen := map[string]bool{}
	for _, m := range localRefRe.FindAllStringSubmatch(html, -1) {
		p := m[1]
		if p == "" {
			p = m[2]
		}
		p = strings.SplitN(p, "?", 2)[0]
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		if _, err := os.Stat(filepath.Join(filepath.Dir(sitePath), p)); err != nil {
			t.Errorf("site/index.html links to %q: %v", p, err)
		}
	}
	if len(seen) < 5 {
		t.Fatalf("found only %d local links; expected fonts and assets", len(seen))
	}
}

var hudPreRe = regexp.MustCompile(`(?s)<pre class="hud" id="hud-lines">(.*?)</pre>`)

// What fails this: empty the <pre id="hud-lines"> and let JavaScript fill it.
func TestSiteHUDHasStaticContent(t *testing.T) {
	m := hudPreRe.FindStringSubmatch(readSite(t))
	if m == nil {
		t.Fatal(`site/index.html has no <pre class="hud" id="hud-lines">`)
	}
	if !strings.Contains(m[1], "[Opus 4.6]") || strings.Count(m[1], "\n") < 3 {
		t.Fatalf("hud-lines must carry a pre-rendered four-line statusline; got %q", m[1])
	}
}

// What fails this: drop the canonical link, make og:image relative, write
// the site path in lowercase, or lose the lang attribute.
func TestSiteHead(t *testing.T) {
	html := readSite(t)
	for _, want := range []string{
		`<html lang="en">`,
		`<link rel="canonical" href="https://ai-scream.ai/Howl/">`,
		`<meta property="og:image" content="https://ai-scream.ai/Howl/assets/og.png">`,
		`<meta name="viewport" content="width=device-width, initial-scale=1">`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("site/index.html lacks %s", want)
		}
	}
	if strings.Contains(strings.ToLower(html), "ai-scream.ai/howl/") && !strings.Contains(html, "ai-scream.ai/Howl/") {
		t.Error("site path must be spelled ai-scream.ai/Howl/ (Pages paths are case-sensitive)")
	}
	if strings.Contains(html, "ai-scream.ai/howl/") {
		t.Error("lowercase ai-scream.ai/howl/ found; the org 404 redirect is not in place yet")
	}
}
