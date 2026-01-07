SHELL := bash

# define standard colors
BLACK        := $(shell tput -Txterm setaf 0)
RED          := $(shell tput -Txterm setaf 1)
GREEN        := $(shell tput -Txterm setaf 2)
YELLOW       := $(shell tput -Txterm setaf 3)
LIGHTPURPLE  := $(shell tput -Txterm setaf 4)
PURPLE       := $(shell tput -Txterm setaf 5)
BLUE         := $(shell tput -Txterm setaf 6)
WHITE        := $(shell tput -Txterm setaf 7)

RESET := $(shell tput -Txterm sgr0)

# add a service here when it uses transifex
L10N_MODULES := \
	services/activitylog \
	services/graph \
	services/notifications \
	services/userlog \
	services/settings

# if you add a module here please also add it to the .drone.star file
OC_MODULES = \
	services/activitylog \
	services/antivirus \
	services/app-provider \
	services/app-registry \
	services/audit \
	services/auth-app \
	services/auth-basic \
	services/auth-bearer \
	services/auth-machine \
	services/auth-service \
	services/clientlog \
	services/collaboration \
	services/eventhistory \
	services/frontend \
	services/gateway \
	services/graph \
	services/groups \
	services/idm \
	services/idp \
	services/invitations \
	services/nats \
	services/notifications \
	services/ocm \
	services/ocs \
	services/policies \
	services/postprocessing \
	services/proxy \
	services/search \
	services/settings \
	services/sharing \
	services/sse \
	services/storage-system \
	services/storage-publiclink \
	services/storage-shares \
	services/storage-users \
	services/thumbnails \
	services/userlog \
	services/users \
	services/web \
	services/webdav\
	services/webfinger\
	opencloud \
	pkg \
	protogen

# bin file definitions
PHP_CS_FIXER=php -d zend.enable_gc=0 vendor-bin/opencloud-codestyle/vendor/bin/php-cs-fixer
PHP_CODESNIFFER=vendor-bin/php_codesniffer/vendor/bin/phpcs
PHP_CODEBEAUTIFIER=vendor-bin/php_codesniffer/vendor/bin/phpcbf
PHAN=php -d zend.enable_gc=0 vendor-bin/phan/vendor/bin/phan
PHPSTAN=php -d zend.enable_gc=0 vendor-bin/phpstan/vendor/bin/phpstan

ifneq (, $(shell command -v go 2> /dev/null)) # suppress `command not found warnings` for non go targets in CI
include .bingo/Variables.mk
endif

.PHONY: help
help:
	@echo "Please use 'make <target>' where <target> is one of the following:"
	@echo
	@echo -e "${GREEN}Testing with test suite natively installed:${RESET}\n"
	@echo -e "${PURPLE}\tdocs: https://opencloud.dev/opencloud/development/testing/#testing-with-test-suite-natively-installed${RESET}\n"
	@echo -e "\tmake test-acceptance-api\t\t${BLUE}run API acceptance tests${RESET}"
	@echo -e "\tmake clean-tests\t\t\t${BLUE}delete API tests framework dependencies${RESET}"
	@echo
	@echo -e "${BLACK}---------------------------------------------------------${RESET}"
	@echo
	@echo -e "${RED}You also should have a look at other available Makefiles:${RESET}"
	@echo
	@echo -e "${GREEN}opencloud:${RESET}\n"
	@echo -e "${PURPLE}\tdocs: https://opencloud.dev/opencloud/development/build/${RESET}\n"
	@echo -e "\tsee ./opencloud/Makefile"
	@echo -e "\tor run ${YELLOW}make -C opencloud help${RESET}"
	@echo
	@echo -e "${GREEN}Documentation:${RESET}\n"
	@echo -e "${PURPLE}\tdocs: https://opencloud.dev/opencloud/development/build-docs/${RESET}\n"
	@echo -e "\tsee ./docs/Makefile"
	@echo -e "\tor run ${YELLOW}make -C docs help${RESET}"
	@echo
	@echo -e "${GREEN}Testing with test suite in docker:${RESET}\n"
	@echo -e "${PURPLE}\tdocs: https://opencloud.dev/opencloud/development/testing/#testing-with-test-suite-in-docker${RESET}\n"
	@echo -e "\tsee ./tests/acceptance/docker/Makefile"
	@echo -e "\tor run ${YELLOW}make -C tests/acceptance/docker help${RESET}"
	@echo
	@echo -e "${GREEN}Tools for developing tests:\n${RESET}"
	@echo -e "\tmake test-php-style\t\t${BLUE}run PHP code style checks${RESET}"
	@echo -e "\tmake test-php-style-fix\t\t${BLUE}run PHP code style checks and fix any issues found${RESET}"
	@echo
	@echo -e "${GREEN}Tools for linting gherkin feature files:\n${RESET}"
	@echo -e "\tmake test-gherkin-lint\t\t${BLUE}run lint checks on Gherkin feature files${RESET}"
	@echo -e "\tmake test-gherkin-lint-fix\t${BLUE}apply lint fixes to gherkin feature files${RESET}"
	@echo

