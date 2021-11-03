FROM golang:1.16-alpine as builder

# Install our build tools

RUN apk add --update git make bash

# Get dependencies

WORKDIR /w
COPY go.mod go.sum /w/
RUN go mod download

# Build goldpinger

COPY . ./
RUN make bin/goldpinger

# Build the asset container, copy over goldpinger

FROM scratch
COPY --from=builder /w/bin/goldpinger /goldpinger
COPY ./static /static
ENTRYPOINT ["/goldpinger", "--static-file-path", "/static"]
