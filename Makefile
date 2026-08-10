.PHONY: all build run test clean deps install dev generate up down logs tidy fmt

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
BINARY_NAME=asnakech-servers
AIR_VERSION=v1.61.7

all: test build

build:
	$(GOBUILD) -o bin/$(BINARY_NAME) ./cmd/api

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -rf bin/ tmp/

run: build
	./bin/$(BINARY_NAME)

deps:
	$(GOCMD) get -u -t -v ./...
	$(GOCMD) mod tidy

install:
	$(GOCMD) mod download

# Hot-reload via Air when available; falls back to go run.
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not found — using go run (install: make tools)"; \
		$(GOCMD) run ./cmd/api; \
	fi

tools:
	$(GOCMD) install github.com/air-verse/air@$(AIR_VERSION)

generate:
	$(GOCMD) generate ./...

tidy:
	$(GOCMD) mod tidy

fmt:
	$(GOCMD) fmt ./...

# Local infrastructure (Postgres, Redis, MinIO) + API
up:
	docker compose up -d --build

down:
	docker compose down

# Infra only (run API via make dev on the host)
up-infra:
	docker compose up -d postgres redis minio minio-init

logs:
	docker compose logs -f api

ps:
	docker compose ps
