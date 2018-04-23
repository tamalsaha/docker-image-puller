FROM alpine

RUN set -x \
  && apk add --update --no-cache ca-certificates

COPY docker-image-puller /usr/bin/docker-image-puller

ENTRYPOINT ["docker-image-puller"]

