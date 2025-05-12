# Build the vers binary
.PHONY: build
build:
	go build -o bin/vers ./cmd/vers

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