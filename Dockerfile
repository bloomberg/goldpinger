FROM golang:1.11-alpine as builder

# Install our build tools

RUN apk add --update git make bash
RUN go get -u github.com/golang/dep/cmd/dep

# Get sources

RUN go get github.com/bloomberg/goldpinger/cmd/goldpinger
WORKDIR /go/src/github.com/bloomberg/goldpinger

# Install our dependencies

RUN make vendor

# Build goldpinger

RUN make bin/goldpinger

# Build the asset container, copy over goldpinger

FROM scratch
COPY --from=builder /go/src/github.com/bloomberg/goldpinger/bin/goldpinger /goldpinger
COPY ./static /static
ENTRYPOINT ["/goldpinger", "--static-file-path", "/static"]

