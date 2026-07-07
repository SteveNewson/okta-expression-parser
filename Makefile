# Build configuration

# Constants
SHELL=/usr/bin/env bash

LINT_PKGS=./...
GOTESTSUM_FMT="dots"

.DEFAULT_GOAL := help

# Targets

.PHONY: help
help:	## Show the help menu
	@echo "Usage: make <target>"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: deps
deps: ## Install development dependencies
	@echo "Installing dependencies"
	@go mod download -x all
	@GOBIN=$(shell pwd)/bin go install gotest.tools/gotestsum@latest
	@GOBIN=$(shell pwd)/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.10.1
	@GOBIN=$(shell pwd)/bin go install golang.org/x/tools/cmd/goimports@latest

.PHONY: lint
lint: ## Run all linters
	@echo "Running golangci-lint locally"
	@if [ -x ./bin/golangci-lint ]; then \
		./bin/golangci-lint run --build-tags=${CI_TAGS} ${LINT_PKGS} || true; \
	else \
		echo "golangci-lint not found, run 'make deps' first"; \
	fi

.PHONY: vet
vet: ## Run go vet
	@go vet ${BUILD_TAGS} --all ${SDK_PKGS}

.PHONY: fmt
fmt:
	@GOFLAGS= ./bin/goimports -w -format-only .

.PHONY: test
test: unit ## Run all tests

##
# Unit tests
##
.PHONY: unit
unit: lint ## Run all unit tests
	@./bin/gotestsum -f ${GOTESTSUM_FMT} -- ${LIB_INTERNAL}
