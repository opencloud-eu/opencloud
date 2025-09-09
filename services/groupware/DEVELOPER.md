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

# Running

Either run everything from the Docker Compose `opencloud_full` setup:


```bash
cd deployments/examples/opencloud_full/
docker compose up -d
```

or from within VSCode, in which case you should run all the services from the Docker Compose setup as above, but stop the `opencloud` service container (as that one will be running from within your IDE instead):


```bash
docker stop opencloud_full-opencloud-1
```

and then use the Launcher `OpenCloud server with external services` in VSCode.

To do the latter on a more permanent basis, comment the following line in `deployments/examples/opencloud_full/.env`:


```yaml
#OPENCLOUD=:opencloud.yml
```

## Feeding an Inbox

Once a Stalwart container is running (using the Docker Compose setup as explained above), use [`imap-filler`](https://github.com/opencloud-eu/imap-filler/):


```bash
cd ~/src/opencloud/
git clone git@github.com:opencloud-eu/imap-filler.git
cd ./imap-filler
EMPTY=true SENDERS=3 \
USERNAME=alan PASSWORD=demo \
URL=localhost:993 FOLDER=Inbox COUNT=20 \
go run .
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


