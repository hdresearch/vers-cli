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
