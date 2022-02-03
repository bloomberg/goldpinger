#!/bin/sh

docker login -u "$DOCKER_USER" -p "$DOCKER_PASSWORD" \
    && make build-release
