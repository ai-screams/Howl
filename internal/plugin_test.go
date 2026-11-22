package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A wrong path in hooks.json only costs a debug-log line in Claude Code, so a
// release would lose the agent rows silently. This fails when hooks.json stops
// naming a module, or names a file that is not there.
func TestPluginHooksModuleExists(t *testing.T) {
	t.Parallel()

	dir := filepath.Join("..", "hooks")
	raw, err := os.ReadFile(filepath.Join(dir, "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Modules []string `json:"modules"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Modules) != 1 {
		t.Fatalf("hooks.json names %d modules, want 1", len(manifest.Modules))
	}
	if _, err := os.Stat(filepath.Join(dir, manifest.Modules[0])); err != nil {
		t.Errorf("hooks module is missing: %v", err)
	}
}
