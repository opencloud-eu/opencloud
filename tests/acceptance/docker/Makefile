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

## default values only for sub-make calls
ifeq ($(LOCAL_TEST),true)
	COMPOSE_FILE ?= src/opencloud-base.yml:src/tika.yml
	ifeq ($(START_EMAIL),true)
		COMPOSE_FILE := $(COMPOSE_FILE):src/email.yml
	endif
else 
	COMPOSE_FILE ?= src/redis.yml:src/opencloud-base.yml:src/acceptance.yml
endif

## user input
BEHAT_FEATURE ?=
TEST_SOURCE ?= opencloud

ifdef OC_IMAGE_TAG
	BUILD_DEV_IMAGE := 0
else
	BUILD_DEV_IMAGE := 1
endif
OC_IMAGE_TAG ?= dev

# run tests with ocwrapper by default
WITH_WRAPPER ?= true
OC_WRAPPER := ../../ocwrapper/bin/ocwrapper

ifdef START_TIKA
	ifeq ($(START_TIKA),true)
		COMPOSE_FILE := $(COMPOSE_FILE):src/tika.yml
		SEARCH_EXTRACTOR_TYPE := tika
	else
		SEARCH_EXTRACTOR_TYPE := basic
	endif
else
	SEARCH_EXTRACTOR_TYPE := basic
endif

# default to posix
STORAGE_DRIVER ?= posix
ifeq ($(STORAGE_DRIVER),posix)
	# posix requires a additional driver config
	COMPOSE_FILE := $(COMPOSE_FILE):src/posix.yml
endif

# static
DIVIDE_INTO_NUM_PARTS := 10
PARTS = 1 2 3 4 5 6 7 8 9 10
LOCAL_API_SUITES = $(shell ls ../features | grep ^api*)

COMPOSE_PROJECT_NAME := opencloud-acceptance-tests

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
.PHONY: test-opencloud-feature-decomposed-storage
test-opencloud-feature-decomposed-storage: ## test a OpenCloud feature with decomposed storage, usage: make ... BEHAT_FEATURE='tests/acceptance/features/apiAccountsHashDifficulty/addUser.feature:10'
	@TEST_SOURCE=opencloud \
	STORAGE_DRIVER=decomposed \
	BEHAT_FEATURE=$(BEHAT_FEATURE) \
	$(MAKE) --no-print-directory testSuite

.PHONY: test-opencloud-feature-decomposeds3-storage
test-opencloud-feature-decomposeds3-storage: ## test a OpenCloud feature with decomposeds3 storage, usage: make ... BEHAT_FEATURE='tests/acceptance/features/apiAccountsHashDifficulty/addUser.feature:10'
	@TEST_SOURCE=opencloud \
	STORAGE_DRIVER=decomposeds3 \
	BEHAT_FEATURE=$(BEHAT_FEATURE) \
	START_CEPH=1 \
	$(MAKE) --no-print-directory testSuite

.PHONY: test-opencloud-feature-posix-storage
test-opencloud-feature-posix-storage: ## test a OpenCloud feature with posix storage, usage: make ... BEHAT_FEATURE='tests/acceptance/features/apiAccountsHashDifficulty/addUser.feature:10'
	@TEST_SOURCE=opencloud \
	STORAGE_DRIVER=posix \
	BEHAT_FEATURE=$(BEHAT_FEATURE) \
	$(MAKE) --no-print-directory testSuite

.PHONY: test-core-feature-decomposed-storage
test-core-feature-decomposed-storage: ## test a core feature with decomposed storage, usage: make ... BEHAT_FEATURE='tests/acceptance/features/coreApiAuth/webDavAuth.feature'
	@TEST_SOURCE=core \
	STORAGE_DRIVER=decomposed \
	BEHAT_FEATURE=$(BEHAT_FEATURE) \
	$(MAKE) --no-print-directory testSuite

.PHONY: test-core-feature-decomposeds3-storage
test-core-feature-decomposeds3-storage: ## test a core feature with decomposeds3 storage, usage: make ... BEHAT_FEATURE='tests/acceptance/features/coreApiAuth/webDavAuth.feature'
	@TEST_SOURCE=core \
	STORAGE_DRIVER=decomposeds3 \
	BEHAT_FEATURE=$(BEHAT_FEATURE) \
	START_CEPH=1 \
	$(MAKE) --no-print-directory testSuite

