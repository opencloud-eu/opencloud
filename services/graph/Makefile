SHELL := bash
NAME := graph
OUTPUT_DIR = ./pkg/l10n
TEMPLATE_FILE = ./pkg/l10n/graph.pot

ifneq (, $(shell command -v go 2> /dev/null)) # suppress `command not found warnings` for non go targets in CI
include ../../.bingo/Variables.mk
endif

include ../../.make/default.mk
include ../../.make/go.mk
include ../../.make/release.mk
include ../../.make/docs.mk

.PHONY: go-generate
go-generate: $(MOCKERY)
	$(MOCKERY)

.PHONY: l10n-pull
l10n-pull:
	cd $(OUTPUT_DIR) && tx pull --all --force --skip --minimum-perc=75

.PHONY: l10n-push
l10n-push:
	cd $(OUTPUT_DIR) && tx push -s --skip

.PHONY: l10n-read
l10n-read: $(GO_XGETTEXT)
	$(GO_XGETTEXT) -o $(OUTPUT_DIR)/graph.pot --keyword=l10n.Template --add-comments -s pkg/service/v0/spacetemplates.go -s pkg/unifiedrole/roles.go

.PHONY: l10n-clean
l10n-clean:
	rm -f $(TEMPLATE_FILE);
