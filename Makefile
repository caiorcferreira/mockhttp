ifneq (,$(wildcard ./.env))
    include .env
    export $(shell sed 's/=.*//' .env)
endif

# Library version
VERSION ?= 1.0.0

# Image URL to use all building/pushing image targets
IMG_NAME ?= boilerplate

# Go native variables
GOOS ?= linux
GOARCH ?= amd64

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development
.PHONY: test
test: ## Run unit tests on project.
	@echo "Running unit tests"
	@if [ ! -z "$(GITHUB_RUN_ID)" ]; then go test -run . ./... -race -count=1 -coverprofile coverage.out -json > go-test-report.json; else go test -race -count=1 -cover ./...; fi;

.PHONY: test
tag: ## Create git tag based on application version.
	git tag -a -m "v$(VERSION)" v$(VERSION)

##@ Golang tools

.PHONY: lint
lint: ## Generate sonar report if running on ci
	@echo "Linting with golangci-lint"
	$(MAKE) golangcilint; \
	$(GOLANGLINT) run -c .golangci.yml;

GOLANGLINT = $(shell pwd)/bin/golangci-lint
golangcilint: ## Download golangci-lint locally if necessary.
	$(call go-install,$(GOLANGLINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.3)

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-install
@[ -f $(1) ] || { \
set -e ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
}
endef
