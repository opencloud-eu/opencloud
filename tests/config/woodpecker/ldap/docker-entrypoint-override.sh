#!/bin/bash
printenv

if [ ! -f /opt/bitnami/openldap/share/openldap.key ]
then
	openssl req -x509 -newkey rsa:4096 -keyout /opt/bitnami/openldap/share/openldap.key -out /opt/bitnami/openldap/share/openldap.crt -sha256 -days 365 -batch -nodes
fi

# the LDIF files in LDAP_CUSTOM_LDIF_DIR are imported by the bitnami setup
# before slapd starts listening, so clients never see a partially seeded tree
exec /opt/bitnami/scripts/openldap/entrypoint.sh "$@"
