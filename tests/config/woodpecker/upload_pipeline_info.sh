#!/bin/bash

set -e

CACHE_KEY="$PUBLIC_BUCKET/$CI_REPO_NAME/pipelines/$CI_COMMIT_SHA-$CI_PIPELINE_EVENT"

mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY

# check previous pipeline
URL="https://s3.ci.opencloud.eu/$CACHE_KEY/prev_pipeline"
status=$(curl -s -o prev_pipeline "$URL" -w '%{http_code}')

if [ "$status" == "200" ];
then
    source prev_pipeline
    REPO_ID=$(printf '%s' "$CI_PIPELINE_URL" | sed 's|.*/repos/\([0-9]*\)/.*|\1|')
    p_status=$(curl -s -o pipeline_info.json "$CI_SYSTEM_URL/api/repos/$REPO_ID/pipelines/$PREV_PIPELINE_NUMBER" -w "%{http_code}")
    if [ "$p_status" != "200" ];
    then
        echo -e "[ERROR] Failed to fetch previous pipeline info.\n  URL: $CI_SYSTEM_URL/api/repos/$REPO_ID/pipelines/$PREV_PIPELINE_NUMBER\n  Status: $p_status"
        exit 1
    fi
    # update previous pipeline info
    mc cp -a pipeline_info.json "s3/$CACHE_KEY/"
fi

# upload current pipeline number for the next pipeline
echo "PREV_PIPELINE_NUMBER=$CI_PIPELINE_NUMBER" > prev_pipeline
mc cp -a prev_pipeline "s3/$CACHE_KEY/"
