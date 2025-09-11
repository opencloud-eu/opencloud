# Introduction

The Groupware component of OpenCloud

* is implemented as yet another microservice within the OpenCloud framework (see `./services/groupware/`)
* is essentially providing a REST API to the OpenCloud UI clients (web, mobile) that is high-level and adapted to the needs of the UIs
* the implementation of that REST API turns those high-level APIs into lower-level [JMAP](https://jmap.io/) API calls to [Stalwart, the JMAP mail server](https://stalw.art/), using our own JMAP client library in `./pkg/jmap/`

# Repository

The code lives in the same tree as the other OpenCloud backend services, albeit in the `groupware` branch, that gets rebased on `main` on a regular basis (at least once per week.)

Use [the `groupware` branch](https://github.com/opencloud-eu/opencloud/tree/groupware)


```bash
cd ~/src/opencloud/
git clone --branch groupware git@github.com:opencloud-eu/opencloud.git
```

Also, you might want to check out these [helper scripts in opencloud-tools](https://github.com/pbleser-oc/opencloud-tools) somewhere and put that directory into your `PATH`, as it contains scripts to test and build the OpenCloud Groupware:

```bash
cd ~/src/opencloud/
git clone git@github.com:pbleser-oc/opencloud-tools.git ./bin
echo 'export PATH="$PATH:$HOME/src/opencloud/bin"' >> ~/.bashrc
```

Those scripts have the following prerequisites:
* the [`jq`](https://github.com/jqlang/jq) JSON query command-line tool to extract access tokens,
* either the [httpie](https://httpie.io/cli) (`pipx install httpie`) or [`xh`](https://github.com/ducaale/xh) (`cargo install xh --locked`) command-line HTTP clients, just out of convenience as their output is much nicer than curl's
* `curl` as well, to retrieve the access tokens from Keycloak (no need for nice output there)

# Running

Since we require having a Stalwart container running at the very least, the preferred way of running OpenCloud and its adjacent services for developing the Groupware component is by using the `opencloud_full` Docker Compose setup.

## Configuration

### Hosts

Make sure to have the following entries in your `/etc/hosts`:

```text
127.0.0.1       cloud.opencloud.test
127.0.0.1       keycloak.opencloud.test
127.0.0.1       collabora.opencloud.test
127.0.0.1       wopiserver.opencloud.test
127.0.0.1       mail.opencloud.test
127.0.0.1       collabora.opencloud.test
127.0.0.1       stalwart.opencloud.test
127.0.0.1       traefik.opencloud.test
```

### Compose

It first needs to be tuned a little, and for that, edit `deployments/examples/opencloud_full/.env`, making the following changes:

* change the container image to `opencloudeu/opencloud:dev`:
```diff
-OC_DOCKER_IMAGE=opencloudeu/opencloud-rolling
+OC_DOCKER_IMAGE=opencloudeu/opencloud
-OC_DOCKER_TAG=
+OC_DOCKER_TAG=dev
```

* add the `groupware` service to `START_ADDITIONAL_SERVICES`:
```diff
-START_ADDITIONAL_SERVICES="notifications"
+START_ADDITIONAL_SERVICES="notifications,groupware"
```

* enable the OpenLDAP container:
```diff
-#LDAP=:ldap.yml
+LDAP=:ldap.yml
```

* enable the Keycloak container:
```diff
-#KEYCLOAK=:keycloak.yml
+KEYCLOAK=:keycloak.yml
```

* enable the Stalwart container:
```diff
-#STALWART=:stalwart.yml
+STALWART=:stalwart.yml
```

* optionally disable the Collabora container
```diff
-COLLABORA=:collabora.yml
+#COLLABORA=:collabora.yml
```

* optionally disable UI containers
```diff
-UNZIP=:web_extensions/unzip.yml
-DRAWIO=:web_extensions/drawio.yml
-JSONVIEWER=:web_extensions/jsonviewer.yml
-PROGRESSBARS=:web_extensions/progressbars.yml
-EXTERNALSITES=:web_extensions/externalsites.yml
+#UNZIP=:web_extensions/unzip.yml
+#DRAWIO=:web_extensions/drawio.yml
+#JSONVIEWER=:web_extensions/jsonviewer.yml
+#PROGRESSBARS=:web_extensions/progressbars.yml
+#EXTERNALSITES=:web_extensions/externalsites.yml
```

## Running

Build the `opencloudeu/opencloud:dev` image first:

```bash
cd ~/src/opencloud/opencloud
make -C opencloud/ clean build dev-docker
```

If you see obscure JavaScript related errors, do this and then try the `make` command above again:

```bash
make -C opencloud/services/idp/ generate
```

And then either run everything from the Docker Compose `opencloud_full` setup:

```bash
cd ./deployments/examples/opencloud_full/
docker compose up -d
```

or from within VSCode, in which case you should run all the services from the Docker Compose setup as above, but stop the `opencloud` service container (as that one will be running from within your IDE instead):

```bash
docker stop opencloud_full-opencloud-1
```

and then use the Launcher `OpenCloud server with external services` in VSCode.

## Keycloak Configuration

Now that Keycloak is running, we also need to add a new `groupware` client to the Keycloak `OpenCloud` realm in order to be able to use our command-line scripts and other test components.

To do so, use your preferred web browser and
* head over to <https://keycloak.opencloud.test/>
* authenticate as `admin` with password `admin` (those credentials are defined in the `.env` file mentioned above, see `KEYCLOAK_ADMIN_USER` and `KEYCLOAK_ADMIN_PASSWORD`)
* select the `OpenCloud` realm in the drop-down list in the top left corner (the realm is defined in the `.env` file, see `KEYCLOAK_REALM`)
* then select the "Clients" menu item on the left
* in the "Clients list" tab, push the "Create client" button:
  * Client type: `OpenID Connect`
  * Client ID: `groupware`
* click the "Next" button:
  * Client authentication: Off
  * Authorization: Off
  * Authentication flow: make sure "Direct access grants" is checked
* click the "Next" button and leave the fields there empty to stick to the defaults
* click "Save"

To check whether it works correctly:
```bash
curl -ks -D- -X POST "https://keycloak.opencloud.test/realms/openCloud/protocol/openid-connect/token" -d username=alan -d password=demo -d grant_type=password -d client_id=groupware -d scope=openid
```
should provide you with a JSON response that contains an `access_token`.

If it is not set up correctly, it should give you this instead:
```json
{"error":"invalid_client","error_description":"Invalid client or Invalid client credentials"}
```

## Checking

To check whether the various services are running correctly:

### LDAP

Run the following command on your host (requires the `ldap-tools` package with the `ldapsearch` CLI tool), which should output a list of DNs of demo users:
```bash
ldapsearch -h localhost -D 'cn=admin,dc=opencloud,dc=eu' -x -w 'admin' -b 'ou=users,dc=opencloud,dc=eu' -LLL '(objectClass=person)' dn
```

Sample output:
```text
dn: uid=alan,ou=users,dc=opencloud,dc=eu

dn: uid=lynn,ou=users,dc=opencloud,dc=eu

dn: uid=mary,ou=users,dc=opencloud,dc=eu

dn: uid=admin,ou=users,dc=opencloud,dc=eu

dn: uid=dennis,ou=users,dc=opencloud,dc=eu

dn: uid=margaret,ou=users,dc=opencloud,dc=eu

```

### Stalwart

To then test the IMAP authentication with Stalwart, run the following command on your host (requires the `openssl` CLI tool):

```bash
openssl s_client -crlf -connect localhost:993
```

When then greeted with the following prompt:
```text
* OK [CAPABILITY ...] Stalwart IMAP4rev2 at your service.
```

enter the following command:
```text
A LOGIN alan demo
```

to which one should receive the following response:
```text
A OK [CAPABILITY IMAP4rev2 ...] Authentication successful
```

### Keycloak

As mentioned previously, use the following command on your host to retrieve an access token from Keycloak:

```bash
curl -ks -D- -X POST "https://keycloak.opencloud.test/realms/openCloud/protocol/openid-connect/token" -d username=alan -d password=demo -d grant_type=password -d client_id=groupware -d scope=openid
```

## Feeding an Inbox

Once a [Stalwart](https://stalw.art/) container is running (using the Docker Compose setup as explained above), use [`imap-filler`](https://github.com/opencloud-eu/imap-filler/) to populate the inbox folder via IMAP APPEND:

```bash
cd ~/src/opencloud/
git clone git@github.com:opencloud-eu/imap-filler.git
cd ./imap-filler
go run . --empty=true --username=alan --password=demo \
--url=localhost:993 --folder=Inbox --senders=3 --count=20
```

# Building

If you run the `opencloud` service as a container, use the following script to update the container image and restart it:


```bash
oc-full-update
```

If you run it from your IDE, there is obviously no need to do that.

# API Docs

The REST API documentation is extracted from the source code structure and documentation using [`go-swagger`](https://goswagger.io/go-swagger/), which needs to be installed locally as a prerequisite:


```bash
go install github.com/go-swagger/go-swagger/cmd/swagger@latest
```

The build chain is integrated within the `Makefile` in `services/groupware/`:


```bash
cd services/groupware/
make apidoc-static
```

That creates a static documentation HTML file using [redocly](https://redocly.com/) named `api.html`

```bash
firefox ./api.html
```

Note that `redocly-cli` does not need to be installed, it will be pulled locally by the `Makefile`, provided that you have [pnpm](https://pnpm.io/) installed as a pre-requisite, which is already necessary for other OpenCloud components.

# Testing

This section assumes that you are using the [helper scripts in opencloud-tools](https://github.com/pbleser-oc/opencloud-tools) as instructed above.

If you are running OpenCloud from within VSCode, then make sure to set the following environment variable first, in the shell from which you will use the scripts, as the OpenCloud process is listening to that address as opposed to <https://cloud.opencloud.test> and going through Traefik as is the case when running it from the Docker Compose `opencloud_full` setup:

```bash
export baseurl=https://localhost:9200
```

The scripts default to using the user `alan` (with the password `demo`), which can be changed by setting the following environment variables:
* `username`
* `password`

Your main swiss army knife tool will be `oc-gw` (mnemonic for "OpenCloud Groupware"), which
* always retrieves an access token from Keycloak, using the credentials defined in `username` and `password` (defaulting to `adam`/`demo`), using the "Direct Access Grant" OIDC or "Resource Owner Password Credentials Grant" OAuth2 flow
* and then use that JWT for Bearer authentication against the OpenCloud Groupware REST API

It will also save you some typing as whenever you use `//` for the URL, it will replace that by the Groupware REST API base URL, e.g.

```bash
oc-gw //accounts
```

will be translated into

```bash
http https://cloud.opencloud.test/groupware/accounts
```

The first thing you might want to test is to query the index, which will ensure everything is working properly, including the authentication and the communication between the Groupware and Stalwart:

```bash
oc-gw //
```

# Services

## Stalwart

### Web UI

To access the Stalwart admin UI, open <https://stalwart.opencloud.test/> and use the following credentials to log in:
* username: `mailadmin`
* password: `admin`

The usual admin username `admin` had to be changed into `mailadmin` because there is already an `admin` user that ships with the default users in OpenCloud, and Stalwart always checks the LDAP directory before its internal usernames.

Those credentials are configured in `deployments/examples/opencloud_full/config/stalwart/config.toml`:
```ruby
authentication.fallback-admin.secret = "$6$4qPYDVhaUHkKcY7s$bB6qhcukb9oFNYRIvaDZgbwxrMa2RvF5dumCjkBFdX19lSNqrgKltf3aPrFMuQQKkZpK2YNuQ83hB1B3NiWzj."
authentication.fallback-admin.user = "mailadmin"
```

### Restart from Scratch

To start with a Stalwart container from scratch, removing all the data (including emails):

```bash
cd deployments/examples/opencloud_full
docker compose stop stalwart
docker compose rm stalwart
docker volume rm opencloud_full_stalwart-data opencloud_full_stalwart-logs
docker compose up -d stalwart
```

### Diagnostics

If anything goes wrong, the first thing to check is Stalwart's logs, that are configured on the most verbose level (trace) and should thus provide a lot of insight:

```bash
docker logs -f opencloud_full-stalwart-1
```

## OpenLDAP

The `opencloud_full-ldap-server-1` container exports the ports 389 (LDAP) and 636 (LDAPS) on the host.

To access the LDAP tree:
* Host: `localhost`
* Port: `389`
* Bind DN: `cn=admin,dc=opencloud,dc=eu`
* Password: `admin`
* Base DN: `dc=opencloud,dc=eu`

As an example, to list all the users, using the `ldap-tools` on your host:

```bash
ldapsearch -h localhost -D 'cn=admin,dc=opencloud,dc=eu' -x -w 'admin' -b 'ou=users,dc=opencloud,dc=eu' -LLL '(objectClass=person)'
```

