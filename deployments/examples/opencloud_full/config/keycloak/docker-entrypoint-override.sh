#!/bin/bash
printenv
# replace openCloud domain in keycloak realm import
mkdir /opt/keycloak/data/import
sed -e "s/cloud.opencloud.test/${OC_DOMAIN}/g" /opt/keycloak/data/import-dist/opencloud-realm.json > /opt/keycloak/data/import/opencloud-realm.json

# run original docker-entrypoint
/opt/keycloak/bin/kc.sh "$@"
