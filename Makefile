# Media Metadata Surgery — Makefile

BINARY=surgery/bin/surgery
VERSION=0.1.2

.PHONY: build clean test fmt

build:
	@echo "Building surgery v$(VERSION)..."
	go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BINARY) ./cli
	@echo "Binary: $(BINARY)"
	@ls -lh $(BINARY)

build-all:
	@echo "Building for all platforms..."
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64  go build -ldflags="-s -w" -o dist/surgery-linux-amd64   ./cli
	GOOS=linux   GOARCH=arm64  go build -ldflags="-s -w" -o dist/surgery-linux-arm64   ./cli
	GOOS=darwin  GOARCH=amd64  go build -ldflags="-s -w" -o dist/surgery-darwin-amd64  ./cli
	GOOS=darwin  GOARCH=arm64  go build -ldflags="-s -w" -o dist/surgery-darwin-arm64  ./cli
	GOOS=windows GOARCH=amd64  go build -ldflags="-s -w" -o dist/surgery-windows.exe   ./cli
	@ls -lh dist/

clean:
	rm -f $(BINARY)
	rm -rf dist/

fmt:
	gofmt -w ./cli ./core

test:
	go vet ./...
	@echo "No test suite yet — coming in v0.1.3"
