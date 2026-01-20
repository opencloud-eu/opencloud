#!/bin/bash

# Set required environment variables
export START_TIKA=true
export START_EMAIL=true
export WITH_WRAPPER=true
export STORAGE_DRIVER=${STORAGE_DRIVER:-posix}
export TEST_ROOT_PATH="/drone/src/tests"

# LOCAL TEST WITHOUT EXTRA ENVS
TEST_SERVER_URL="https://opencloud-server:9200"
OC_WRAPPER_URL="http://opencloud-server:5200"

if [ "$STORAGE_DRIVER" = "posix" ]; then
    EXPECTED_FAILURES_FILE="tests/acceptance/expected-failures-posix-storage.md"
else
    EXPECTED_FAILURES_FILE="tests/acceptance/expected-failures-decomposed-storage.md"
fi

# Start server
make -C tests/acceptance/docker start-services

# Wait until the server responds with HTTP 200
echo "Waiting for server to start..."
for i in {1..60}; do
    response_code=$(curl -sk -u admin:admin "${TEST_SERVER_URL}/graph/v1.0/users/admin" -w "%{http_code}" -o /dev/null)
    
    echo "Attempt $i: Received response code $response_code"  # Debugging line to see the status

    if [ "$response_code" == "200" ]; then
        echo "✅ Server is up and running!"
        break
    fi
    sleep 1
done

if [ "$response_code" != "200" ]; then
    echo "❌ Server is not up after 60 attempts."
    exit 1
fi

# List of suites to run
SUITES=(
    "apiArchiver"
    "apiContract"
    "apiCors"
    "apiAsyncUpload"
    "apiDownloads"
    "apiDepthInfinity"
    "apiLocks"
    "apiActivities"
    "apiSettings"
    "apiGraph"
    "apiServiceAvailability"
    "apiGraphUserGroup"
    "apiSpaces"
    "apiSpacesShares"
    "apiSpacesDavOperation"
    "apiSearch1"
    "apiSearch2"
    "apiReshare"
    "apiSharingNg1"
    "apiSharingNg2"
    "apiSharingNgShareInvitation"
    "apiSharingNgLinkSharePermission"
    "apiSharingNgLinkShareRoot"
    "apiSearchContent"
    "apiNotification"
)

# List of suites from core
CORE_SUITES=(
    "coreApiAuth"
    "coreApiCapabilities"
    "coreApiFavorites"
    "coreApiMain"
    "coreApiShareCreateSpecialToShares1"
    "coreApiShareCreateSpecialToShares2"
    "coreApiSharees"
    "coreApiShareManagementBasicToShares"
    "coreApiShareManagementToShares"
    "coreApiShareOperationsToShares1"
    "coreApiShareOperationsToShares2"
    "coreApiSharePublicLink1"
    "coreApiSharePublicLink2"
    "coreApiShareUpdateToShares"
    "coreApiTrashbin"
    "coreApiTrashbinRestore"
    "coreApiVersions"
    "coreApiWebdavDelete"
    "coreApiWebdavEtagPropagation1"
    "coreApiWebdavEtagPropagation2"
    "coreApiWebdavMove1"
    "coreApiWebdavMove2"
    "coreApiWebdavOperations"
    "coreApiWebdavPreviews"
    "coreApiWebdavProperties"
    "coreApiWebdavUpload"
    "coreApiWebdavUploadTUS"
)

EXTRA_SUITE=(
    "apiAntivirus"
    "apiOcm"
    "apiCollaboration"
    "apiAuthApp"
    "cliCommands"
)

# Create log directory
LOG_DIR="./suite-logs"
mkdir -p "$LOG_DIR"

SUCCESS_COUNT=0
FAILURE_COUNT=0

for SUITE in "${SUITES[@]}"; do
    echo "=============================================="
    echo "Running suite: $SUITE"
    echo "=============================================="

    LOG_FILE="$LOG_DIR/${SUITE}.log"

    # Run suite
    make test-acceptance-api TEST_SERVER_URL=$TEST_SERVER_URL OC_WRAPPER_URL=$OC_WRAPPER_URL EXPECTED_FAILURES_FILE=$EXPECTED_FAILURES_FILE BEHAT_SUITE=$SUITE SEND_SCENARIO_LINE_REFERENCES=true > "$LOG_FILE" 2>&1
    
    # Check if suite was successful
    if [ $? -eq 0 ]; then
        echo "✅ Suite $SUITE completed successfully."
        ((SUCCESS_COUNT++))
    else
        echo "❌ Suite $SUITE failed. Check log: $LOG_FILE"
        ((FAILURE_COUNT++))
    fi
done

for SUITE in "${CORE_SUITES[@]}"; do
    echo "=============================================="
    echo "Running suite: $SUITE"
    echo "=============================================="

    LOG_FILE="$LOG_DIR/${SUITE}.log"

    # Run suite
    make test-acceptance-api TEST_SERVER_URL=$TEST_SERVER_URL EXPECTED_FAILURES_FILE=$EXPECTED_FAILURES_FILE BEHAT_SUITE=$SUITE SEND_SCENARIO_LINE_REFERENCES=true > "$LOG_FILE" 2>&1
    
    # Check if suite was successful
    if [ $? -eq 0 ]; then
        echo "✅ Suite $SUITE completed successfully."
        ((SUCCESS_COUNT++))
    else
        echo "❌ Suite $SUITE failed. Check log: $LOG_FILE"
        ((FAILURE_COUNT++))
    fi
done

# Report summary
echo "=============================================="
echo "Test Summary:"
echo "✅ Successful suites: $SUCCESS_COUNT"
echo "❌ Failed suites: $FAILURE_COUNT"
echo "Logs saved in: $LOG_DIR"
echo "=============================================="
