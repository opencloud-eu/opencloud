.ONESHELL:
SHELL := bash

# define standard colors
BLACK        := $(shell tput -Txterm setaf 0)
RED          := $(shell tput -Txterm setaf 1)
GREEN        := $(shell tput -Txterm setaf 2)
YELLOW       := $(shell tput -Txterm setaf 3)
BLUE         := $(shell tput -Txterm setaf 4)
PURPLE       := $(shell tput -Txterm setaf 5)
CYAN         := $(shell tput -Txterm setaf 6)
WHITE        := $(shell tput -Txterm setaf 7)

RESET := $(shell tput -Txterm sgr0)

COMPOSE_FILE := src/opencloud-base.yml

# run tests with ocwrapper by default
WITH_WRAPPER ?= true
OC_WRAPPER := ../../ocwrapper/bin/ocwrapper

# enable tika for full text extraction
ifeq ($(START_TIKA),true)
	COMPOSE_FILE := src/tika.yml:$(COMPOSE_FILE)
	export SEARCH_EXTRACTOR_TYPE := tika
else
	export SEARCH_EXTRACTOR_TYPE := basic
endif

# enable email server
ifeq ($(START_EMAIL),true)
	COMPOSE_FILE := src/email.yml:$(COMPOSE_FILE)
	export OC_ADD_RUN_SERVICES := notifications
endif

# enable antivirus
ifeq ($(START_ANTIVIRUS),true)
	COMPOSE_FILE := src/antivirus.yml:$(COMPOSE_FILE)
	export OC_ADD_RUN_SERVICES := $(OC_ADD_RUN_SERVICES) antivirus
	export POSTPROCESSING_STEPS := virusscan
endif

# enable wopi services
ifeq ($(ENABLE_WOPI),true)
	COMPOSE_FILE := $(COMPOSE_FILE):src/wopi.yml
endif

# default to posix
STORAGE_DRIVER ?= posix
ifeq ($(STORAGE_DRIVER),posix)
	# posix requires a additional driver config
	COMPOSE_FILE := $(COMPOSE_FILE):src/posix.yml
else ifeq ($(STORAGE_DRIVER),decomposeds3)
	COMPOSE_FILE := src/ceph.yml:$(COMPOSE_FILE)
endif

# use latest as default tag if OC_IMAGE is provided but no tag is set
ifneq ($(strip $(OC_IMAGE)),)
	ifeq ($(strip $(OC_IMAGE_TAG)),)
		OC_IMAGE_TAG := latest
	endif
endif

COMPOSE_PROJECT_NAME := opencloud-acceptance-tests

# Export variables for sub-make calls
export COMPOSE_PROJECT_NAME
export COMPOSE_FILE

export OC_IMAGE
export OC_IMAGE_TAG

# test configurations
export STORAGE_DRIVER
export WITH_WRAPPER
export BEHAT_SUITE
export BEHAT_FEATURE
export TEST_SERVER_URL
export USE_BEARER_TOKEN

