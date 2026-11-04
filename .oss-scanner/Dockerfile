# Dockerfile OpenCloud for oss-scanner
#

# The IDP login page is a React app that the idp service embeds. Build it in its own stage.
FROM node:24-trixie AS idp
COPY services/idp /idp
WORKDIR /idp
RUN corepack enable \
    && pnpm install --frozen-lockfile \
    && pnpm build \
    && cp src/images/favicon.svg src/images/icon-lilac.svg assets/identifier/static/ \
    && rm -f assets/identifier/static/favicon.ico


FROM golang:1.26-trixie
ENV DEBIAN_FRONTEND=noninteractive \
    GOTOOLCHAIN=local \
    OC_CONFIG_DIR=/src/config \
    OC_BASE_DATA_PATH=/src/data \
    OC_URL=https://localhost:9200 \
    OC_INSECURE=true \
    OC_LOG_LEVEL=warn \
    PROXY_ENABLE_BASIC_AUTH=true \
    IDM_CREATE_DEMO_USERS=true

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates build-essential git make pkg-config \
      curl jq procps \
      && rm -rf /var/lib/apt/lists/*


# scanner contract: the checkout lives inside the image, at /src
COPY --chown=www-data:www-data . /src
WORKDIR /src
COPY --from=idp --chown=www-data:www-data /idp/assets/identifier /src/services/idp/assets/identifier

# Create the config and data directories and set the correct user
RUN mkdir config data && chown www-data config data

# Create a little tooling for starting and stopping and saving a pristine config
RUN printf '%s\n' '#!/bin/sh' 'exec setpriv --reuid=www-data --regid=www-data --init-groups env HOME=/tmp "$@"' \
       > /usr/local/bin/as-www-data && chmod +x /usr/local/bin/as-www-data \
    && printf '%s\n' '#!/bin/sh' \
         'if curl -ks -o /dev/null https://localhost:9200/; then echo "OpenCloud is already running"; exit 0; fi' \
         'nohup /src/opencloud/bin/opencloud server  >>/var/log/opencloud.log 2>&1 &' \
         'for i in $(seq 1 10); do' \
         '  echo "Checking cloud"' \
         '  curl -ksf -o /dev/null -u admin:admin -X PROPFIND https://localhost:9200/remote.php/dav/files/admin/ && exit 0' \
         '  sleep 1' \
         'done' \
         'echo "OpenCloud did not start, see /var/log/opencloud.log" >&2' \
         'exit 1' \
         > /usr/local/bin/opencloud-start \
    && printf '%s\n' '#!/bin/sh' \
         'pkill -f "opencloud server" || exit 0' \
         'while pgrep -f "opencloud server" >/dev/null; do sleep 1; done' \
         > /usr/local/bin/opencloud-stop \
    && printf '%s\n' '#!/bin/sh' 'set -e' \
         'opencloud-stop' \
         'rm -rf ./data ./config' \
         'mkdir -p ./data' \
         'cp -a ./config.pristine ./config' \
         'echo "Reset done. Run opencloud-start to start the server again."' \
         > /usr/local/bin/opencloud-reset \
    && chmod +x /usr/local/bin/opencloud-start /usr/local/bin/opencloud-stop /usr/local/bin/opencloud-reset

# Fetch all modules now; there is no network access after the build.
RUN as-www-data go mod download

# Build opencloud with the native Makefile
RUN as-www-data make -C opencloud build

# Initialize OpenCloud
RUN as-www-data ./opencloud/bin/opencloud init --insecure true --admin-password admin \
    && cp -a ./config ./config.pristine \
    && opencloud-start \
    && sleep 3 \
    && opencloud-reset

# Compile the test binaries once, without running them, so `go test` in the scanner starts from a warm build cache.
RUN as-www-data go test -count=1 -exec /bin/true ./... >/dev/null || echo "WARNING: some test packages do not compile"
