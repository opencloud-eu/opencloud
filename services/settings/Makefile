SHELL := bash
NAME := settings
OUTPUT_DIR = ./pkg/service/v0/l10n
TEMPLATE_FILE = ./pkg/service/v0/l10n/settings.pot

ifneq (, $(shell command -v go 2> /dev/null)) # suppress `command not found warnings` for non go targets in CI
include ../../.bingo/Variables.mk
endif

include ../../.make/default.mk
include ../../.make/go.mk
include ../../.make/release.mk
include ../../.make/docs.mk
include ../../.make/protobuf.mk

.PHONY: docs-generate
docs-generate: grpc-docs-generate

.PHONY: go-generate
go-generate: $(MOCKERY) protobuf
	$(MOCKERY)

.PHONY: protobuf
protobuf: buf-generate $(PROTOC_GO_INJECT_TAG)
	cd ../../protogen/gen/ && $(PROTOC_GO_INJECT_TAG) -input="opencloud/messages/settings/v0/*.pb.go"

.PHONY: l10n-pull
l10n-pull:
	cd $(OUTPUT_DIR) && tx pull --all --force --skip --minimum-perc=75

.PHONY: l10n-push
l10n-push:
	cd $(OUTPUT_DIR) && tx push -s --skip

.PHONY: l10n-read
l10n-read: $(GO_XGETTEXT)
	$(GO_XGETTEXT) -o $(OUTPUT_DIR)/settings.pot \
	--keyword=l10n.Template --add-comments -s \
	pkg/store/defaults/templates.go

.PHONY: l10n-clean
l10n-clean:
	rm -f $(TEMPLATE_FILE);
