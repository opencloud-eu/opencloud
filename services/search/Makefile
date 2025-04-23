SHELL := bash
NAME := search

ifneq (, $(shell command -v go 2> /dev/null)) # suppress `command not found warnings` for non go targets in CI
include ../../.bingo/Variables.mk
endif

include ../../.make/default.mk
include ../../.make/go.mk
include ../../.make/release.mk
include ../../.make/docs.mk

.PHONY: go-generate
go-generate: $(PIGEON) $(MOCKERY)
	$(MOCKERY)
	$(PIGEON) -optimize-grammar -optimize-parser -o ../../pkg/kql/dictionary_gen.go ../../pkg/kql/dictionary.peg
