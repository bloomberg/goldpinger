name ?= goldpinger
version ?= v3.4.0
bin ?= goldpinger
pkg ?= "github.com/bloomberg/goldpinger"
tag = $(name):$(version)
goos ?= ${GOOS}
goarch ?= ${GOARCH}
namespace ?= "bloomberg/"
files = $(shell find . -iname "*.go")


bin/$(bin): $(files)
	GOOS=${goos} PKG=${pkg} ARCH=${goarch} VERSION=${version} BIN=${bin} ./build/build.sh

clean:
	rm -rf ./vendor
	rm -f ./bin/$(bin)

vendor:
	rm -rf ./vendor
	go mod vendor

# Download the latest swagger releases from: https://github.com/go-swagger/go-swagger/releases/
swagger:
	swagger generate server -t pkg -f ./swagger.yml --exclude-main -A goldpinger && \
	swagger generate client -t pkg -f ./swagger.yml -A goldpinger

build-multistage: build

build-release:
	docker buildx build --push --platform linux/amd64,linux/arm64 -t $(namespace)$(tag) --build-arg GO_MOD_ACTION=download --target simple -f ./Dockerfile .
	docker buildx build --push --platform linux/amd64,linux/arm64 -t $(namespace)$(tag)-vendor --build-arg GO_MOD_ACTION=vendor --target vendor -f ./Dockerfile .

build:
	docker build -t $(namespace)$(tag) --build-arg GO_MOD_ACTION=download --target simple -f ./Dockerfile .

run:
	go run ./cmd/goldpinger/main.go

version:
	@echo $(namespace)$(tag)

vendor-build:
	docker build -t $(namespace)$(tag)-vendor --build-arg GO_MOD_ACTION=vendor --target vendor -f ./Dockerfile .

.PHONY: clean vendor swagger build build-multistage build-release vendor-build run version