.PHONY: clean-tests
clean-tests:
	@rm -Rf vendor-bin/**/vendor vendor-bin/**/composer.lock tests/acceptance/output

BEHAT_BIN=vendor-bin/behat/vendor/bin/behat

.PHONY: test-acceptance-api
test-acceptance-api: vendor-bin/behat/vendor
	BEHAT_BIN=$(BEHAT_BIN) tests/acceptance/scripts/run.sh

vendor/bamarni/composer-bin-plugin: composer.lock
	composer install

vendor-bin/behat/vendor: vendor/bamarni/composer-bin-plugin vendor-bin/behat/composer.lock
	composer bin behat install --no-progress

vendor-bin/behat/composer.lock: vendor-bin/behat/composer.json
	@echo behat composer.lock is not up to date.
	@rm vendor-bin/behat/composer.lock || true

composer.lock: composer.json
	@echo composer.lock is not up to date.
	@rm composer.lock || true

.PHONY: generate
generate: generate-prod # production is always the default

.PHONY: generate-prod
generate-prod:
	@for mod in $(OC_MODULES); do \
		printf '\n%s:\n---------------------------\n' $$mod; \
        $(MAKE) -C $$mod generate-prod || exit 1; \
    done

.PHONY: generate-dev
generate-dev:
	@for mod in $(OC_MODULES); do \
		printf '\n%s:\n---------------------------\n' $$mod; \
        $(MAKE) -C $$mod generate-dev || exit 1; \
    done

.PHONY: go-generate
go-generate:
	@for mod in $(OC_MODULES); do \
		printf '\n%s:\n---------------------------\n' $$mod; \
        $(MAKE) -C $$mod go-generate || exit 1; \
    done

.PHONY: node-generate-prod
node-generate-prod:
	@for mod in $(OC_MODULES); do \
		printf '\n%s:\n---------------------------\n' $$mod; \
        $(MAKE) -C $$mod node-generate-prod || exit 1; \
    done

.PHONY: node-generate-dev
node-generate-dev:
	@for mod in $(OC_MODULES); do \
		printf '\n%s:\n---------------------------\n' $$mod; \
        $(MAKE) -C $$mod node-generate-dev || exit 1; \
    done

.PHONY: clean
clean:
	@for mod in $(OC_MODULES); do \
        $(MAKE) --no-print-directory -C $$mod clean || exit 1; \
    done

.PHONY: check-env-var-annotations
check-env-var-annotations:
	.make/check-env-var-annotations.sh

.PHONY: go-mod-tidy
go-mod-tidy:
	@for mod in $(OC_MODULES); do \
        $(MAKE) --no-print-directory -C $$mod go-mod-tidy || exit 1; \
    done

.PHONY: test
test:
	@go test -v -tags '$(TAGS)' -coverprofile coverage.out ./...

.PHONY: test-with-race
test-with-race:
	@go test -race -v -tags '$(TAGS)' -coverprofile coverage.out ./...

.PHONY: go-coverage
go-coverage:
	@if [ ! -f coverage.out ]; then $(MAKE) test  &>/dev/null; fi;
	@for mod in $(OC_MODULES); do \
        echo -n "% coverage $$mod: "; $(MAKE) --no-print-directory -C $$mod go-coverage || exit 1; \
    done

.PHONY: protobuf
protobuf:
	@for mod in ./services/thumbnails ./services/settings; do \
        echo -n "% protobuf $$mod: "; $(MAKE) --no-print-directory -C $$mod protobuf || exit 1; \
    done

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --modules-download-mode vendor --timeout 15m0s --issues-exit-code 0 --out-format checkstyle > checkstyle.xml

.PHONY: ci-golangci-lint
ci-golangci-lint:
	$(GOLANGCI_LINT) run --modules-download-mode vendor --timeout 15m0s --issues-exit-code 0 --out-format checkstyle > checkstyle.xml

.PHONY: golangci-lint-fix
golangci-lint-fix: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --fix --modules-download-mode vendor --timeout 15m0s --issues-exit-code 0 --out-format checkstyle > checkstyle.xml

.PHONY: test-gherkin-lint
test-gherkin-lint:
	gherlint tests/acceptance/features -c tests/acceptance/config/.gherlintrc.json

.PHONY: test-gherkin-lint-fix
test-gherkin-lint-fix:
	gherlint --fix tests/acceptance/features -c tests/acceptance/config/.gherlintrc.json

.PHONY: bingo-update
bingo-update: $(BINGO)
	$(BINGO) get -l -v -t 20

.PHONY: check-licenses
check-licenses: $(GO_LICENSES) ci-go-check-licenses ci-node-check-licenses

.PHONY: save-licenses
save-licenses: $(GO_LICENSES) ci-go-save-licenses ci-node-save-licenses

