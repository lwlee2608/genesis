GO = $(shell which go 2>/dev/null)

APP             := project-00
VERSION         ?= $(shell cat VERSION)
COMMIT_SHA      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
LDFLAGS         := -ldflags "-X main.AppVersion=$(VERSION)-$(COMMIT_SHA)"

.PHONY: all help build clean run test dep generate

all: clean build

help:
	@echo "Usage: make <target>"
	@echo
	@echo "Targets:"
	@echo "  all         Clean and build the binary"
	@echo "  help        Show this help"
	@echo "  dep         Install build/dev dependencies (sqlc)"
	@echo "  clean       Remove build artifacts and clear the test cache"
	@echo "  build       Build the application binary into bin/"
	@echo "  run         Run the application from source"
	@echo "  test        Run unit tests"
	@echo "  generate    Regenerate sqlc code from SQL queries"

dep:
	$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0
clean:
	$(GO) clean -testcache
	$(RM) -rf bin/*
build:
	$(GO) build -o bin/$(APP) $(LDFLAGS) cmd/$(APP)/*.go
run:
	$(GO) run $(LDFLAGS) cmd/$(APP)/*.go
test:
	$(GO) test ./...
generate:
	sqlc generate
