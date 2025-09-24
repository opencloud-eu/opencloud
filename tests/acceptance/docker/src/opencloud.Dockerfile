ARG OC_IMAGE_TAG
FROM opencloudeu/opencloud:${OC_IMAGE_TAG}

USER root

COPY ["./serve-opencloud.sh", "/usr/bin/serve-opencloud"]
RUN chmod +x /usr/bin/serve-opencloud

USER 1000

ENTRYPOINT [ "serve-opencloud" ]