.PHONY: ci-go-check-licenses
ci-go-check-licenses:
	$(GO_LICENSES) check ./...

.PHONY: ci-node-check-licenses
ci-node-check-licenses:
	@for mod in $(OC_MODULES); do \
        echo -e "% check-license $$mod:"; $(MAKE) --no-print-directory -C $$mod ci-node-check-licenses || exit 1; \
    done

.PHONY: ci-go-save-licenses
ci-go-save-licenses:
	@mkdir -p ./third-party-licenses/go/opencloud/third-party-licenses
	$(GO_LICENSES) csv ./... > ./third-party-licenses/go/opencloud/third-party-licenses.csv
	$(GO_LICENSES) save ./... --force --save_path="./third-party-licenses/go/opencloud/third-party-licenses"

.PHONY: ci-node-save-licenses
ci-node-save-licenses:
	@for mod in $(OC_MODULES); do \
        $(MAKE) --no-print-directory -C $$mod ci-node-save-licenses || exit 1; \
    done

CHANGELOG_VERSION =

.PHONY: changelog
changelog: $(CALENS)
ifndef CHANGELOG_VERSION
	$(error CHANGELOG_VERSION is undefined)
endif
	mkdir -p opencloud/dist
	$(CALENS) --version $(CHANGELOG_VERSION) -o opencloud/dist/CHANGELOG.md

.PHONY: changelog-csv
changelog-csv: $(CALENS)
	mkdir -p opencloud/dist
	$(CALENS) -t changelog/changelog-csv.tmpl -o opencloud/dist/changelog.csv

.PHONY: govulncheck
govulncheck: $(GOVULNCHECK)
	$(GOVULNCHECK) ./...

.PHONY: l10n-push
l10n-push:
	@for extension in $(L10N_MODULES); do \
		$(MAKE) -C $$extension l10n-push || exit 1; \
	done

.PHONY: l10n-pull
l10n-pull:
	@for extension in $(L10N_MODULES); do \
		$(MAKE) -C $$extension l10n-pull || exit 1; \
	done

.PHONY: l10n-clean
l10n-clean:
	@for extension in $(L10N_MODULES); do \
		$(MAKE) -C $$extension l10n-clean || exit 1; \
	done

.PHONY: l10n-read
l10n-read:
	@for extension in $(L10N_MODULES); do \
		$(MAKE) -C $$extension l10n-read || exit 1; \
    done

.PHONY: l10n-write
l10n-write:
	@for extension in $(L10N_MODULES); do \
		$(MAKE) -C $$extension l10n-write || exit 1; \
    done

.PHONY: ci-format
ci-format: $(BUILDIFIER)
	$(BUILDIFIER) --mode=fix .woodpecker.star

.PHONY: test-php-style
test-php-style: vendor-bin/opencloud-codestyle/vendor vendor-bin/php_codesniffer/vendor
	$(PHP_CS_FIXER) fix -v --diff --allow-risky yes --dry-run
	$(PHP_CODESNIFFER) --cache --runtime-set ignore_warnings_on_exit --standard=phpcs.xml tests/acceptance tests/acceptance/TestHelpers

.PHONY: test-php-style-fix
test-php-style-fix: vendor-bin/opencloud-codestyle/vendor
	$(PHP_CS_FIXER) fix -v --diff --allow-risky yes
	$(PHP_CODEBEAUTIFIER) --cache --runtime-set ignore_warnings_on_exit --standard=phpcs.xml tests/acceptance

.PHONY: vendor-bin-codestyle
vendor-bin-codestyle: vendor-bin/opencloud-codestyle/vendor

.PHONY: vendor-bin-codesniffer
vendor-bin-codesniffer: vendor-bin/php_codesniffer/vendor

vendor-bin/opencloud-codestyle/vendor: vendor/bamarni/composer-bin-plugin vendor-bin/opencloud-codestyle/composer.lock
	composer bin opencloud-codestyle install --no-progress

vendor-bin/opencloud-codestyle/composer.lock: vendor-bin/opencloud-codestyle/composer.json
	@echo opencloud-codestyle composer.lock is not up to date.

vendor-bin/php_codesniffer/vendor: vendor/bamarni/composer-bin-plugin vendor-bin/php_codesniffer/composer.lock
	composer bin php_codesniffer install --no-progress

vendor-bin/php_codesniffer/composer.lock: vendor-bin/php_codesniffer/composer.json
	@echo php_codesniffer composer.lock is not up to date.

.PHONY: generate-qa-activity-report
generate-qa-activity-report: node_modules
	@if [ -z "${MONTH}" ] || [ -z "${YEAR}" ]; then \
		echo "Please set the MONTH and YEAR environment variables. Usage: make generate-qa-activity-report MONTH=<month> YEAR=<year>"; \
		exit 1; \
	fi
	go run tests/qa-activity-report/generate-qa-activity-report.go --month ${MONTH} --year ${YEAR}