.PHONY: test-opencloud-feature-posix-storage
test-core-opencloud-feature-posix-storage: ## test a core feature with posix storage, usage: make ... BEHAT_FEATURE='tests/acceptance/features/apiAccountsHashDifficulty/addUser.feature:10'
	@TEST_SOURCE=core \
	STORAGE_DRIVER=posix \
	BEHAT_FEATURE=$(BEHAT_FEATURE) \
	$(MAKE) --no-print-directory testSuite

localSuiteOpencloud = $(addprefix localApiTests-, $(addsuffix -decomposed,${LOCAL_API_SUITES}))
.PHONY: $(localSuiteOpencloud)
$(localSuiteOpencloud): ## run local api test suite with decomposed storage
	@$(eval BEHAT_SUITE=$(shell echo "$@" | cut -d'-' -f2))
	@TEST_SOURCE=opencloud \
	STORAGE_DRIVER=decomposed \
	BEHAT_SUITE=$(BEHAT_SUITE) \
	$(MAKE) --no-print-directory testSuite

localSuiteDecomposedS3 = $(addprefix localApiTests-, $(addsuffix -decomposeds3,${LOCAL_API_SUITES}))
.PHONY: $(localSuiteDecomposedS3)
$(localSuiteDecomposedS3): ## run local api test suite with s3 storage
	@$(eval BEHAT_SUITE=$(shell echo "$@" | cut -d'-' -f2))
	@TEST_SOURCE=opencloud \
	STORAGE_DRIVER=decomposeds3 \
	BEHAT_SUITE=$(BEHAT_SUITE) \
	$(MAKE) --no-print-directory testSuite

localSuitePosix = $(addprefix localApiTests-, $(addsuffix -posix,${LOCAL_API_SUITES}))
.PHONY: $(localSuitePosix)
$(localSuitePosix): ## run local api test suite with posix storage
	@$(eval BEHAT_SUITE=$(shell echo "$@" | cut -d'-' -f2))
	@TEST_SOURCE=opencloud \
	STORAGE_DRIVER=posix \
	BEHAT_SUITE=$(BEHAT_SUITE) \
	$(MAKE) --no-print-directory testSuite

targetsOC = $(addprefix Core-API-Tests-decomposed-storage-,$(PARTS))
.PHONY: $(targetsOC)
$(targetsOC):
	@$(eval RUN_PART=$(shell echo "$@" | tr -dc '0-9'))
	@TEST_SOURCE=core \
	STORAGE_DRIVER=decomposed \
	RUN_PART=$(RUN_PART) \
	$(MAKE) --no-print-directory testSuite

targetsDecomposedS3 = $(addprefix Core-API-Tests-decomposeds3-storage-,$(PARTS))
.PHONY: $(targetsDecomposedS3)
$(targets):
	@$(eval RUN_PART=$(shell echo "$@" | tr -dc '0-9'))
	@TEST_SOURCE=core \
	STORAGE_DRIVER=decomposeds3 \
	RUN_PART=$(RUN_PART) \
	$(MAKE) --no-print-directory testSuite

.PHONY: testSuite
testSuite: $(OC_WRAPPER) build-dev-image clean-docker-container
	@COMPOSE_FILE_VAR="$(COMPOSE_FILE)"; \
	\
	if [ "${STORAGE_DRIVER}" = "decomposeds3" ]; then \
		COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
		docker compose -f src/ceph.yml run start_ceph; \
	fi; \
	if [ "${START_EMAIL}" = "true" ]; then \
		COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
		docker compose -f src/email.yml run start_email; \
	fi; \
	if [ "${START_ANTIVIRUS}" = "true" ]; then \
		COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
		docker compose -f src/antivirus.yml run start_antivirus; \
	fi; \
	if [ "${START_TIKA}" = "true" ]; then \
		COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
		docker compose -f src/tika.yml run tika-service; \
	fi; \
	if [ "${ENABLE_WOPI}" = "true" ]; then \
		COMPOSE_FILE_VAR="$${COMPOSE_FILE_VAR}:src/wopi.yml"; \
	fi; \
	\
	
	COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
	COMPOSE_FILE=$${COMPOSE_FILE_VAR} \
	STORAGE_DRIVER=$(STORAGE_DRIVER) \
	TEST_SOURCE=$(TEST_SOURCE) \
	WITH_WRAPPER=$(WITH_WRAPPER) \
	OC_ASYNC_UPLOADS=$(OC_ASYNC_UPLOADS) \
	OC_ADD_RUN_SERVICES=$(OC_ADD_RUN_SERVICES) \
	POSTPROCESSING_STEPS=$(POSTPROCESSING_STEPS) \
	SEARCH_EXTRACTOR_TYPE=$(SEARCH_EXTRACTOR_TYPE) \
	OC_IMAGE_TAG=$(OC_IMAGE_TAG) \
	BEHAT_SUITE=$(BEHAT_SUITE) \
	BEHAT_FEATURE=$(BEHAT_FEATURE) \
	DIVIDE_INTO_NUM_PARTS=$(DIVIDE_INTO_NUM_PARTS) \
	RUN_PART=$(RUN_PART) \
	docker compose up -d --build --force-recreate

