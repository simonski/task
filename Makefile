.PHONY: help default build tools bump-version test test-go test-go-cover test-playwright clean

VERSION_FILE := cmd/ticket/VERSION

default: help

help:
	@printf "Available targets:\n\n"
	@printf "  make build           Build the ticket binary into ./bin.\n"
	@printf "                       Also increments the patch version in ./VERSION.\n"
	@printf "  make tools           Build helper binaries in the repo root.\n"
	@printf "  make test            Run all tests.\n"
	@printf "  make test-go         Run Go tests.\n"
	@printf "  make test-go-cover   Run Go tests with package coverage thresholds.\n"
	@printf "  make test-playwright Run browser/frontend smoke checks.\n"
	@printf "  make clean           Remove built binaries from ./bin.\n"
	@printf "\n"

build:
	@$(MAKE) bump-version
	@mkdir -p bin
	go build -o ./bin/ticket ./cmd/ticket

tools:
	@mkdir -p bin
	@set -e; \
	for tool in $$(find tools -mindepth 2 -maxdepth 2 -type f -name '*.go' ! -name '*_test.go' | sort); do \
		name=$$(basename $$(dirname $$tool)); \
		printf "Building %s -> bin/%s\n" "$$tool" "$$name"; \
		go build -o "bin/$$name" "$$tool"; \
	done

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

test-go-cover:
	@set -e; \
	for entry in \
		"./cmd/ticket 55" \
		"./libticket 65" \
		"./libtickethttp 75" \
		"./internal/client 55" \
		"./internal/store 70" \
		"./internal/config 70" \
		"./tools/parser 75"; do \
		pkg=$${entry% *}; \
		min=$${entry#* }; \
		out=$$(go test "$$pkg" -cover | tail -n 1); \
		printf "%s\n" "$$out"; \
		pct=$$(printf "%s" "$$out" | sed -n 's/.*coverage: \([0-9.]*\)%.*/\1/p'); \
		if [ -z "$$pct" ]; then \
			printf "could not parse coverage for %s\n" "$$pkg" >&2; \
			exit 1; \
		fi; \
		awk -v pct="$$pct" -v min="$$min" 'BEGIN { if (pct + 0 < min + 0) exit 1 }' || { \
			printf "coverage threshold failed for %s: got %s%%, need %s%%\n" "$$pkg" "$$pct" "$$min" >&2; \
			exit 1; \
		}; \
	done

test-playwright:
	@printf "No Playwright tests implemented yet; frontend smoke checks are deferred.\n"

clean:
	@rm -rf bin
	@rm -f parser
