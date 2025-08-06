#!/bin/bash
printenv

if [ ! -f /opt/bitnami/openldap/share/openldap.key ]
then	
	openssl req -x509 -newkey rsa:4096 -keyout /opt/bitnami/openldap/share/openldap.key -out /opt/bitnami/openldap/share/openldap.crt -sha256 -days 365 -batch -nodes
fi

mkdir -p /opt/bitnami/openldap/ldifs

if [ -d "/tmp/ldif-files" ]; then
    cp /tmp/ldif-files/*.ldif /opt/bitnami/openldap/ldifs/
fi

/opt/bitnami/scripts/openldap/entrypoint.sh "$@" &
ENTRYPOINT_PID=$!

echo "Waiting for LDAP server to start..."
while ! ldapsearch -x -H ldap://localhost:1389 -D "cn=admin,dc=opencloud,dc=eu" -w admin -b "dc=opencloud,dc=eu" > /dev/null 2>&1; do
    sleep 2
done

echo "LDAP server is running, importing LDIF files..."

if [ -f "/opt/bitnami/openldap/ldifs/10_base.ldif" ]; then
    echo "Importing 10_base.ldif..."
    ldapadd -x -H ldap://localhost:1389 -D "cn=admin,dc=opencloud,dc=eu" -w admin -f /opt/bitnami/openldap/ldifs/10_base.ldif
fi

if [ -f "/opt/bitnami/openldap/ldifs/20_users.ldif" ]; then
    echo "Importing 20_users.ldif..."
    ldapadd -x -H ldap://localhost:1389 -D "cn=admin,dc=opencloud,dc=eu" -w admin -f /opt/bitnami/openldap/ldifs/20_users.ldif
fi

if [ -f "/opt/bitnami/openldap/ldifs/30_groups.ldif" ]; then
    echo "Importing 30_groups.ldif..."
    ldapadd -x -H ldap://localhost:1389 -D "cn=admin,dc=opencloud,dc=eu" -w admin -f /opt/bitnami/openldap/ldifs/30_groups.ldif
fi

echo "LDIF import completed!"

wait $ENTRYPOINT_PID
