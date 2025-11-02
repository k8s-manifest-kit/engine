MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
LOCAL_BIN_PATH := ${PROJECT_PATH}/bin

LINT_GOGC := 10
LINT_TIMEOUT := 10m

## Tools
GOLANGCI_VERSION ?= v2.6.0
GOLANGCI ?= go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif


ifndef ignore-not-found
  ignore-not-found = false
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: test

.PHONY: clean
clean:
	go clean -x
	go clean -x -testcache

.PHONY: fmt
fmt:
	@$(GOLANGCI) fmt --config .golangci.yml
	go fmt ./...

.PHONY: test
test: test/lib test/examples

.PHONY: test/examples
test/examples:
	go test -v ./examples/...

.PHONY: test/lib
test/lib:
	go test -v ./pkg/...

.PHONY: bench
bench:
	go test -bench=. -benchmem ./pkg/renderer/... ./pkg/engine/...

.PHONY: bench/helm
bench/helm:
	go test -bench=. -benchmem ./pkg/renderer/helm

.PHONY: bench/gotemplate
bench/gotemplate:
	go test -bench=. -benchmem ./pkg/renderer/gotemplate

.PHONY: bench/kustomize
bench/kustomize:
	go test -bench=. -benchmem ./pkg/renderer/kustomize

.PHONY: bench/yaml
bench/yaml:
	go test -bench=. -benchmem ./pkg/renderer/yaml

.PHONY: bench/engine
bench/engine:
	go test -bench=. -benchmem ./pkg/engine

.PHONY: bench/engine/parallel
bench/engine/parallel:
	go test -bench=BenchmarkEngine.*Parallel -benchmem ./pkg/engine

.PHONY: bench/engine/sequential
bench/engine/sequential:
	go test -bench=BenchmarkEngine.*Sequential -benchmem ./pkg/engine

.PHONY: bench/engine/helm
bench/engine/helm:
	go test -bench=BenchmarkEngineHelm -benchmem -benchtime=10x ./pkg/engine

.PHONY: bench/compare
bench/compare:
	go test -bench=. -benchmem -benchtime=10s ./pkg/renderer/...

.PHONY: deps
deps:
	go mod tidy

.PHONY: lint
lint:
	@$(GOLANGCI) run --config .golangci.yml --timeout $(LINT_TIMEOUT)

.PHONY: lint/fix
lint/fix:
	@$(GOLANGCI) run --config .golangci.yml --timeout $(LINT_TIMEOUT) --fix

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)


