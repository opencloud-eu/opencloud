PACKAGE  = github.com/libregraph/lico
PACKAGE_NAME = libregraph-$(shell basename $(PACKAGE))

# Tools

GO      ?= go
GOFMT   ?= gofmt
GOLINT  ?= golangci-lint
DLV     ?= dlv

GOTESTSUM ?= gotestsum
GOCOVERCOBERTURA ?= gocover-cobertura

CHGLOG ?= git-chglog

# Cgo
CGO_ENABLED ?= 0

# Go modules

GO111MODULE ?= on

# Variables

export CGO_ENABLED GO111MODULE
unexport GOPATH

ARGS    ?=
PWD     := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2>/dev/null | sed 's/^v//' || \
			cat $(CURDIR)/.version 2> /dev/null || echo 0.0.0-unreleased)
PKGS     = $(or $(PKG),$(shell $(GO) list -mod=readonly ./... | grep -v "^$(PACKAGE)/vendor/"))
TESTPKGS = $(shell $(GO) list -mod=readonly -f '{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' $(PKGS) 2>/dev/null)
# CURDIR, because make always sets it. PWD comes from a shell, and the
# Dockerfile.build CMD execs make with none, which would leave this find
# looking at /cmd/ and resolve no command targets at all.
CMDS     = $(or $(CMD),$(addprefix cmd/,$(notdir $(shell find "$(CURDIR)/cmd/" -type d))))
TIMEOUT  = 30

GOLINT_ARGS ?= --new

# Debug variables

DLV_APIVERSION ?= 2
DLV_ARGS       ?=
DLV_EXECUTABLE ?= bin/licod
DLV_ATTACH_PID ?= $(shell pgrep -f $(DLV_EXECUTABLE))

# Build

LDFLAGS  ?= -s -w
ASMFLAGS ?=
GCFLAGS  ?=

.PHONY: all
all: vendor | $(CMDS) identifier-webapp

.PHONY: commands
commands: $(CMDS)

.PHONY: $(CMDS)
$(CMDS): vendor ; $(info building $@ ...) @
	$(GO) build \
		-mod vendor \
		-trimpath \
		-tags release \
		-buildmode=exe \
		-asmflags '$(ASMFLAGS)' \
		-gcflags '$(GCFLAGS)' \
		-ldflags '$(LDFLAGS) -buildid=reproducible/$(VERSION) -X $(PACKAGE)/version.Version=$(VERSION) -X $(PACKAGE)/version.BuildDate=$(DATE) -extldflags -static' \
		-o bin/$(notdir $@) ./$@

.PHONY: identifier-webapp
identifier-webapp:
	$(MAKE) -C identifier build

# Helpers

.PHONY: lint
lint: vendor ; $(info running $(GOLINT) ...)	@
	$(GOLINT) run $(GOLINT_ARGS)
	$(MAKE) -C identifier lint

.PHONY: lint-checkstyle
lint-checkstyle: vendor ; $(info running $(GOLINT) checkstyle ...)     @
	@mkdir -p test
	$(GOLINT) run $(GOLINT_ARGS) --out-format checkstyle --issues-exit-code 0 > test/tests.lint.xml
	$(MAKE) -C identifier lint-checkstyle

.PHONY: fulllint
fulllint: GOLINT_ARGS=
fulllint: lint

