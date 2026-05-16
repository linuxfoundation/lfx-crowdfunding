# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT

BINARY      := initiatives-api
MODULE      := github.com/linuxfoundation/lfx-v2-initiatives-service
CMD         := ./cmd/initiatives-api
IMAGE_NAME  := lfx-v2-initiatives-service
IMAGE_TAG   ?= dev
DOCKER_REG  ?= ghcr.io/linuxfoundation

# kind / local deploy
KIND_CLUSTER     ?= initiatives-dev
NAMESPACE        ?= lfx-v2-initiatives-service
RELEASE_NAME     ?= initiatives
CHART_DIR        ?= charts/lfx-v2-initiatives-service
DEPLOYMENT_NAME  ?= $(RELEASE_NAME)-lfx-v2-initiatives-service
DEPLOY_TAG       ?= dev-$(shell date +%Y%m%d%H%M%S)

.PHONY: all clean deps build run test fmt lint license-check \
        docker-build docker-run helm-install deploy-kind db-seed

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

## db-seed: load development seed data into the local database (localhost only)
db-seed:
	@echo "$$DATABASE_URL" | grep -qE "localhost|127\.0\.0\.1" || \
		{ echo "ERROR: db-seed refused — DATABASE_URL does not point to localhost"; exit 1; }
	psql "$$DATABASE_URL" -f db/seed.sql

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

## deploy-kind: build image, load into kind, helm upgrade, wait, port-forward
##   Requires: kind, kubectl, helm, docker, .env file
##   Override cluster:   make deploy-kind KIND_CLUSTER=my-cluster
##   Override namespace: make deploy-kind NAMESPACE=my-ns
deploy-kind:
	@echo "==> Ensuring kind cluster '$(KIND_CLUSTER)' exists..."
	@kind get clusters | grep -q '^$(KIND_CLUSTER)$$' || kind create cluster --name $(KIND_CLUSTER)
	@echo "==> Switching kubectl context to kind-$(KIND_CLUSTER)..."
	@kubectl config use-context kind-$(KIND_CLUSTER)
	@echo "==> Building Docker image $(IMAGE_NAME):$(DEPLOY_TAG)..."
	@docker build -t $(IMAGE_NAME):$(DEPLOY_TAG) .
	@echo "==> Loading image into kind cluster..."
	@kind load docker-image $(IMAGE_NAME):$(DEPLOY_TAG) --name $(KIND_CLUSTER)
	@echo "==> Ensuring namespace '$(NAMESPACE)' exists..."
	@kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f - >/dev/null
	@echo "==> Seeding k8s Secret from .env (placeholder — ESO not present in kind)..."
	@set -a; . ./.env; set +a; \
		KIND_DB_URL=$$(echo "$$DATABASE_URL" | sed 's/localhost/host.docker.internal/g'); \
		kubectl -n $(NAMESPACE) create secret generic lfx-v2-initiatives-service-secrets \
			--from-literal=DATABASE_URL="$$KIND_DB_URL" \
			--from-literal=STRIPE_SECRET_KEY="$$STRIPE_SECRET_KEY" \
			--from-literal=STRIPE_WEBHOOK_SECRET="$$STRIPE_WEBHOOK_SECRET" \
			--from-literal=LEDGER_API_KEY="$$LEDGER_API_KEY" \
			--dry-run=client -o yaml | kubectl apply -f - >/dev/null
	@echo "==> Running helm upgrade --install..."
	@helm upgrade --install $(RELEASE_NAME) $(CHART_DIR) \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--set image.repository=$(IMAGE_NAME) \
		--set image.tag=$(DEPLOY_TAG) \
		--set image.pullPolicy=IfNotPresent \
		--set autoscaling.enabled=false \
		--set createPlaceholderSecret=false \
		--set-string config.DISABLED_MOCK_LOCAL_PRINCIPAL=local-dev-user \
		--set-string config.LEDGER_BASE_URL=https://ledger-api.lfx.dev
	@echo "==> Waiting for rollout..."
	@kubectl -n $(NAMESPACE) rollout status deploy/$(DEPLOYMENT_NAME) --timeout=180s
	@echo "==> Setting up port-forward on :18080 -> svc :80..."
	@existing_pid=$$(lsof -tiTCP:18080 -sTCP:LISTEN 2>/dev/null || true); \
	if [ -n "$$existing_pid" ]; then \
		echo "Stopping existing listener on 18080 (PID $$existing_pid)..."; \
		kill $$existing_pid 2>/dev/null || true; \
	fi
	@nohup kubectl -n $(NAMESPACE) port-forward svc/$(DEPLOYMENT_NAME) 18080:80 \
		>/tmp/$(RELEASE_NAME)-port-forward.log 2>&1 &
	@sleep 2
	@curl -sf http://127.0.0.1:18080/livez >/dev/null \
		&& echo "==> Deployment ready. Local access: http://127.0.0.1:18080" \
		|| echo "==> Port-forward started; verify with: tail -f /tmp/$(RELEASE_NAME)-port-forward.log"
