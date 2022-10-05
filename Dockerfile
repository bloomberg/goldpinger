FROM golang:1.15-alpine as builder
ARG TARGETARCH
ARG TARGETOS
ENV GO111MODULE=on

# Install our build tools

RUN apk add --update git make bash

# Get dependencies

WORKDIR /w
COPY go.mod go.sum ./
RUN go mod download

# Build goldpinger
COPY . ./
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make bin/goldpinger
# Create vendor folder
RUN go mod vendor

# Build the asset container, copy over goldpinger
FROM scratch as simple
COPY --from=builder /w/bin/goldpinger /goldpinger
COPY ./static /static
COPY ./config /config
ENTRYPOINT ["/goldpinger", "--static-file-path", "/static"]

# For vendor builds, use the simple build and add the vendor'd files
FROM simple as vendor
COPY --from=builder /w/vendor /goldpinger-vendor-sources
