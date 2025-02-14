SHELL := bash
NAME := web
WEB_ASSETS_VERSION = 552ff3c3f8e9543de653537087161c070698661e

include ../../.make/recursion.mk

############ tooling ############
ifneq (, $(shell command -v go 2> /dev/null)) # suppress `command not found warnings` for non go targets in CI
include ../../.bingo/Variables.mk
endif

############ go tooling ############
include ../../.make/go.mk

############ release #########
include ../../.make/release.mk

############ docs generate ############
include ../../.make/docs.mk

.PHONY: docs-generate
docs-generate: config-docs-generate

############ generate ############
include ../../.make/generate.mk

.PHONY: ci-go-generate
ci-go-generate: # CI runs ci-node-generate automatically before this target

.PHONY: ci-node-generate-no-clone
ci-node-generate-no-clone: make-assets

.PHONY: ci-node-generate
ci-node-generate: pull-assets

.PHONY: make-assets
make-assets:
	make -C assets/core/origin release
	tar xfv assets/core/origin/release/web.tar.gz -C assets/core/
	rm -rf assets/core/origin

.PHONY: pull-assets
pull-assets:
	git clean -xfd assets
	git clone -b main --depth 1 https://github.com/opencloud-eu/web.git assets/core/origin
	make -C assets/core/origin release
	tar xfv assets/core/origin/release/web.tar.gz -C assets/core/
	rm -rf assets/core/origin

############ licenses ############
.PHONY: ci-node-check-licenses
ci-node-check-licenses: # OpenCloud Web does the check on it's own

.PHONY: ci-node-save-licenses
ci-node-save-licenses:
	@mkdir -p ../../third-party-licenses/node/web
	@curl --fail -slL -o- https://github.com/opencloud-eu/web/releases/download/$(WEB_ASSETS_VERSION)/third-party-licenses.tar.gz | tar xzf - -C ../../third-party-licenses/node/web
