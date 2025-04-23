SHELL := bash
NAME := activitylog
OUTPUT_DIR = ./pkg/service/l10n
TEMPLATE_FILE = ./pkg/service/l10n/activitylog.pot

ifneq (, $(shell command -v go 2> /dev/null)) # suppress `command not found warnings` for non go targets in CI
include ../../.bingo/Variables.mk
endif

include ../../.make/default.mk
include ../../.make/go.mk
include ../../.make/release.mk
include ../../.make/docs.mk

.PHONY: l10n-pull
l10n-pull:
	cd $(OUTPUT_DIR) && tx pull --all --force --skip --minimum-perc=75

.PHONY: l10n-push
l10n-push:
	cd $(OUTPUT_DIR) && tx push -s --skip

.PHONY: l10n-read
l10n-read: $(GO_XGETTEXT)
	$(GO_XGETTEXT) -o $(OUTPUT_DIR)/activitylog.pot --keyword=l10n.Template -s pkg/service/response.go

.PHONY: l10n-clean
l10n-clean:
	rm -f $(TEMPLATE_FILE);
