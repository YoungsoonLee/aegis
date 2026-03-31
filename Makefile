APP_NAME := aegis
VERSION := 0.1.0
BUILD_DIR := bin
GO := go

.PHONY: all build run test lint clean docker

all: build

build:
	$(GO) build -ldflags "-s -w" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/aegis

run: build
	AEGIS_CONFIG=configs/aegis.yaml $(BUILD_DIR)/$(APP_NAME)

test:
	$(GO) test -race -cover ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)

docker:
	docker build -t $(APP_NAME):$(VERSION) -f deployments/docker/Dockerfile .

compose-up:
	docker compose -f examples/docker-compose/docker-compose.yaml up --build

compose-down:
	docker compose -f examples/docker-compose/docker-compose.yaml down --rmi local

.PHONY: help compose-up compose-down
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  run          - Build and run locally"
	@echo "  test         - Run tests"
	@echo "  lint         - Run linter"
	@echo "  clean        - Remove build artifacts"
	@echo "  docker       - Build Docker image"
	@echo "  compose-up   - Start Docker Compose demo"
	@echo "  compose-down - Stop Docker Compose demo"
