# goboot developer tasks. Run `make help` for a summary.

GO ?= go

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build all packages
	$(GO) build ./...

.PHONY: test
test: ## Run all tests
	$(GO) test ./...

.PHONY: race
race: ## Run all tests with the race detector
	$(GO) test -race ./...

.PHONY: cover
cover: ## Run tests with coverage and print the total
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -1

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: fmt
fmt: ## Format all Go files
	gofmt -w .

.PHONY: fmt-check
fmt-check: ## Fail if any file is not gofmt-clean
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Not gofmt-clean:"; echo "$$unformatted"; exit 1; \
	fi

.PHONY: fuzz
fuzz: ## Run the parser fuzz targets briefly
	$(GO) test ./annotation/ -run '^$$' -fuzz FuzzParseComment -fuzztime 15s
	$(GO) test ./sqlgen/ -run '^$$' -fuzz FuzzCompile -fuzztime 15s

.PHONY: check
check: fmt-check vet build race ## Run the full pre-PR gate

.PHONY: install
install: ## Install the goboot CLI
	$(GO) install ./cmd/goboot

.PHONY: tidy
tidy: ## Tidy go.mod/go.sum
	$(GO) mod tidy
