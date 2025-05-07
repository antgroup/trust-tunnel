# go.mk is a Go project general Makefile, encapsulated some common Target.
# Project repository: https://github.com/elliotxx/go-makefile

APPROOT     		?= $(shell basename $(PWD))
GOPKG       		?= $(shell go list 2>/dev/null)
GOPKGS      		?= $(shell go list ./... 2>/dev/null)
GOSOURCES   		?= $(shell find . -type f -name '*.go' ! -path '*Godeps/_workspace*')
# You can also customize GOSOURCE_PATHS, e.g. ./pkg/... ./cmd/...
GOSOURCE_PATHS		?= ././...


# Go tools
GOFORMATER			?= gofumpt
GOFORMATER_VERSION	?= v0.2.0
GOLINTER			?= golangci-lint
GOLINTER_VERSION	?= v1.58.2

LICENSE_CHECKER ?= license-eye
LICENSE_CHECKER_VERSION ?= main


# To generate help information
.DEFAULT_GOAL := help
.PHONY: help
help:  ## This help message :)
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' go.mk | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: format
format:  ## Format source code of frontend and backend
	@which $(GOFORMATER) > /dev/null || (echo "Installing $(GOFORMATER)@$(GOFORMATER_VERSION) ..."; go install mvdan.cc/gofumpt@$(GOFORMATER_VERSION) && echo -e "Installation complete!\n")
	@find . -name "*.go" -exec gofumpt -w -e {} +;

.PHONY: lint
lint:  ## Lint, will not fix but sets exit code on error
	@which $(GOLINTER) > /dev/null || (echo "Installing $(GOLINTER)@$(GOLINTER_VERSION) ..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLINTER_VERSION) && echo -e "Installation complete!\n")
	$(GOLINTER) run $(GOSOURCE_PATHS)

.PHONY: lint-fix
lint-fix:  ## Lint, will try to fix errors and modify code
	@which $(GOLINTER) > /dev/null || (echo "Installing $(GOLINTER)@$(GOLINTER_VERSION) ..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLINTER_VERSION) && echo -e "Installation complete!\n")
	$(GOLINTER) run  $(GOSOURCE_PATHS) --fix

.PHONY: check-license
check-license:  ## Checks if repo files contain valid license header
	@which $(LICENSE_CHECKER) > /dev/null || (echo "Installing $(LICENSE_CHECKER)@$(LICENSE_CHECKER_VERSION) ..."; go install github.com/apache/skywalking-eyes/cmd/$(LICENSE_CHECKER)@$(LICENSE_CHECKER_VERSION) && echo -e "Installation complete!\n")
	@${GOPATH}/bin/$(LICENSE_CHECKER) header check

.PHONY: fix-license
fix-license:  ## Adds missing license header to repo files
	@which $(LICENSE_CHECKER) > /dev/null || (echo "Installing $(LICENSE_CHECKER)@$(LICENSE_CHECKER_VERSION) ..."; go install github.com/apache/skywalking-eyes/cmd/$(LICENSE_CHECKER)@$(LICENSE_CHECKER_VERSION) && echo -e "Installation complete!\n")
	@${GOPATH}/bin/$(LICENSE_CHECKER) header fix
