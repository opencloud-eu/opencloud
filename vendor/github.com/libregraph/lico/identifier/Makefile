# Tools

BUN  ?= bun

# --bun matters: package binaries carry a "#!/usr/bin/env node" shebang and
# bun x honours it. bunfig.toml's "[run] bun = true" covers bun run only, and
# only when bun starts in the directory holding that file, which i18n/ is not.
BUNX ?= $(BUN) --bun x

# Variables

VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2>/dev/null | sed 's/^v//' || \
			cat $(CURDIR)/../.version 2> /dev/null || echo 0.0.0-unreleased)

# Build

.PHONY: all
all: build

.PHONY: build
build: vendor | src i18n ; $(info building identifier Webapp ...)	@
	@rm -rf build

	VITE_KOPANO_BUILD="${VERSION}" CI=false $(BUN) --bun run build

.PHONY: src
src:
	@$(MAKE) -C src

.PHONY: i18n
i18n: vendor
	@$(MAKE) -C i18n

.PHONY: lint
lint: vendor ; $(info running biome ...)	@
	@$(BUN) --bun run lint && echo "biome: no lint errors"

.PHONY: lint-checkstyle
lint-checkstyle: vendor ; $(info running biome checkstyle ...)	@
	@mkdir -p ../test
	$(BUNX) biome lint --error-on-warnings --reporter=checkstyle --reporter-file=../test/tests.biome.xml .

# The generated locale JSON is what makes i18n a prerequisite here: both the
# typecheck and the suite import it, and without it they fail with missing
# modules rather than anything meaningful. The suite also renders the app icon,
# which src generates. Linting reads neither.
.PHONY: typecheck
typecheck: vendor | i18n ; $(info running tsc ...)	@
	@$(BUNX) tsc -p tsconfig.json --noEmit && echo "tsc: no type errors"

.PHONY: test
test: vendor | src i18n ; $(info running vitest ...)	@
	@$(BUN) --bun run test --run

# Cheapest gate first, so a lint slip fails in a tenth of a second instead of
# after the suite. Deliberately no production build: this stays quick enough to
# run constantly, and `make build` is there when the output itself matters.
.PHONY: check
check: lint typecheck test

# Bun

.PHONY: vendor
vendor: package.json bun.lock ; $(info getting depdencies with bun ...)   @
	@$(BUN) install --frozen-lockfile

# Stuff

.PHONY: licenses
licenses:
	echo "## LibreGraph Connect identifier web app\n"
	@$(BUN) --bun run licenses

.PHONY: clean ; $(info cleaning identifier Webapp ...)	@
clean:
	$(BUN) pm cache rm
	@rm -rf build
	@rm -rf node_modules

	@$(MAKE) -C src clean

.PHONY: version
version:
	@echo $(VERSION)
