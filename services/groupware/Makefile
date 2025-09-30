SHELL := bash
NAME := groupware

ifneq (, $(shell command -v go 2> /dev/null)) # suppress `command not found warnings` for non go targets in CI
include ../../.bingo/Variables.mk
endif

include ../../.make/default.mk
include ../../.make/go.mk
include ../../.make/release.mk
include ../../.make/docs.mk

.PHONY: apidoc
apidoc: swagger.yml

.PHONY: tsnode
tsnode: node_modules

.PHONY: node_modules
node_modules:
	pnpm install

.PHONY: swagger.yml
swagger.yml:	apidoc.yml tsnode
	swagger generate spec --include='groupware' --include='jmap' --include='jscalendar' --include='jscontact' --scan-models --input=$< | NODE_OPTIONS='--no-warnings' pnpm exec ts-node apidoc-process.ts > $@

APIDOC_PORT=9999

.PHONY: serve-apidoc
serve-apidoc: swagger.yml tsnode
	swagger serve --no-open --port=$(APIDOC_PORT) --host=127.0.0.1 --flavor=redoc $<

api.html: swagger.yml favicon.png tsnode
	pnpm exec redocly build-docs --output=$@.template --title="OpenCloud Groupware API" --theme.openapi.hideHostname=false --theme.openapi.hideTryItPanel=false --theme.openapi.pathInMiddlePanel=true $<
	NODE_OPTIONS='--no-warnings' pnpm exec ts-node ./apidoc-postprocess-html.ts favicon.png < $@.template > $@
	rm $@.template

.PHONY: apidoc-static
apidoc-static: api.html
