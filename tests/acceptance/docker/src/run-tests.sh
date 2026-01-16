#!/bin/bash

#mkdir -p /drone/src/vendor-bin/behat
#cp /tmp/vendor-bin/behat/composer.json /drone/src/vendor-bin/behat/composer.json

git config --global advice.detachedHead false

## CONFIGURE TEST
BEHAT_FILTER_TAGS='~@skip'
EXPECTED_FAILURES_FILE=''

if [ "$STORAGE_DRIVER" = "posix" ]; then
    BEHAT_FILTER_TAGS+='&&~@skipOnOpencloud-posix-Storage'
    EXPECTED_FAILURES_FILE='/drone/src/tests/acceptance/expected-failures-on-posix-storage.md'
elif [ "$STORAGE_DRIVER" = "decomposed" ]; then
    BEHAT_FILTER_TAGS+='&&~@skipOnOpencloud-decomposed-Storage'
    EXPECTED_FAILURES_FILE='/drone/src/tests/acceptance/expected-failures-on-decomposed-storage.md'
elif [ "$STORAGE_DRIVER" = "decomposeds3" ]; then
    BEHAT_FILTER_TAGS+='&&~@skipOnOpencloud-decomposeds3-Storage'
    # EXPECTED_FAILURES_FILE='/drone/src/tests/acceptance/expected-failures-on-decomposeds3-storage.md'
fi

export BEHAT_FILTER_TAGS
export EXPECTED_FAILURES_FILE

if [ -n "$BEHAT_FEATURE" ]; then
    export BEHAT_FEATURE
    echo "[INFO] Running feature: $BEHAT_FEATURE"
    # allow running without filters if its a feature
    unset BEHAT_FILTER_TAGS
    unset BEHAT_SUITE
    unset DIVIDE_INTO_NUM_PARTS
    unset RUN_PART
    unset EXPECTED_FAILURES_FILE
elif [ -n "$BEHAT_SUITE" ]; then
    export BEHAT_SUITE
    echo "[INFO] Running suite: $BEHAT_SUITE"
    unset BEHAT_FEATURE
    unset DIVIDE_INTO_NUM_PARTS
    unset RUN_PART
elif [ -n "$DIVIDE_INTO_NUM_PARTS" ] && [ -n "$RUN_PART" ]; then
    export DIVIDE_INTO_NUM_PARTS
    export RUN_PART
    echo "[INFO] Dividing tests into $DIVIDE_INTO_NUM_PARTS parts, running part $RUN_PART"
    unset BEHAT_FEATURE
    unset BEHAT_SUITE
fi

## RUN TEST
sleep 10
make -C "$OC_ROOT" test-acceptance-api

chmod -R 777 vendor-bin/**/vendor vendor-bin/**/composer.lock tests/acceptance/output
