# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT

BINARY      := initiatives-api
MODULE      := github.com/linuxfoundation/lfx-v2-initiatives-service
CMD         := ./cmd/initiatives-api
IMAGE_NAME  := lfx-v2-initiatives-service
IMAGE_TAG   ?= dev
DOCKER_REG  ?= ghcr.io/linuxfoundation

.PHONY: all clean deps build run test fmt lint license-check \
        docker-build docker-run helm-install

all: build

## deps: download Go module dependencies
deps:
	go mod download

## build: compile the binary
build: deps
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/$(BINARY) $(CMD)

## run: run the service locally (requires .env to be sourced)
run: build
	./bin/$(BINARY)

## test: run all unit tests
test:
	go test -race -count=1 ./...

## fmt: format all Go source files
fmt:
	gofmt -w .
	goimports -w .

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## license-check: verify all .go files have license headers
license-check:
	@missing=$$(grep -rL "SPDX-License-Identifier: MIT" --include="*.go" .); \
	if [ -n "$$missing" ]; then \
	  echo "Missing license header in:"; \
	  echo "$$missing"; \
	  exit 1; \
	fi
	@echo "All .go files have license headers."

## clean: remove build artefacts
clean:
	rm -rf bin/

## docker-build: build the Docker image
docker-build:
	docker build -t $(DOCKER_REG)/$(IMAGE_NAME):$(IMAGE_TAG) .

## docker-run: run the Docker image locally
docker-run: docker-build
	docker run --rm --env-file .env -p 8080:8080 $(DOCKER_REG)/$(IMAGE_NAME):$(IMAGE_TAG)

## helm-install: install / upgrade the Helm chart in the current k8s context
helm-install:
	helm upgrade --install lfx-v2-initiatives-service \
	  charts/lfx-v2-initiatives-service \
	  --namespace lfx --create-namespace \
	  --values charts/lfx-v2-initiatives-service/values.yaml
