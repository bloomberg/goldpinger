FROM golang:1.21 as builder
ARG TARGETARCH
ARG TARGETOS

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
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /w/bin/goldpinger /goldpinger
COPY ./static /static
COPY ./config /config
ENTRYPOINT ["/goldpinger", "--static-file-path", "/static"]

# For vendor builds, use the simple build and add the vendor'd files
FROM simple as vendor
COPY --from=builder /w/vendor /goldpinger-vendor-sources
