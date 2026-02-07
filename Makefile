BINARY := howl
BUILD_DIR := build
INSTALL_DIR := $(HOME)/.claude/hud

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: build install clean test unit-test release-dry release-check

build:
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/howl

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"
	@echo "Add to ~/.claude/settings.json:"
	@echo '  "statusLine": { "type": "command", "command": "$(INSTALL_DIR)/$(BINARY)" }'

clean:
	rm -rf $(BUILD_DIR) dist

test: build
	@echo '{"session_id":"test-123","model":{"id":"claude-opus-4-6","display_name":"Opus 4.6"},"cost":{"total_cost_usd":0.23,"total_duration_ms":4980000,"total_api_duration_ms":897000,"total_lines_added":156,"total_lines_removed":23},"context_window":{"total_input_tokens":15234,"total_output_tokens":4521,"context_window_size":200000,"used_percentage":42,"current_usage":{"input_tokens":8500,"output_tokens":1200,"cache_creation_input_tokens":5000,"cache_read_input_tokens":12000}},"workspace":{"current_dir":"/Users/hanyul/Works/AiScream/hud","project_dir":"/Users/hanyul/Works/AiScream/hud"},"version":"2.1.33"}' | $(BUILD_DIR)/$(BINARY)

unit-test:
	go test ./... -v

release-dry:
	goreleaser release --snapshot --clean

release-check:
	goreleaser check
