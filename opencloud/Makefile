SHELL := bash
NAME := opencloud
TAGS := disable_crypt
GOARCH := $(shell go env GOARCH)
CONFIG_DOCS_BASE_PATH := ../docs
ifdef ENABLE_VIPS
	TAGS := ${TAGS},enable_vips
endif

ifneq (, $(shell command -v go 2> /dev/null)) # suppress `command not found warnings` for non go targets in CI
include ../.bingo/Variables.mk
endif
include ../.make/default.mk
include ../.make/go.mk
include ../.make/release.mk
include ../.make/docs.mk

.PHONY: dev-docker
dev-docker:
	docker build -f docker/Dockerfile.multiarch -t opencloudeu/opencloud:dev ..

.PHONY: dev-docker-multiarch
dev-docker-multiarch:
	@echo "+-------------------------------------------------------------------------+"
	@echo "| Are you sure you have run make node-generate-prod in the repository root? |"
	@echo "+-------------------------------------------------------------------------+"
	sleep 10
	docker buildx rm opencloudbuilder || true
	docker buildx create --platform linux/arm64,linux/amd64 --name opencloudbuilder
	docker buildx use opencloudbuilder
	cd .. && docker buildx build --platform linux/arm64,linux/amd64 --output type=docker --file opencloud/docker/Dockerfile.multiarch --tag opencloudeu/opencloud:dev-multiarch .
	docker buildx rm opencloudbuilder

.PHONY: debug-docker
debug-docker:
	$(MAKE) --no-print-directory debug-linux-docker-$(GOARCH)
	docker build -f docker/Dockerfile.linux.debug.$(GOARCH) -t opencloudeu/opencloud:debug .
