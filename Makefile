SHELL := /bin/bash

GO ?= go
GO_PACKAGES := ./...
# Prefer tracked .go files so we match `git` state; if none (e.g. not a repo / nothing committed yet), scan the tree.
GO_FILES := $(shell git ls-files '*.go' 2>/dev/null)
ifeq ($(strip $(GO_FILES)),)
GO_FILES := $(shell find . \( -path './vendor' -o -path './.git' \) -prune -o -name '*.go' -print 2>/dev/null | sort)
endif

## Location to install dependencies to
LOCALBIN ?= $(CURDIR)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool binaries
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
ADDLICENSE ?= $(LOCALBIN)/addlicense

## Tool versions
GOLANGCI_LINT_VERSION ?= v2.11.4
ADDLICENSE_VERSION ?= latest

.PHONY: test lint license add-license license-header tools golangci-lint addlicense go-check arch-test

test:
	$(GO) test -race -cover $(GO_PACKAGES)

tools: golangci-lint addlicense

lint: golangci-lint
	$(GOLANGCI_LINT) run $(GO_PACKAGES) --fix

# arch-test enforces the hexagonal dependency rules described in
# refactoring_plan.md §3.3 by running depguard via golangci-lint and
# failing the build on any violation. It is a focused subset of `make
# lint` intended for use in CI as a separate gate.
arch-test: golangci-lint
	$(GOLANGCI_LINT) run $(GO_PACKAGES) --enable-only=depguard

# Install the binary (google/addlicense). To actually write headers, use: make license-header (or: make add-license)
addlicense: $(ADDLICENSE) ## Download addlicense locally if necessary.
$(ADDLICENSE): $(LOCALBIN)
	$(call go-install-tool,$(ADDLICENSE),github.com/google/addlicense,$(ADDLICENSE_VERSION))

# Add Apache-2.0 file headers to Go sources (see hack/license-header.txt)
license-header: addlicense
	@if [ -z "$(GO_FILES)" ]; then \
		echo "license-header: no .go files found (not in repo? run from repo root, or add/commit .go files)"; \
		exit 1; \
	fi
	$(ADDLICENSE) -f hack/license-header.txt $(GO_FILES)

add-license: license-header
license: license-header

go-check:
	$(call error-if-empty,$(GO),go)

golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint v2 locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) $(GO) install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

define error-if-empty
@if [ -z "$(1)" ]; then echo "$(2) not installed"; false; fi
endef
