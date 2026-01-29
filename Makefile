name ?= goldpinger
version ?= v3.10.3
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

run:
	go run ./cmd/goldpinger/main.go

build:
	docker build -t $(namespace)$(tag) --target simple -f ./Dockerfile .

build-vendor:
	docker build -t $(namespace)$(tag)-vendor --target vendor -f ./Dockerfile .

build-release:
	docker buildx build --push --platform linux/amd64,linux/arm64 --target simple -t $(namespace)$(tag) -f ./Dockerfile .
	docker buildx build --push --platform linux/amd64,linux/arm64 --target vendor -t $(namespace)$(tag)-vendor -f ./Dockerfile .

version:
	@echo $(namespace)$(tag)


.PHONY: clean vendor swagger build build-release build-vendor run version