.PHONY: fmt
fmt: ; $(info running gofmt ...)	@
	@ret=0 && for d in $$($(GO) list -mod=readonly -f '{{.Dir}}' ./... | grep -v /vendor/); do \
		$(GOFMT) -l -w $$d/*.go || ret=$$? ; \
	done ; exit $$ret

.PHONY: check
check: ; $(info checking dependencies ...) @
	@$(GO) mod verify && echo OK

# Tests

TEST_TARGETS := test-default test-bench test-short test-race test-verbose
.PHONY: $(TEST_TARGETS)
test-bench:   ARGS=-run=_Bench* -test.benchmem -bench=.
test-short:   ARGS=-short
test-race:    ARGS=-race
test-race:    CGO_ENABLED=1
test-verbose: ARGS=-v
$(TEST_TARGETS): NAME=$(MAKECMDGOALS:test-%=%)
$(TEST_TARGETS): test

.PHONY: test
test: ; $(info running $(NAME:%=% )tests ...)	@
	@CGO_ENABLED=1 $(GO) test -timeout $(TIMEOUT)s $(ARGS) $(TESTPKGS)

TEST_XML_TARGETS := test-xml-default test-xml-short test-xml-race
.PHONY: $(TEST_XML_TARGETS)
test-xml-short: ARGS=-short
test-xml-race:  ARGS=-race
test-xml-race:  CGO_ENABLED=1
$(TEST_XML_TARGETS): NAME=$(MAKECMDGOALS:test-%=%)
$(TEST_XML_TARGETS): test-xml

.PHONY: test-xml
test-xml: ; $(info running $(NAME:%=% )tests ...)	@
	@mkdir -p test
	@CGO_ENABLED=1 $(GOTESTSUM) --junitfile test/tests.xml --format standard-verbose \
		-- -timeout $(TIMEOUT)s $(ARGS) $(TESTPKGS)

COVERAGE_PROFILE = $(COVERAGE_DIR)/profile.out
COVERAGE_XML = $(COVERAGE_DIR)/coverage.xml
COVERAGE_HTML = $(COVERAGE_DIR)/coverage.html
.PHONY: test-coverage
test-coverage: COVERAGE_DIR := $(CURDIR)/test/coverage.$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
test-coverage: ; $(info running coverage tests ...)
	@mkdir -p $(COVERAGE_DIR)
	@CGO_ENABLED=1 $(GOTESTSUM) --junitfile test/tests.xml --format standard-verbose \
		-- -timeout $(TIMEOUT)s \
		-coverpkg=./... \
		-covermode=atomic \
		-coverprofile=$(COVERAGE_PROFILE) \
		$(TESTPKGS)
	@$(GO) tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@$(GOCOVERCOBERTURA) < $(COVERAGE_PROFILE) > $(COVERAGE_XML)

# Debug

.PHONY: dlv
dlv: ; $(info attaching Delve debugger ...)
	$(DLV) attach --api-version=$(DLV_APIVERSION) $(DLV_ARGS) $(DLV_ATTACH_PID) $(DLV_EXECUTABLE)

# Mod

.PHONY: go.sum
go.sum: go.mod ; $(info updating dependencies ...)
	@$(GO) mod tidy -v
	@touch $@

.PHONY: vendor
vendor: go.mod ; $(info retrieving dependencies ...)
	@$(GO) mod vendor -v
	@touch $@

# Dist

.PHONY: licenses
licenses: vendor ; $(info building licenses files ...)
	$(CURDIR)/scripts/go-license-ranger.py > $(CURDIR)/3rdparty-LICENSES.md
	make -s -C identifier licenses >> $(CURDIR)/3rdparty-LICENSES.md

3rdparty-LICENSES.md: licenses

# The tarball takes whatever bin/ happens to hold, so packaging asserts there is
# something to ship first. The emptiness test on $(CMDS) carries its own weight:
# an empty list makes the per-command loop pass by never running once.
.PHONY: dist-check
dist-check: ; $(info checking dist contents ...)	@
	@test -n "$(CMDS)" || { echo "dist: no command targets resolved under $(CURDIR)/cmd"; exit 1; }
	@for cmd in $(notdir $(CMDS)); do \
		test -x "bin/$$cmd" || { echo "dist: bin/$$cmd missing or not executable, run make first"; exit 1; }; \
	done

.PHONY: dist
dist: 3rdparty-LICENSES.md | dist-check ; $(info building dist tarball ...)
	@rm -rf "dist/${PACKAGE_NAME}-${VERSION}"
	@mkdir -p "dist/${PACKAGE_NAME}-${VERSION}"
	@mkdir -p "dist/${PACKAGE_NAME}-${VERSION}/scripts"
	@cd dist && \
	cp -avf ../LICENSE.txt "${PACKAGE_NAME}-${VERSION}" && \
	cp -avf ../README.md "${PACKAGE_NAME}-${VERSION}" && \
	cp -avf ../3rdparty-LICENSES.md "${PACKAGE_NAME}-${VERSION}" && \
	cp -avf ../*.yaml.in "${PACKAGE_NAME}-${VERSION}" && \
	cp -avf ../bin/* "${PACKAGE_NAME}-${VERSION}" && \
	cp -avr ../identifier/build "${PACKAGE_NAME}-${VERSION}/identifier-webapp" && \
	cp -avf ../scripts/licod.binscript "${PACKAGE_NAME}-${VERSION}/scripts" && \
	cp -avf ../scripts/licod.service "${PACKAGE_NAME}-${VERSION}/scripts" && \
	cp -avf ../scripts/licod.cfg "${PACKAGE_NAME}-${VERSION}/scripts" && \
	tar --owner=0 --group=0 -czvf ${PACKAGE_NAME}-${VERSION}.tar.gz "${PACKAGE_NAME}-${VERSION}" && \
	cd ..
	@for cmd in $(notdir $(CMDS)); do \
		tar -tzf "dist/${PACKAGE_NAME}-${VERSION}.tar.gz" | \
			grep -qx "${PACKAGE_NAME}-${VERSION}/$$cmd" || \
			{ echo "dist: $$cmd is not in the tarball"; exit 1; }; \
	done

.PHONE: changelog
changelog: ; $(info updating changelog ...)
	$(CHGLOG) --output CHANGELOG.md $(ARGS)

# Rest

.PHONY: clean
clean: ; $(info cleaning ...)	@
	@rm -rf bin
	@rm -rf test/test.*
	@$(MAKE) -C identifier clean

.PHONY: version
version:
	@echo $(VERSION)
