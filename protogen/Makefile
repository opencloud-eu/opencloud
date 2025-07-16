SHELL := bash
NAME := protogen

ifneq (, $(shell command -v go 2> /dev/null)) # suppress `command not found warnings` for non go targets in CI
include ../.bingo/Variables.mk
endif
include ../.make/default.mk

.PHONY: go-generate
go-generate: $(MOCKERY)
	$(MOCKERY)

.PHONY: clean
clean:
	$(NOOP)
