# custom Dockerfile required to run ocwrapper command
# mounting 'ocwrapper' binary doesn't work with image 'amd64/alpine:3.17' (busybox based)

ARG OC_IMAGE_TAG
FROM opencloudeu/opencloud:${OC_IMAGE_TAG} AS opencloud

FROM ubuntu:22.04
COPY --from=opencloud /usr/bin/opencloud /usr/bin/opencloud
RUN apt-get update && apt-get install -y inotify-tools

COPY ["./serve-opencloud.sh", "/usr/bin/serve-opencloud"]
RUN chmod +x /usr/bin/serve-opencloud

ENTRYPOINT [ "serve-opencloud" ]