# E2E Test Makefile for kubectl-mft
SHELL := /bin/bash

GO := go
GO_TEST_OPTS := -v -race
GINKGO_OPTS := -ginkgo.v

BUILD_TAG := e2e

# Docker settings for test registry
DOCKER := docker
REGISTRY_IMAGE := registry:3

.PHONY: help
help: ## Show this help message
	@echo 'E2E Test Makefile for kubectl-mft'
	@echo ''
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: test
test: check-docker ## Run E2E tests
	@echo "Running E2E tests..."
	$(GO) test -tags=$(BUILD_TAG) $(GO_TEST_OPTS) ./... $(GINKGO_OPTS)

.PHONY: check-docker
check-docker: ## Check if Docker is running
	@if ! $(DOCKER) info > /dev/null 2>&1; then \
		echo "Error: Docker is not running. Please start Docker and try again."; \
		exit 1; \
	fi

.PHONY: pull-registry
pull-registry: check-docker ## Pull registry image for tests
	@echo "Pulling registry image..."
	$(DOCKER) pull $(REGISTRY_IMAGE)

.PHONY: clean
clean: ## Clean up test artifacts
	@echo "Cleaning up test artifacts..."
	@rm -rf /tmp/kubectl-mft-test-*
	@rm -rf /tmp/kubectl-mft
	@$(DOCKER) rm -f $$($(DOCKER) ps -a -q --filter name=kubectl-mft-test) 2>/dev/null || true

.PHONY: setup
setup: pull-registry ## Setup test environment
	@echo "Test environment setup complete"

.PHONY: coverage
coverage: check-docker ## Run E2E tests with coverage
	@echo "Running E2E tests with coverage..."
	$(GO) test -tags=$(BUILD_TAG) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Coverage report:"
	$(GO) tool cover -func=coverage.out

.PHONY: coverage-html
coverage-html: coverage ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.DEFAULT_GOAL := help
