REGISTRY = freenow
PROJECT_NAME ?= secrets-store-csi-driver-provider-spring-cloud-config
BUILD_GITHASH ?= $(shell git rev-parse HEAD)
BUILD_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
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
	docker build --rm --pull -t ${CONTAINER_IMAGE} .

.PHONY: test
test:
	ls $(PWD)
	docker run --rm -e CGO_ENABLED=0 -v "$(PWD):/go/src/github.com/freenowtech/$(PROJECT_NAME)" -w "/go/src/github.com/freenowtech/$(PROJECT_NAME)" golang:1.13.4-alpine  go test

.PHONY: release
release: test package
	docker push ${CONTAINER_IMAGE}
ifeq (${BUILD_BRANCH},master)
	docker tag ${CONTAINER_IMAGE} ${REGISTRY}/${PROJECT_NAME}:latest
	docker push ${REGISTRY}/${PROJECT_NAME}:latest
endif
