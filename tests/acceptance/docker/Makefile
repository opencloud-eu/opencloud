.ONESHELL:
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

COMPOSE_PROJECT_NAME := opencloud-acceptance-tests

# Export variables for sub-make calls
export COMPOSE_PROJECT_NAME
export COMPOSE_FILE

export OC_IMAGE_TAG

export STORAGE_DRIVER
export WITH_WRAPPER
export BEHAT_SUITE
export BEHAT_FEATURE

## make definition
.PHONY: help
help:
	@echo "Please use 'make <target>' where <target> is one of the following:"
	@echo
	@echo -e "${PURPLE}docs: https://docs.opencloud.eu/opencloud/development/testing/#testing-with-test-suite-in-docker${RESET}\n"
	@echo
	@echo -e "OpenCloud feature tests and test suites can be found here:"
	@echo -e "\thttps://github.com/opencloud-eu/opencloud/tree/main/tests/acceptance/features"
	@echo
	@echo -e "test suites that test core compatibility are found here and they start with prefix coreApi-:"
	@echo -e "\thttps://github.com/opencloud-eu/opencloud/tree/main/tests/acceptance/features"
	@echo
	@echo -e "The OpenCloud to be tested will be build from your current working state."
	@echo -e "You also can select the OpenCloud Docker image for all tests by setting"
	@echo -e "\tmake ... ${YELLOW}OC_IMAGE_TAG=latest${RESET}"
	@echo -e "where ${YELLOW}latest${RESET} is an example for any valid Docker image tag from"
	@echo -e "https://hub.docker.com/r/opencloud-eu/opencloud."
	@echo
	@echo -e "${GREEN}Run full OpenCloud test suites with decomposed storage:${RESET}\n"
	@echo -e "\tmake localApiTests-apiAccountsHashDifficulty-decomposed\t\t${BLUE}run apiAccountsHashDifficulty test suite,  where available test suite are apiAccountsHashDifficulty apiArchiver apiContract apiGraph apiSpaces apiSpacesShares apiAsyncUpload apiCors${RESET}"
	@echo
	@echo -e "${GREEN}Run full OpenCloud test suites with decomposeds3 storage:${RESET}\n"
	@echo -e "\tmake localApiTests-apiAccountsHashDifficulty-decomposeds3\t\t${BLUE}run apiAccountsHashDifficulty test suite, where available test suite are apiAccountsHashDifficulty apiArchiver apiContract apiGraph apiSpaces apiSpacesShares apiAsyncUpload apiCors${RESET}"
	@echo
	@echo -e "${GREEN}Run full OpenCloud test suites with decomposed storage:${RESET}\n"
	@echo -e "\tmake Core-API-Tests-decomposed-storage-${RED}X${RESET}\t\t${BLUE}run test suite number X, where ${RED}X = 1 .. 10${RESET}"
	@echo
	@echo -e "${GREEN}Run full OpenCloud test suites with decomposeds3 storage:${RESET}\n"
	@echo -e "\tmake Core-API-Tests-decomposeds3-storage-${RED}X${RESET}\t\t${BLUE}run test suite number X, where ${RED}X = 1 .. 10${RESET}"
	@echo
	@echo -e "${GREEN}Run an OpenCloud feature test with decomposed storage:${RESET}\n"
	@echo -e "\tmake test-opencloud-feature-decomposed-storage ${YELLOW}BEHAT_FEATURE='...'${RESET}\t${BLUE}run single feature test${RESET}"
	@echo
	@echo -e "\twhere ${YELLOW}BEHAT_FEATURE='...'${RESET} contains a relative path to the feature definition."
	@echo -e "\texample: ${RED}tests/acceptance/features/apiAccountsHashDifficulty/addUser.feature${RESET}"
	@echo
	@echo -e "${GREEN}Run an OpenCloud feature test with decomposeds3 storage:${RESET}\n"
	@echo -e "\tmake test-opencloud-feature-decomposeds3-storage ${YELLOW}BEHAT_FEATURE='...'${RESET}\t${BLUE}run single feature test${RESET}"
	@echo
	@echo -e "\twhere ${YELLOW}BEHAT_FEATURE='...'${RESET} contains a relative path to the feature definition."
	@echo -e "\texample: ${RED}tests/acceptance/features/apiAccountsHashDifficulty/addUser.feature${RESET}"
	@echo
	@echo -e "\twhere ${YELLOW}BEHAT_FEATURE='...'${RESET} contains a relative path to the feature definition."
	@echo -e "\texample: ${RED}tests/acceptance/features/apiAccountsHashDifficulty/addUser.feature${RESET}"
	@echo
	@echo -e "${GREEN}Run a core test against OpenCloud with decomposed storage:${RESET}\n"
	@echo -e "\tmake test-core-feature-decomposed-storage ${YELLOW}BEHAT_FEATURE='...'${RESET}\t${BLUE}run single feature test${RESET}"
	@echo
	@echo -e "\twhere ${YELLOW}BEHAT_FEATURE='...'${RESET} contains a relative path to the feature definition."
	@echo -e "\texample: ${RED}tests/acceptance/features/coreApiAuth/webDavAuth.feature${RESET}"
	@echo
	@echo -e "${GREEN}Run a core test against OpenCloud with decomposeds3 storage:${RESET}\n"
	@echo -e "\tmake test-core-feature-decomposeds3-storage ${YELLOW}BEHAT_FEATURE='...'${RESET}\t${BLUE}run single feature test${RESET}"
	@echo
	@echo -e "\twhere ${YELLOW}BEHAT_FEATURE='...'${RESET} contains a relative path to the feature definition."
	@echo -e "\texample: ${RED}tests/acceptance/features/coreApiAuth/webDavAuth.feature${RESET}"
	@echo
	@echo -e "\twhere ${YELLOW}BEHAT_FEATURE='...'${RESET} contains a relative path to the feature definition."
	@echo -e "\texample: ${RED}tests/acceptance/features/coreApiAuth/webDavAuth.feature${RESET}"
	@echo
	@echo
	@echo -e "${GREEN}Show output of tests:${RESET}\n"
	@echo -e "\tmake show-test-logs\t\t${BLUE}show output of running or finished tests${RESET}"
	@echo
	@echo
	@echo -e "${GREEN}Clean up after testing:${RESET}\n"
	@echo -e "\tmake clean\t${BLUE}clean up all${RESET}"
	@echo -e "\tmake clean-docker-container\t\t${BLUE}stops and removes used docker containers${RESET}"
	@echo -e "\tmake clean-docker-volumes\t\t${BLUE}removes used docker volumes (used for caching)${RESET}"
	@echo

