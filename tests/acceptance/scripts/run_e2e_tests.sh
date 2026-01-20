#!/bin/bash

# Set required environment variables
export START_TIKA=true
export WITH_WRAPPER=false

TEST_SERVER_URL="https://opencloud-server:9200"

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


E2E_SUITES=(
    "admin-settings"
    "file-action"
    "journeys"
    "navigation"
    "search"
    "shares"
    "spaces"
    "user-settings"
)

EXTRA_E2E_SUITE="app-providerapp-store,keycloak,ocm,oidc"

# Create log directory
LOG_DIR="./suite-logs"
mkdir -p "$LOG_DIR"

SUCCESS_COUNT=0
FAILURE_COUNT=0

# Clone the repository and install dependencies
git clone https://github.com/opencloud-eu/web
cd web || exit 1
pnpm i
echo "Installation complete, moving to tests/e2e directory..."

# Run e2e suites
for SUITE in "${E2E_SUITES[@]}"; do
    echo "=============================================="
    echo "Running e2e suite: $SUITE"
    echo "=============================================="

    LOG_FILE="$LOG_DIR/${SUITE}.log"

    # Run suite
    (
        cd tests/e2e || exit 1
        OC_BASE_URL=$TEST_SERVER_URL RETRY=1 HEADLESS=true PARALLEL=4 ./run-e2e.sh --suites $SUITE > "../../../$LOG_FILE" 2>&1
    )
    
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

# Cleanup: Remove the cloned web directory
echo "🧹 Cleaning up..."
cd ..
rm -rf web
echo "✅ Cleanup complete."
