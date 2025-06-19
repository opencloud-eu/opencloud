# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.9. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for bingo variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(BINGO)
#	@echo "Running bingo"
#	@$(BINGO) <flags/args..>
#
BINGO := $(GOBIN)/bingo-v0.9.0
$(BINGO): $(BINGO_DIR)/bingo.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/bingo-v0.9.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=bingo.mod -o=$(GOBIN)/bingo-v0.9.0 "github.com/bwplotka/bingo"

BUF := $(GOBIN)/buf-v1.39.0
$(BUF): $(BINGO_DIR)/buf.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/buf-v1.39.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=buf.mod -o=$(GOBIN)/buf-v1.39.0 "github.com/bufbuild/buf/cmd/buf"

BUILDIFIER := $(GOBIN)/buildifier-v0.0.0-20220323134444-a9f46b2bb3de
$(BUILDIFIER): $(BINGO_DIR)/buildifier.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/buildifier-v0.0.0-20220323134444-a9f46b2bb3de"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=buildifier.mod -o=$(GOBIN)/buildifier-v0.0.0-20220323134444-a9f46b2bb3de "github.com/bazelbuild/buildtools/buildifier"

CALENS := $(GOBIN)/calens-v0.4.0
$(CALENS): $(BINGO_DIR)/calens.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/calens-v0.4.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=calens.mod -o=$(GOBIN)/calens-v0.4.0 "github.com/restic/calens"

GO_LICENSES := $(GOBIN)/go-licenses-v1.5.0
$(GO_LICENSES): $(BINGO_DIR)/go-licenses.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/go-licenses-v1.5.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=go-licenses.mod -o=$(GOBIN)/go-licenses-v1.5.0 "github.com/google/go-licenses"

GO_XGETTEXT := $(GOBIN)/go-xgettext-v0.0.0-20160830220431-74466a0a0c4a
$(GO_XGETTEXT): $(BINGO_DIR)/go-xgettext.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/go-xgettext-v0.0.0-20160830220431-74466a0a0c4a"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=go-xgettext.mod -o=$(GOBIN)/go-xgettext-v0.0.0-20160830220431-74466a0a0c4a "github.com/gosexy/gettext/go-xgettext"

GOLANGCI_LINT := $(GOBIN)/golangci-lint-v1.64.6
$(GOLANGCI_LINT): $(BINGO_DIR)/golangci-lint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/golangci-lint-v1.64.6"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=golangci-lint.mod -o=$(GOBIN)/golangci-lint-v1.64.6 "github.com/golangci/golangci-lint/cmd/golangci-lint"

GOVULNCHECK := $(GOBIN)/govulncheck-v1.1.4
$(GOVULNCHECK): $(BINGO_DIR)/govulncheck.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/govulncheck-v1.1.4"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=govulncheck.mod -o=$(GOBIN)/govulncheck-v1.1.4 "golang.org/x/vuln/cmd/govulncheck"

MOCKERY := $(GOBIN)/mockery-v3.4.0
$(MOCKERY): $(BINGO_DIR)/mockery.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/mockery-v3.4.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=mockery.mod -o=$(GOBIN)/mockery-v3.4.0 "github.com/vektra/mockery/v3"

MUTAGEN := $(GOBIN)/mutagen-v0.18.1
$(MUTAGEN): $(BINGO_DIR)/mutagen.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/mutagen-v0.18.1"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=mutagen.mod -o=$(GOBIN)/mutagen-v0.18.1 "github.com/mutagen-io/mutagen/cmd/mutagen"

PIGEON := $(GOBIN)/pigeon-v1.3.0
$(PIGEON): $(BINGO_DIR)/pigeon.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/pigeon-v1.3.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=pigeon.mod -o=$(GOBIN)/pigeon-v1.3.0 "github.com/mna/pigeon"

PROTOC_GEN_DOC := $(GOBIN)/protoc-gen-doc-v1.5.1
$(PROTOC_GEN_DOC): $(BINGO_DIR)/protoc-gen-doc.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/protoc-gen-doc-v1.5.1"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=protoc-gen-doc.mod -o=$(GOBIN)/protoc-gen-doc-v1.5.1 "github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc"

PROTOC_GEN_GO := $(GOBIN)/protoc-gen-go-v1.28.1
$(PROTOC_GEN_GO): $(BINGO_DIR)/protoc-gen-go.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/protoc-gen-go-v1.28.1"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=protoc-gen-go.mod -o=$(GOBIN)/protoc-gen-go-v1.28.1 "google.golang.org/protobuf/cmd/protoc-gen-go"

PROTOC_GEN_MICRO := $(GOBIN)/protoc-gen-micro-v1.0.0
$(PROTOC_GEN_MICRO): $(BINGO_DIR)/protoc-gen-micro.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/protoc-gen-micro-v1.0.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=protoc-gen-micro.mod -o=$(GOBIN)/protoc-gen-micro-v1.0.0 "github.com/go-micro/generator/cmd/protoc-gen-micro"

PROTOC_GEN_MICROWEB := $(GOBIN)/protoc-gen-microweb-v0.0.0-20220808092353-b5d6c3960e19
$(PROTOC_GEN_MICROWEB): $(BINGO_DIR)/protoc-gen-microweb.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/protoc-gen-microweb-v0.0.0-20220808092353-b5d6c3960e19"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=protoc-gen-microweb.mod -o=$(GOBIN)/protoc-gen-microweb-v0.0.0-20220808092353-b5d6c3960e19 "github.com/owncloud/protoc-gen-microweb"

PROTOC_GEN_OPENAPIV2 := $(GOBIN)/protoc-gen-openapiv2-v2.13.0
$(PROTOC_GEN_OPENAPIV2): $(BINGO_DIR)/protoc-gen-openapiv2.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/protoc-gen-openapiv2-v2.13.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=protoc-gen-openapiv2.mod -o=$(GOBIN)/protoc-gen-openapiv2-v2.13.0 "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"

PROTOC_GO_INJECT_TAG := $(GOBIN)/protoc-go-inject-tag-v1.4.0
$(PROTOC_GO_INJECT_TAG): $(BINGO_DIR)/protoc-go-inject-tag.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/protoc-go-inject-tag-v1.4.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=protoc-go-inject-tag.mod -o=$(GOBIN)/protoc-go-inject-tag-v1.4.0 "github.com/favadi/protoc-go-inject-tag"

REFLEX := $(GOBIN)/reflex-v0.3.1
$(REFLEX): $(BINGO_DIR)/reflex.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/reflex-v0.3.1"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=reflex.mod -o=$(GOBIN)/reflex-v0.3.1 "github.com/cespare/reflex"

