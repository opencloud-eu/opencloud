#!/bin/bash
printenv
# replace openCloud domain and LDAP password in keycloak realm import
mkdir /opt/keycloak/data/import
sed -e "s/cloud.opencloud.test/${OC_DOMAIN}/g" -e "s/ldap-admin-password/${LDAP_ADMIN_PASSWORD:-admin}/g" /opt/keycloak/data/import-dist/openCloud-realm.json > /opt/keycloak/data/import/openCloud-realm.json

# run original docker-entrypoint
/opt/keycloak/bin/kc.sh "$@"