## make definition
.PHONY: help
help:
	@echo -e "Test suites: ${CYAN}https://github.com/opencloud-eu/opencloud/tree/main/tests/acceptance/features${RESET}"
	@echo -e "Testing docs: ${CYAN}https://github.com/opencloud-eu/opencloud/tree/main/tests/README.md${RESET}"
	@echo
	@echo "Available commands (targets):"
	@echo -e "  ${GREEN}run-api-tests\t\t${RESET}Build dev image, start services and run the tests."
	@echo -e "  ${GREEN}start-services\t${RESET}Start service containers."
	@echo -e "  ${GREEN}run-test-only\t\t${RESET}Run the tests only."
	@echo -e "  ${GREEN}show-test-logs\t${RESET}Show the test logs."
	@echo -e "  ${GREEN}ps\t\t\t${RESET}Show the running containers."
	@echo
	@echo -e "  ${YELLOW}clean-dev-image\t${RESET}Delete the docker image built during acceptance tests."
	@echo -e "  ${YELLOW}clean-containers\t${RESET}Delete the docker containers and volumes."
	@echo -e "  ${YELLOW}clean-tests\t\t${RESET}Delete test dependencies and results."
	@echo -e "  ${YELLOW}clean-all\t\t${RESET}Clean all resources: images, containers, volumes, test files."
	@echo
	@echo "Available environment variables:"
	@echo -e "  ${PURPLE}OC_IMAGE\t\t${RESET}${CYAN}[image_repo]${RESET} OpenCloud image to use. If provided, the dev image build is skipped."
	@echo -e "  ${PURPLE}OC_IMAGE_TAG\t\t${RESET}${CYAN}[image_tag]${RESET} OpenCloud image tag to use. If provided, the dev image build is skipped."
	@echo -e "  ${PURPLE}WITH_WRAPPER\t\t${RESET}${CYAN}[true|false]${RESET} Start OpenCloud server using ocwrapper. Default: ${YELLOW}true${RESET}"
	@echo -e "  ${PURPLE}STORAGE_DRIVER\t${RESET}${CYAN}[posix|decomposed|decomposeds3]${RESET} Storage driver to use. Default: ${YELLOW}posix${RESET}"
	@echo -e "  ${PURPLE}BEHAT_FEATURE\t\t${RESET}${RESET}${CYAN}[path]${RESET} Path to a feature file. Example: ${YELLOW}tests/acceptance/features/apiGraph/changeRole.feature${RESET}"
	@echo -e "  ${PURPLE}BEHAT_SUITE\t\t${RESET}${RESET}${CYAN}[suite_name]${RESET} Test suite to run. Example: ${YELLOW}apiGraph${RESET}"
	@echo -e "  ${PURPLE}TEST_SERVER_URL\t${RESET}${CYAN}[url]${RESET} URL of the OpenCloud server to test against."
	@echo -e "  ${PURPLE}USE_BEARER_TOKEN\t${RESET}${CYAN}[true|false]${RESET} Use a bearer token for authentication. Default: ${YELLOW}false${RESET}"
	@echo
	@echo -e "Example usage:"
	@echo -e "  ${PURPLE}WITH_WRAPPER${RESET}=${YELLOW}false${RESET} \\"
	@echo -e "  ${PURPLE}STORAGE_DRIVER${RESET}=${YELLOW}posix${RESET} \\"
	@echo -e "  ${PURPLE}BEHAT_FEATURE${RESET}=${YELLOW}tests/acceptance/features/apiGraph/changeRole.feature${RESET} \\"
	@echo -e "  make ${GREEN}run-api-tests${RESET}"

.PHONY: run-api-tests
run-api-tests: $(OC_WRAPPER) build-dev-image clean-containers
	@echo "${BLUE}[INFO]${RESET} Compose project: ${YELLOW}$(COMPOSE_PROJECT_NAME)${RESET}"
	@echo "${BLUE}[INFO]${RESET} Compose file: ${YELLOW}$(COMPOSE_FILE)${RESET}"
	@echo "${BLUE}[INFO]${RESET} Using storage driver: ${YELLOW}$(STORAGE_DRIVER)${RESET}"
	# force use local server when using this command
	export TEST_SERVER_URL=https://opencloud-server:9200
	$(MAKE) --no-print-directory start-services
	$(MAKE) --no-print-directory run-test-only

.PHONY: start-services
start-services: $(OC_WRAPPER) ## start services
	docker compose up -d --build --force-recreate

.PHONY: run-test-only
run-test-only:
	docker compose -f src/acceptance.yml up

.PHONY: show-test-logs
show-test-logs: ## show test logs
	docker compose logs --no-log-prefix -f acceptance-tests | less

.PHONY: ps
ps: ## show running containers
	docker compose ps

$(OC_WRAPPER):
	@if [ "$(WITH_WRAPPER)" == "true" ]; then \
		$(MAKE) --no-print-directory -C ../../ocwrapper build \
	; fi;

.PHONY: build-dev-image
build-dev-image:
	@if [ -z "$(OC_IMAGE)" ] && [ -z "$(OC_IMAGE_TAG)" ]; then \
		$(MAKE) --no-print-directory -C ../../../opencloud dev-docker \
	; fi;

.PHONY: clean-dev-image
clean-dev-image: ## clean docker image built during acceptance tests
	@docker image rm opencloudeu/opencloud:dev || true

.PHONY: clean-containers
clean-containers: ## clean docker containers created during acceptance tests
	docker compose down --remove-orphans -v

.PHONY: clean-tests
clean-tests:
	@$(MAKE) --no-print-directory -C ../../../. clean-tests

.PHONY: clean
clean-all: clean-containers clean-dev-image clean-tests ## clean all