.PHONY: show-test-logs
show-test-logs: ## show logs of test
	@COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
	docker compose logs --no-log-prefix -f acceptance-tests | less

.PHONY: ps
ps: ## show docker status
	@COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
	docker compose ps

$(OC_WRAPPER):
	@if [ "$(WITH_WRAPPER)" == "true" ]; then \
		$(MAKE) --no-print-directory -C ../../ocwrapper build \
	; fi;

.PHONY: build-dev-image
build-dev-image:
	@if [ $(BUILD_DEV_IMAGE) -eq 1 ]; then \
		$(MAKE) --no-print-directory -C ../../../opencloud dev-docker \
	; fi;

.PHONY: clean-dev-docker-image
clean-dev-docker-image: ## clean docker image built during acceptance tests
	@docker image rm opencloudeu/opencloud:dev || true

.PHONY: clean-docker-container
clean-docker-container: ## clean docker containers created during acceptance tests
	@COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
	COMPOSE_FILE=$(COMPOSE_FILE) \
	BEHAT_SUITE="" \
	DIVIDE_INTO_NUM_PARTS="" \
	OC_IMAGE_TAG="" \
	RUN_PART="" \
	STORAGE_DRIVER="" \
	TEST_SOURCE="" \
	docker compose down --remove-orphans

.PHONY: clean-docker-volumes
clean-docker-volumes: ## clean docker volumes created during acceptance tests
	@COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
	COMPOSE_FILE=$(COMPOSE_FILE) \
	BEHAT_SUITE="" \
	DIVIDE_INTO_NUM_PARTS="" \
	OC_IMAGE_TAG="" \
	RUN_PART="" \
	STORAGE_DRIVER="" \
	TEST_SOURCE="" \
	docker compose down --remove-orphans -v

.PHONY: clean-files
clean-files:
	@$(MAKE) --no-print-directory -C ../../../. clean-tests

.PHONY: clean
clean: clean-docker-container clean-docker-volumes clean-dev-docker-image clean-files ## clean all

.PHONY: start-server
start-server: $(OC_WRAPPER) ## build and start server
	@echo "Build and start server..."
	COMPOSE_FILE=$(COMPOSE_FILE) \
	COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
	OC_IMAGE_TAG=dev \
	WITH_WRAPPER=$(WITH_WRAPPER) \
	TEST_SOURCE=opencloud \
	STORAGE_DRIVER=$(STORAGE_DRIVER) \
	OC_ASYNC_UPLOADS=true \
	SEARCH_EXTRACTOR_TYPE=$(SEARCH_EXTRACTOR_TYPE) \
	OC_ADD_RUN_SERVICES=notifications \
	docker compose up -d --build --force-recreate

.PHONY: run-test
run-test:
	COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
	STORAGE_DRIVER=$(STORAGE_DRIVER) \
	TEST_SOURCE=$(TEST_SOURCE) \
	BEHAT_SUITE=$(BEHAT_SUITE) \
	BEHAT_FEATURE=$(BEHAT_FEATURE) \
	DIVIDE_INTO_NUM_PARTS=$(DIVIDE_INTO_NUM_PARTS) \
	RUN_PART=$(RUN_PART) \
	docker compose -f src/acceptance.yml up
