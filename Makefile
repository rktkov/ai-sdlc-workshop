GO ?= go
GOLANGCI_VERSION := v2.6.2
COVERAGE_THRESHOLD ?= 80

.PHONY: help lint test coverage coverage-check build check clean

help: ## Show this target list
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "%-18s %s\n", $$1, $$2}'

lint: ## Run golangci-lint with the pinned configuration
	golangci-lint run

test: ## Run unit tests
	$(GO) test ./...

coverage: ## Show coverage per package
	$(GO) test -cover ./...

coverage-check: ## Fail when a tested package is below the threshold
	@out=$$($(GO) test -cover ./...); echo "$$out"; echo "$$out" | awk -v t=$(COVERAGE_THRESHOLD) '/coverage:/ {gsub("%", "", $$5); if ($$5 + 0 < t) {print "coverage below threshold: " $$2 " " $$5 "%"; bad = 1}} END {exit bad}'

build: ## Compile all packages
	$(GO) build ./...

check: lint test build ## Run the full local gate

clean: ## Remove local build output
	$(GO) clean ./...
