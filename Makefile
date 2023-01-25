REGISTRY = freenow
PROJECT_NAME ?= secrets-store-csi-driver-provider-spring-cloud-config
BUILD_GITHASH ?= $(shell git rev-parse HEAD)
BUILD_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
CONTAINER_IMAGE ?= $(REGISTRY)/$(PROJECT_NAME):$(BUILD_GITHASH)

.DEFAULT_GOAL = package

GO111MODULE ?= on
export GO111MODULE

setup:
	@echo "Setup..."
	$Q go env

.PHONY: build
build: setup
	@echo "Building..."
	$Q GOOS=linux CGO_ENABLED=0 go build .

.PHONY: package
package:
	docker buildx build --platform linux/amd64,linux/arm64 -t ${CONTAINER_IMAGE} .

.PHONY: test
test:
	docker run --rm -e CGO_ENABLED=0 -v "$(PWD):/go/src/github.com/freenowtech/$(PROJECT_NAME)" -w "/go/src/github.com/freenowtech/$(PROJECT_NAME)" golang:1.13.4-alpine  go test ./...

.PHONY: release
release: test
	docker buildx build --push --platform linux/amd64,linux/arm64 -t ${CONTAINER_IMAGE} .

.PHONY: release_latest
release_latest: test
ifeq (${BUILD_BRANCH},master)
	docker buildx build --push --platform linux/amd64,linux/arm64 -t ${REGISTRY}/${PROJECT_NAME}:latest -t ${CONTAINER_IMAGE} .
endif
