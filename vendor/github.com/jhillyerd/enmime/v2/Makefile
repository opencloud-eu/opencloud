SHELL := /bin/sh

SRC := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
PKGS := $(shell go list ./... | grep -v /vendor/)

.PHONY: all build clean fmt lint reflex simplify test

all: clean test lint build

clean:
	go clean $(PKGS)

deps:
	go get ./...

build:
	go build

test:
	go test -race ./...

fmt:
	@gofmt -l -w $(SRC)

simplify:
	@gofmt -s -l -w $(SRC)

lint:
	@test -z "$(shell gofmt -l . | tee /dev/stderr)" || echo "[WARN] Fix formatting issues with 'make fmt'"
	@go vet $(PKGS)

reflex:
	reflex -r '\.go$$' -- sh -c 'echo; date; echo; go test ./... && echo ALL PASS'
