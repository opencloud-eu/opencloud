#!/bin/bash

mkdir -p "${OC_ROOT}/vendor-bin/behat"
if [ ! -f "${OC_ROOT}/vendor-bin/behat/composer.json" ]; then
    cp /tmp/vendor-bin/behat/composer.json "${OC_ROOT}/vendor-bin/behat/composer.json"
fi

git config --global advice.detachedHead false

## CONFIGURE TEST
BEHAT_FILTER_TAGS='~@skip'
EXPECTED_FAILURES_FILE=''

if [ "$STORAGE_DRIVER" = "posix" ]; then
    BEHAT_FILTER_TAGS+='&&~@skipOnOpencloud-posix-Storage'
    EXPECTED_FAILURES_FILE="${OC_ROOT}/tests/acceptance/expected-failures-posix-storage.md"
elif [ "$STORAGE_DRIVER" = "decomposed" ]; then
    BEHAT_FILTER_TAGS+='&&~@skipOnOpencloud-decomposed-Storage'
    EXPECTED_FAILURES_FILE="${OC_ROOT}/tests/acceptance/expected-failures-decomposed-storage.md"
fi

export BEHAT_FILTER_TAGS
export EXPECTED_FAILURES_FILE

if [ -n "$BEHAT_FEATURE" ]; then
    export BEHAT_FEATURE
    echo "[INFO] Running feature: $BEHAT_FEATURE"
    # allow running without filters if its a feature
    unset BEHAT_FILTER_TAGS
    unset BEHAT_SUITE
    unset EXPECTED_FAILURES_FILE
elif [ -n "$BEHAT_SUITE" ]; then
    export BEHAT_SUITE
    echo "[INFO] Running suite: $BEHAT_SUITE"
    unset BEHAT_FEATURE
fi

## RUN TEST
sleep 10
make -C "$OC_ROOT" test-acceptance-api

chmod -R 777 "${OC_ROOT}/vendor-bin/"*"/vendor" "${OC_ROOT}/vendor-bin/"*"/composer.lock" "${OC_ROOT}/tests/acceptance/output" 2>/dev/null || true
