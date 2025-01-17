SHELL := bash
NAME := idp

include ../../.make/recursion.mk

############ tooling ############
ifneq (, $(shell command -v go 2> /dev/null)) # suppress `command not found warnings` for non go targets in CI
include ../../.bingo/Variables.mk
endif

############ go tooling ############
include ../../.make/go.mk

############ release ############
include ../../.make/release.mk

############ docs generate ############
include ../../.make/docs.mk

.PHONY: docs-generate
docs-generate: config-docs-generate

############ generate ############
include ../../.make/generate.mk

.PHONY: ci-go-generate
ci-go-generate: # CI runs ci-node-generate automatically before this target

.PHONY: ci-node-generate
ci-node-generate: assets

.PHONY: assets
assets: pnpm-build \
		assets/identifier/static \
		assets/identifier/static/favicon.ico

assets/identifier/static:
	mkdir -p assets/identifier/static

# check if url is stable, if not replace with stable location
FAVICON_URL = https://opencloud.eu/themes/salsa/dist/assets/favicon/favicon.ico

.PHONY: assets/identifier/static/favicon.ico # force overwrite
assets/identifier/static/favicon.ico:
	curl --fail -o assets/identifier/static/favicon.ico ${FAVICON_URL}

.PHONY: pnpm-build
pnpm-build: node_modules
	#pnpm lint #TODO: activate
	#pnpm test #TODO: activate
	pnpm build

.PHONY: node_modules
node_modules:
	pnpm install

############ licenses ############
.PHONY: ci-node-check-licenses
ci-node-check-licenses: node_modules
	pnpm licenses:check

.PHONY: ci-node-save-licenses
ci-node-save-licenses: node_modules
	pnpm licenses:csv
	pnpm licenses:save
