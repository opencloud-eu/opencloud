SHELL := bash
NAME := thumbnails

ifneq (, $(shell command -v go 2> /dev/null)) # suppress `command not found warnings` for non go targets in CI
include ../../.bingo/Variables.mk
endif

include ../../.make/default.mk
include ../../.make/go.mk
include ../../.make/release.mk
include ../../.make/docs.mk
include ../../.make/protobuf.mk

.PHONY: go-generate
go-generate: protobuf

.PHONY: protobuf
protobuf: buf-generate
