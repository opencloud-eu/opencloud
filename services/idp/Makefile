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
		assets/identifier/static/favicon.ico \
		assets/identifier/static/opentalk-500.woff2 \
		assets/identifier/static/opentalk-750.woff2

assets/identifier/static:
	mkdir -p assets/identifier/static

# check if urls are stable, if not replace with stable location
FAVICON_URL = https://opencloud.eu/themes/cloudy/src/assets/favicon/favicon.ico
FONT_VARIATION_500_URL = https://gitlab.opencode.de/opentalk/web-frontend/-/raw/main/app/public/assets/fonts/opentalk/regular/opentalk-regular.woff2
FONT_VARIATION_750_URL = https://gitlab.opencode.de/opentalk/web-frontend/-/raw/main/app/public/assets/fonts/opentalk/bold/opentalk-bold.woff2

.PHONY: assets/identifier/static/favicon.ico # force overwrite
assets/identifier/static/favicon.ico:
	curl --fail -o assets/identifier/static/favicon.ico ${FAVICON_URL}

.PHONY: assets/identifier/static/opentalk-500.woff2
assets/identifier/static/opentalk-500.woff2:
	curl --fail -o assets/identifier/static/opentalk-500.woff2 ${FONT_VARIATION_500_URL}

.PHONY: assets/identifier/static/opentalk-750.woff2
assets/identifier/static/opentalk-750.woff2:
	curl --fail -o assets/identifier/static/opentalk-750.woff2 ${FONT_VARIATION_750_URL}

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
