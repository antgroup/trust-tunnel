SHELL := /bin/bash

# Define output directory.
OUTPUT_DIR := out
VERSION ?= latest

# Go environment variables and commands.
GO := GO111MODULE=on go
GO_PATH = $(shell $(GO) env GOPATH)
GO_BUILD = $(GO) build
GO_TEST = $(GO) test
GO_LINT = $(GO_PATH)/bin/golangci-lint

TONGSUO_HOME ?= /opt/tongsuo

# NTLS related compile options.
NTLS_CGO_ENABLED = CGO_ENABLED=1
NTLS_CGO_CFLAGS = "-I${TONGSUO_HOME}/include -Wno-deprecated-declarations"
NTLS_CGO_LDFLAGS = "-L${TONGSUO_HOME}/lib"
NTLS_LD_LIBRARY_PATH = ${TONGSUO_HOME}/lib

# Linker flags to inject version information into the agent and client.
LDFLAGS_AGENT := "-X 'trust-tunnel/cmd/trust-tunnel-agent/app.Version=$(VERSION)'"
LDFLAGS_CLIENT := "-X 'trust-tunnel/cmd/trust-tunnel-client/app.Version=$(VERSION)'"

# Define supported target operating systems and architectures.
TARGETS := linux_amd64 linux_arm64

# .PHONY to declare non-file targets.
.PHONY: all version lint prepare iamges clean trust-tunnel-agent-all trust-tunnel-client-all $(TARGETS)

# Default target.
all: trust-tunnel-agent trust-tunnel-client trust-tunnel-agent-all trust-tunnel-client-all

# Output the current version.
version:
	@echo "Current Version: $(VERSION)"

# Run linting.
lint: $(GO_LINT)
	$(GO_LINT) run -v ./...

# Prepare the output directory.
prepare:
	@mkdir -p $(OUTPUT_DIR)

# Helper function to build targets with NTLS check.
define build_target
	$(eval OS := $(firstword $(subst _, ,$2)))
	$(eval ARCH := $(lastword $(subst _, ,$2)))
	$(eval LDFLAGS := $(3))
	@mkdir -p $(OUTPUT_DIR)/$(OS)_$(ARCH)
	@if [ "$(NTLS_ENABLED)" = "1" ]; then \
		echo "$(NTLS_CGO_ENABLED) GOOS=$(OS) GOARCH=$(ARCH) CGO_CFLAGS=$(NTLS_CGO_CFLAGS) CGO_LDFLAGS=$(NTLS_CGO_LDFLAGS) LD_LIBRARY_PATH=$(NTLS_LD_LIBRARY_PATH) $(GO_BUILD) -ldflags=$(LDFLAGS) -o $(OUTPUT_DIR)/$(OS)_$(ARCH)/trust-tunnel-$(1) ./cmd/trust-tunnel-$(1)"; \
		$(NTLS_CGO_ENABLED) GOOS=$(OS) GOARCH=$(ARCH) CGO_CFLAGS=$(NTLS_CGO_CFLAGS) CGO_LDFLAGS=$(NTLS_CGO_LDFLAGS) LD_LIBRARY_PATH=$(NTLS_LD_LIBRARY_PATH) $(GO_BUILD) -ldflags=$(LDFLAGS) -tags ntls -o $(OUTPUT_DIR)/$(OS)_$(ARCH)/trust-tunnel-$(1) ./cmd/trust-tunnel-$(1); \
	else \
		echo "CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) $(GO_BUILD) -ldflags=$(LDFLAGS) -o $(OUTPUT_DIR)/$(OS)_$(ARCH)/trust-tunnel-$(1) ./cmd/trust-tunnel-$(1)"; \
		CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) $(GO_BUILD) -ldflags=$(LDFLAGS) -o $(OUTPUT_DIR)/$(OS)_$(ARCH)/trust-tunnel-$(1) ./cmd/trust-tunnel-$(1); \
	fi
endef

