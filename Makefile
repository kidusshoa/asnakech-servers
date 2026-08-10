.PHONY: all build run test clean deps install dev generate up down logs tidy fmt \
	tools migrate-up migrate-down migrate-version migrate-force migrate-create up-infra ps

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
BINARY_NAME=asnakech-servers
AIR_VERSION=v1.61.7
MIGRATE_VERSION=v4.18.2
MIGRATE=$(GOCMD) run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)

# Default local DB (override via env or .env)
DATABASE_URL ?= postgres://asnakech:asnakech@localhost:5432/asnakech?sslmode=disable

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

# --- Database migrations ----------------------------------------------------

migrate-up:
	$(MIGRATE) -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	$(MIGRATE) -path ./migrations -database "$(DATABASE_URL)" down 1

migrate-version:
	$(MIGRATE) -path ./migrations -database "$(DATABASE_URL)" version

migrate-force:
	@test -n "$(VERSION)" || (echo "Usage: make migrate-force VERSION=N"; exit 1)
	$(MIGRATE) -path ./migrations -database "$(DATABASE_URL)" force $(VERSION)

migrate-create:
	@test -n "$(NAME)" || (echo "Usage: make migrate-create NAME=add_users"; exit 1)
	$(MIGRATE) create -ext sql -dir ./migrations -seq $(NAME)

# --- Docker -----------------------------------------------------------------

up:
	docker compose up -d --build

down:
	docker compose down

up-infra:
	docker compose up -d postgres redis minio minio-init

logs:
	docker compose logs -f api

ps:
	docker compose ps
