OC_REPO := github.com/opencloud-eu/opencloud
IMPORT := ($OC_REPO)/$(NAME)
BIN := bin
DIST := dist

ifeq ($(OS), Windows_NT)
	EXECUTABLE := $(NAME).exe
	UNAME := Windows
else
	EXECUTABLE := $(NAME)
	UNAME := $(shell uname -s)
endif

GOBUILD ?= go build

SOURCES ?= $(shell find . -name "*.go" -type f -not -path "./node_modules/*")

TAGS ?=

ifndef OUTPUT
	ifneq ($(CI_COMMIT_TAG),)
		OUTPUT ?= $(subst v,,$(CI_COMMIT_TAG))
	else
		OUTPUT ?= testing
	endif
endif

ifeq ($(VERSION), daily)
	STRING ?= $(shell git rev-parse --short HEAD)
else ifeq ($(VERSION),)
	STRING ?= $(shell git rev-parse --short HEAD)
endif


ifndef DATE
	DATE := $(shell date -u '+%Y%m%d')
endif

LDFLAGS += -X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn -s -w \
	-X "$(OC_REPO)/pkg/version.Edition=$(EDITION)" \
	-X "$(OC_REPO)/pkg/version.String=$(STRING)" \
	-X "$(OC_REPO)/pkg/version.Tag=$(VERSION)" \
	-X "$(OC_REPO)/pkg/version.Date=$(DATE)"

DEBUG_LDFLAGS += -X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn \
	-X "$(OC_REPO)/pkg/version.Edition=$(EDITION)" \
	-X "$(OC_REPO)/pkg/version.String=$(STRING)" \
	-X "$(OC_REPO)/pkg/version.Tag=$(VERSION)" \
	-X "$(OC_REPO)/pkg/version.Date=$(DATE)"

DOCKER_LDFLAGS += -X "$(OC_REPO)/pkg/config/defaults.BaseDataPathType=path" -X "$(OC_REPO)/pkg/config/defaults.BaseDataPathValue=/var/lib/opencloud"
DOCKER_LDFLAGS += -X "$(OC_REPO)/pkg/config/defaults.BaseConfigPathType=path" -X "$(OC_REPO)/pkg/config/defaults.BaseConfigPathValue=/etc/opencloud"

GCFLAGS += all=-N -l

.PHONY: all
all: build

.PHONY: sync
sync:
	go mod download

.PHONY: clean
clean:
	@echo "- $(NAME): clean"
	@go clean -i ./...
	@rm -rf $(BIN) $(DIST)

.PHONY: go-mod-tidy
go-mod-tidy:
	@echo "- $(NAME): go-mod-tidy"
	@go mod tidy

.PHONY: fmt
fmt:
	@echo "- $(NAME): fmt"
	gofmt -s -w $(SOURCES)

.PHONY: test
test:
	@go test -v -tags '$(TAGS)' -coverprofile coverage.out ./...

.PHONY: go-coverage
go-coverage:
	@if [ ! -f coverage.out ]; then $(MAKE) test  &>/dev/null; fi;
	@go tool cover -func coverage.out | tail -1 | grep -Eo "[0-9]+\.[0-9]+"

.PHONY: install
install: $(SOURCES)
	go install -v -tags '$(TAGS)' -ldflags '$(LDFLAGS)' ./cmd/$(NAME)

.PHONY: build-all
build-all: build build-debug

.PHONY: build
build: $(BIN)/$(EXECUTABLE)

.PHONY: build-debug
build-debug: $(BIN)/$(EXECUTABLE)-debug

$(BIN)/$(EXECUTABLE): $(SOURCES)
	$(GOBUILD) -v -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o $@ ./cmd/$(NAME)

$(BIN)/$(EXECUTABLE)-debug: $(SOURCES)
	$(GOBUILD) -v -tags '$(TAGS)' -ldflags '$(DEBUG_LDFLAGS)' -gcflags '$(GCFLAGS)' -o $@ ./cmd/$(NAME)

.PHONY: watch
watch: $(REFLEX)
	$(REFLEX) -c reflex.conf

debug-linux-docker-amd64: release-dirs
	GOOS=linux \
	GOARCH=amd64 \
	go build \
        -gcflags="all=-N -l" \
		-tags 'netgo $(TAGS)' \
		-buildmode=exe \
		-trimpath \
		-ldflags '-extldflags "-static" $(DEBUG_LDFLAGS) $(DOCKER_LDFLAGS)' \
		-o '$(DIST)/binaries/$(EXECUTABLE)-linux-amd64' \
		./cmd/$(NAME)

debug-linux-docker-arm64: release-dirs
	GOOS=linux \
	GOARCH=arm64 \
	go build \
        -gcflags="all=-N -l" \
		-tags 'netgo $(TAGS)' \
		-buildmode=exe \
		-trimpath \
		-ldflags '-extldflags "-static" $(DEBUG_LDFLAGS) $(DOCKER_LDFLAGS)' \
		-o '$(DIST)/binaries/$(EXECUTABLE)-linux-arm64' \
		./cmd/$(NAME)
