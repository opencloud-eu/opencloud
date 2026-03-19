# Please use this Dockerfile only if
# you want to build an image from source without
# pnpm and Go installed on your dev machine.

# You can build OpenCloud using this Dockerfile
# by running following command:
# `docker build -t opencloud/opencloud:custom .`

# In most other cases you might want to run the
# following command instead:
# `make -C opencloud dev-docker`
# It will build a `opencloud/opencloud:dev` image for you
# and use your local pnpm and Go caches and therefore
# is a lot faster than the build steps below.


FROM owncloudci/nodejs:22 AS generate

COPY ./ /opencloud/

WORKDIR /opencloud/opencloud
RUN make node-generate-prod

FROM golang:1.24-alpine AS build
RUN apk add bash make git curl gcc musl-dev libc-dev binutils-gold inotify-tools vips-dev

COPY --from=generate /opencloud /opencloud

WORKDIR /opencloud/opencloud
RUN make go-generate build ENABLE_VIPS=true

FROM alpine:3.20

RUN apk add --no-cache attr ca-certificates curl mailcap tree vips && \
	echo 'hosts: files dns' >| /etc/nsswitch.conf

LABEL maintainer="OpenCloud GmbH <devops@opencloud.eu>" \
        org.opencontainers.image.title="OpenCloud" \
        org.opencontainers.image.vendor="OpenCloud GmbH" \
        org.opencontainers.image.authors="OpenCloud GmbH" \
        org.opencontainers.image.description="OpenCloud is a modern file-sync and share platform" \
        org.opencontainers.image.licenses="Apache-2.0" \
        org.opencontainers.image.documentation="https://github.com/opencloud-eu/opencloud" \
        org.opencontainers.image.source="https://github.com/opencloud-eu/opencloud"

ENTRYPOINT ["/usr/bin/opencloud"]
CMD ["server"]

COPY --from=build /opencloud/opencloud/bin/opencloud /usr/bin/opencloud
