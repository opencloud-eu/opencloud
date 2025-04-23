SHELL := bash
NAME := idp

ifneq (, $(shell command -v go 2> /dev/null)) # suppress `command not found warnings` for non go targets in CI
include ../../.bingo/Variables.mk
endif

include ../../.make/default.mk
include ../../.make/go.mk
include ../../.make/release.mk
include ../../.make/docs.mk

.PHONY: node-generate-prod
node-generate-prod: assets

.PHONY: assets
assets: pnpm-build \
		assets/identifier/static/favicon.svg \
		assets/identifier/static/icon-lilac.svg

.PHONY: assets/identifier/static/favicon.svg # force overwrite
assets/identifier/static/favicon.svg: pnpm-build
	cp src/images/favicon.svg assets/identifier/static/favicon.svg
	rm assets/identifier/static/favicon.ico

.PHONY: assets/identifier/static/icon-lilac.svg
assets/identifier/static/icon-lilac.svg: pnpm-build
	cp src/images/icon-lilac.svg assets/identifier/static/icon-lilac.svg

.PHONY: pnpm-build
pnpm-build: node_modules
	#pnpm lint #TODO: activate
	#pnpm test #TODO: activate
	pnpm build

.PHONY: node_modules
node_modules:
	pnpm install

.PHONY: ci-node-check-licenses
ci-node-check-licenses: node_modules
	pnpm licenses:check

.PHONY: ci-node-save-licenses
ci-node-save-licenses: node_modules
	pnpm licenses:csv
	pnpm licenses:save