.PHONY: run-api-tests
run-api-tests: $(OC_WRAPPER) build-dev-image clean-docker-container
	@echo "[INFO] Compose project: $(COMPOSE_PROJECT_NAME)"
	@echo "[INFO] Compose file: $(COMPOSE_FILE)"
	@echo "[INFO] Using storage driver: $(STORAGE_DRIVER)"
	$(MAKE) --no-print-directory start-services
	$(MAKE) --no-print-directory run-test-only

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
	@if [ -z "$(OC_IMAGE_TAG)" ]; then \
		$(MAKE) --no-print-directory -C ../../../opencloud dev-docker \
	; fi;

.PHONY: clean-dev-image
clean-dev-image: ## clean docker image built during acceptance tests
	@docker image rm opencloudeu/opencloud:dev || true

.PHONY: clean-containers
clean-containers: ## clean docker containers created during acceptance tests
	docker compose down --remove-orphans -v

.PHONY: clean-files
clean-files:
	@$(MAKE) --no-print-directory -C ../../../. clean-tests

.PHONY: clean
clean-all: clean-containers clean-dev-image clean-files ## clean all

.PHONY: start-services
start-services: $(OC_WRAPPER) ## start services
	docker compose up -d --build --force-recreate

.PHONY: run-test-only
run-test-only:
	docker compose -f src/acceptance.yml up
