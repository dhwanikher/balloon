BIN     := bin/balloon
PKG     := ./cmd/balloon
GOFILES := $(shell find . -name '*.go' -not -path './web/*')

.PHONY: all build test cover vet fmt demo clean

all: fmt vet test build

build: $(BIN)

$(BIN): $(GOFILES)
	go build -o $(BIN) $(PKG)

test:
	go test ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

vet:
	go vet ./...

fmt:
	gofmt -l -w ./cmd ./internal

# Regenerates the picture in the README.
demo: $(BIN)
	./$(BIN) demo -o docs/demo.svg

clean:
	rm -rf bin coverage.out
