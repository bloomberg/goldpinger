#!/bin/bash

# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# MODIFIED FROM https://github.com/thockin/go-build-template/blob/master/build/build.sh

set -o errexit
set -o nounset
set -o pipefail

if [ -z "${PKG}" ]; then
    echo "PKG must be set"
    exit 1
fi
if [ -z "${VERSION}" ]; then
    echo "VERSION must be set"
    exit 1
fi

export CGO_ENABLED=0
export GOARCH="${ARCH:-amd64}"
export GOOS=${GOOS:-linux}

go build \
    -ldflags "-X 'main.Version=${VERSION}' -X 'main.Build=`date`'" \
    -o ./bin/${BIN} \
    ./cmd/${BIN}
