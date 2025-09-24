#!/bin/bash
printenv

if [ ! -f /opt/bitnami/openldap/share/openldap.key ]
then	
	openssl req -x509 -newkey rsa:4096 -keyout /opt/bitnami/openldap/share/openldap.key -out /opt/bitnami/openldap/share/openldap.crt -sha256 -days 365 -batch -nodes
fi
# run original docker-entrypoint
/opt/bitnami/scripts/openldap/entrypoint.sh "$@"
