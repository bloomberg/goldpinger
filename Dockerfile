FROM golang:1.11-alpine as builder

# Install our build tools

RUN apk add --update git make bash
RUN go get -u github.com/golang/dep/cmd/dep

# Get dependencies

WORKDIR /go/src/github.com/bloomberg/goldpinger
COPY Gopkg.toml Gopkg.lock Makefile ./
RUN make vendor

# Build goldpinger

COPY . ./
RUN make bin/goldpinger

# Build the asset container, copy over goldpinger

FROM scratch
COPY --from=builder /go/src/github.com/bloomberg/goldpinger/bin/goldpinger /goldpinger
COPY ./static /static
ENTRYPOINT ["/goldpinger", "--static-file-path", "/static"]