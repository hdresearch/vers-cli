# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Manifest information - can be overridden
NAME ?= vers-cli
DESCRIPTION ?= A CLI tool for version management
AUTHOR ?= Tynan Daly
REPOSITORY ?= https://github.com/hdresearch/vers-cli
LICENSE ?= MIT

# Build flags
LDFLAGS = -s -w \
	-X 'github.com/hdresearch/vers-cli/cmd.Version=$(VERSION)' \
	-X 'github.com/hdresearch/vers-cli/cmd.GitCommit=$(GIT_COMMIT)' \
	-X 'github.com/hdresearch/vers-cli/cmd.BuildDate=$(BUILD_DATE)' \
	-X 'github.com/hdresearch/vers-cli/cmd.Name=$(NAME)' \
	-X 'github.com/hdresearch/vers-cli/cmd.Description=$(DESCRIPTION)' \
	-X 'github.com/hdresearch/vers-cli/cmd.Author=$(AUTHOR)' \
	-X 'github.com/hdresearch/vers-cli/cmd.Repository=$(REPOSITORY)' \
	-X 'github.com/hdresearch/vers-cli/cmd.License=$(LICENSE)'

# Build the vers binary (static, no libc dependency)
.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS) -extldflags '-static'" -o bin/vers ./cmd/vers

# Build static Linux binaries for distribution
.PHONY: build-linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS) -extldflags '-static'" -o bin/vers-linux-amd64 ./cmd/vers
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS) -extldflags '-static'" -o bin/vers-linux-arm64 ./cmd/vers

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf bin/

# Ensure bin directory exists
bin:
	mkdir -p bin

# Install CLI binary to /usr/local/bin
install:
	cp bin/vers /usr/local/bin

# Build and install binary
build-and-install: build install

.PHONY: vet
# Run go vet on all packages
vet:
	go vet ./...

.PHONY: test test-unit test-integration
# Run all unit tests (excludes ./test integration package). Includes MCP-tagged tests.
test test-unit:
	go test -tags mcp $$(go list ./... | grep -v '/test$$') -v

# Run integration tests under ./test (requires VERS_URL, VERS_API_KEY, etc.)
test-integration:
	@if [ -z "$$VERS_URL" ] || [ -z "$$VERS_API_KEY" ]; then \
		echo "[!] Missing VERS_URL or VERS_API_KEY env for integration tests"; \
		echo "    export VERS_URL=https://..."; \
		echo "    export VERS_API_KEY=..."; \
		exit 1; \
	fi
	cd test && go test -v $(ARGS)

