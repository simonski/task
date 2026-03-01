.PHONY: help default build tools bump-version test test-go test-playwright clean

VERSION_FILE := cmd/task/VERSION

default: help

help:
	@printf "Available targets:\n\n"
	@printf "  make build           Build the task binary into ./bin.\n"
	@printf "                      Also increments the patch version in ./VERSION.\n"
	@printf "  make tools           Build helper binaries in the repo root.\n"
	@printf "  make test            Run all tests.\n"
	@printf "  make test-go         Run Go tests.\n"
	@printf "  make test-playwright Run browser/frontend smoke checks.\n"
	@printf "  make clean           Remove built binaries from ./bin.\n"
	@printf "\n"

build:
	@$(MAKE) bump-version
	@mkdir -p bin
	go build -o ./bin/task ./cmd/task

tools:
	go build -o ./parser ./tools/parser.go

bump-version:
	@if [ ! -f "$(VERSION_FILE)" ]; then \
		printf "0.1.0\n" > "$(VERSION_FILE)"; \
	else \
		version=$$(tr -d '[:space:]' < "$(VERSION_FILE)"); \
		major=$${version%%.*}; \
		rest=$${version#*.}; \
		minor=$${rest%%.*}; \
		patch=$${rest##*.}; \
		patch=$$((patch + 1)); \
		printf "%s.%s.%s\n" "$$major" "$$minor" "$$patch" > "$(VERSION_FILE)"; \
	fi

test: test-go test-playwright

test-go:
	go test ./...

test-playwright:
	@printf "No Playwright tests implemented yet; frontend smoke checks are deferred.\n"

clean:
	@rm -rf bin
	@rm -f parser
