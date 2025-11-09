SHELL:=/bin/bash

GO = go
GO_VET_OPTS = -v
GO_TEST_OPTS=-v -race

GO_FMT=gofmt
GO_FMT_OPTS=-s -l

GO_IMPORTS=goimports
GO_IMPORTS_OPTS=-w -local github.com/chez-shanpu/kubectl-mft

STATIC_CHECK=staticcheck

.PHONY: fmt
fmt:
	$(GO_FMT) $(GO_FMT_OPTS) .
	$(GO_IMPORTS) $(GO_IMPORTS_OPTS) .

.PHONY: mod
mod:
	$(GO) mod tidy

.PHONY: check-diff
check-diff: mod fmt
	git diff --exit-code --name-only

.PHONY: vet
vet:
	$(GO) vet $(GO_VET_OPTS) ./...

.PHONY: test
test:
	$(STATIC_CHECK) ./...
	$(GO) test $(GO_TEST_OPTS) ./...

.PHONY: test-e2e
test-e2e:
	@$(MAKE) -C test test

.PHONY: build
build:
	$(GO) build -o bin/ .

.PHONY: clean
clean:
	-$(GO) clean
	-rm $(RM_OPTS) $(BIN_DIR)

.PHONY: check
check: vet check-diff test

.PHONY: check-all
check-all: check test-e2e

.PHONY: all
all: check build

.DEFAULT_GOAL=all