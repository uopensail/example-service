FROM ubuntu:jammy

ARG GIT_HASH=unknown
ARG GIT_TAG=latest
ARG BUILD_TIME=unknown

LABEL org.opencontainers.image.revision="$GIT_HASH" \
      org.opencontainers.image.version="$GIT_TAG" \
      org.opencontainers.image.created="$BUILD_TIME"

RUN mkdir -pv /data/logs/
RUN mkdir -pv /app/conf
COPY dist /app
WORKDIR /app
ENTRYPOINT [ "/app/main" ]