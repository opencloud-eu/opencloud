#!/bin/bash
source .woodpecker.env

# if no $1 is supplied end the script
# Can be web, acceptance or e2e
if [ -z "$1" ]; then
  echo "No cache item is supplied."
  exit 1
fi

echo "Checking web version - $WEB_COMMITID in cache"

URL="$CACHE_ENDPOINT/$CACHE_BUCKET/opencloud/web-test-runner/$WEB_COMMITID/$1.tar.gz"

echo "Checking for the web cache at '$URL'."

if curl --output /dev/null --silent --head --fail "$URL"
then
	echo "$1 cache with commit id $WEB_COMMITID already available."
	# https://discourse.drone.io/t/how-to-exit-a-pipeline-early-without-failing/3951
	# exit a Pipeline early without failing
	exit 78
else
	echo "$1 cache with commit id $WEB_COMMITID was not available."
fi
