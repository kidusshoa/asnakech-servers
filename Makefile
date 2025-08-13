.PHONY: build run test clean

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=asnakech-servers

all: test build

build:
	$(GOBUILD) -o bin/$(BINARY_NAME) ./cmd/api

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f bin/$(BINARY_NAME)

run:
	$(GOBUILD) -o bin/$(BINARY_NAME) ./cmd/api
	./bin/$(BINARY_NAME)

deps:
	$(GOGET) -u -t -v ./...

# Install dependencies
install:
	go mod download

# Run the application in development mode
dev:
	echo "Starting development server..."
	go run cmd/api/main.go

# Generate code (if you have any code generation steps)
generate:
	go generate ./...