# Build the 'trust-tunnel-agent' binary.
trust-tunnel-agent: prepare
	@if [ "$(NTLS_ENABLED)" = "1" ]; then \
		echo "$(NTLS_CGO_ENABLED) GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) CGO_CFLAGS=$(NTLS_CGO_CFLAGS) CGO_LDFLAGS=$(NTLS_CGO_LDFLAGS) LD_LIBRARY_PATH=$(NTLS_LD_LIBRARY_PATH) $(GO_BUILD) -ldflags=$(LDFLAGS_AGENT) -tags ntls -o $(OUTPUT_DIR)/trust-tunnel-agent ./cmd/trust-tunnel-agent"; \
		$(NTLS_CGO_ENABLED) GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) CGO_CFLAGS=$(NTLS_CGO_CFLAGS) CGO_LDFLAGS=$(NTLS_CGO_LDFLAGS) LD_LIBRARY_PATH=$(NTLS_LD_LIBRARY_PATH) $(GO_BUILD) -ldflags=$(LDFLAGS_AGENT) -tags ntls -o $(OUTPUT_DIR)/trust-tunnel-agent ./cmd/trust-tunnel-agent; \
	else \
		echo "CGO_ENABLED=0 GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) $(GO_BUILD) -ldflags=$(LDFLAGS_AGENT) -o $(OUTPUT_DIR)/trust-tunnel-agent ./cmd/trust-tunnel-agent"; \
		CGO_ENABLED=0 GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) $(GO_BUILD) -ldflags=$(LDFLAGS_AGENT) -o $(OUTPUT_DIR)/trust-tunnel-agent ./cmd/trust-tunnel-agent; \
	fi

# Build the 'trust-tunnel-client' binary.
trust-tunnel-client: prepare
	@if [ "$(NTLS_ENABLED)" = "1" ]; then \
		echo "$(NTLS_CGO_ENABLED) GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) CGO_CFLAGS=$(NTLS_CGO_CFLAGS) CGO_LDFLAGS=$(NTLS_CGO_LDFLAGS) LD_LIBRARY_PATH=$(NTLS_LD_LIBRARY_PATH) $(GO_BUILD) -ldflags=$(LDFLAGS_CLIENT) -tags ntls -o $(OUTPUT_DIR)/trust-tunnel-client ./cmd/trust-tunnel-client"; \
		$(NTLS_CGO_ENABLED) GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) CGO_CFLAGS=$(NTLS_CGO_CFLAGS) CGO_LDFLAGS=$(NTLS_CGO_LDFLAGS) LD_LIBRARY_PATH=$(NTLS_LD_LIBRARY_PATH) $(GO_BUILD) -ldflags=$(LDFLAGS_CLIENT) -tags ntls -o $(OUTPUT_DIR)/trust-tunnel-client ./cmd/trust-tunnel-client; \
	else \
		echo "CGO_ENABLED=0 GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) $(GO_BUILD) -ldflags=$(LDFLAGS_CLIENT) -o $(OUTPUT_DIR)/trust-tunnel-client ./cmd/trust-tunnel-client"; \
		CGO_ENABLED=0 GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) $(GO_BUILD) -ldflags=$(LDFLAGS_CLIENT) -o $(OUTPUT_DIR)/trust-tunnel-client ./cmd/trust-tunnel-client; \
	fi

# Build 'trust-tunnel-agent' for all supported target platforms.
trust-tunnel-agent-all: $(addprefix trust-tunnel-agent-, $(TARGETS))

# Build 'trust-tunnel-client' for all supported target platforms.
trust-tunnel-client-all: $(addprefix trust-tunnel-client-, $(TARGETS))

# Build the trust-tunnel-agent for a specific OS and ARCH.
trust-tunnel-agent-%: prepare
	$(call build_target,agent,$*,${LDFLAGS_AGENT})

# Build the trust-tunnel-client for a specific OS and ARCH.
trust-tunnel-client-%: prepare
	$(call build_target,client,$*,${LDFLAGS_CLIENT})

# Clean task to remove generated binaries.
clean:
	@rm -rf $(OUTPUT_DIR)

# build trust-tunnel-agent image
images:
	docker build -t trust-tunnel-agent -f ./build/trust-tunnel-agent/Dockerfile .
	docker build -t trust-tunnel-sidecar -f ./build/trust-tunnel-sidecar/Dockerfile .