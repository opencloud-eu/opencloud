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

.PHONY: swagger.yml
swagger.yml:	apidoc.yml
	swagger generate spec --output=$@ --include='groupware' --include='jmap' --scan-models --input=$<

APIDOC_PORT=9999

.PHONY: serve-apidoc
serve-apidoc: swagger.yml
	swagger serve --no-open --port=$(APIDOC_PORT) --host=127.0.0.1 --flavor=redoc $<
