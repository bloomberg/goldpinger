#!/bin/sh

docker login -u "$DOCKER_USER" -p "$DOCKER_PASSWORD" \
    && make tag \
    && make push \
    && make vendor-tag \
    && make vendor-push

