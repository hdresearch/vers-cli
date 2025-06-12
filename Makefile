# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Manifest information - can be overridden
NAME ?= vers-cli
DESCRIPTION ?= A CLI tool for version management
AUTHOR ?= Tynan Daly
REPOSITORY ?= https://github.com/tynandaly/vers-cli
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

# Build the vers binary
.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o bin/vers ./cmd/vers

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
