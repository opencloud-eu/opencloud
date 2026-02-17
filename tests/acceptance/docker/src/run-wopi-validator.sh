#!/bin/sh

set -xe

if [ -z "$TEST_GROUP" ]; then
    echo "TEST_GROUP not set"
    exit 1
fi

echo "Waiting for collaboration WOPI endpoint..."

until curl -s http://collaboration:9304 >/dev/null; do
    echo "Waiting for collaboration WOPI endpoint..."
    sleep 2
done

echo "Collaboration is up"

if [ -z "$OC_URL" ]; then
    OC_URL="https://opencloud-server:9200"
fi

curl -vk -X DELETE "$OC_URL/remote.php/webdav/test.wopitest" -u admin:admin   
curl -vk -X PUT "$OC_URL/remote.php/webdav/test.wopitest" -u admin:admin -D headers.txt
cat headers.txt
FILE_ID="$(cat headers.txt | sed -n -e 's/^.*oc-fileid: //Ip')"
export FILE_ID
URL="$OC_URL/app/open?app_name=FakeOffice&file_id=$FILE_ID"
URL="$(echo "$URL" | tr -d '[:cntrl:]')"
export URL
curl -vk -X POST "$URL" -u admin:admin > open.json
cat open.json
cat open.json | jq .form_parameters.access_token | tr -d '"' > accesstoken
cat open.json | jq .form_parameters.access_token_ttl | tr -d '"' > accesstokenttl
WOPI_FILE_ID="$(cat open.json | jq .app_url | sed -n -e 's/^.*files%2F//p' | tr -d '"')"
echo "http://collaboration:9300/wopi/files/$WOPI_FILE_ID" > wopisrc

WOPI_TOKEN=$(cat accesstoken)
export WOPI_TOKEN
WOPI_TTL=$(cat accesstokenttl)
export WOPI_TTL
WOPI_SRC=$(cat wopisrc)
export WOPI_SRC

/app/Microsoft.Office.WopiValidator -s -t "$WOPI_TOKEN" -w "$WOPI_SRC" -l "$WOPI_TTL" --testgroup $TEST_GROUP
