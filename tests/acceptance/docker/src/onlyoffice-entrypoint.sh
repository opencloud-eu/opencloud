#!/bin/bash

set -e

mkdir -p /var/www/onlyoffice/Data/certs
cd /var/www/onlyoffice/Data/certs
openssl req -x509 -newkey rsa:4096 -keyout onlyoffice.key -out onlyoffice.crt -sha256 -days 365 -batch -nodes
chmod 400 /var/www/onlyoffice/Data/certs/onlyoffice.key

/app/ds/run-document-server.sh
