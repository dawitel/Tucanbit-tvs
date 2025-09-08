BINARY_NAME=bin/tvs
ENTRY_FILE=cmd/tvs/main.go

MIGRATE_CMD=migrate
MIGRATION_DIR=db/migrations
SCHEMA_DIR=db/schema

include .env
export $(shell sed 's/=.*//' .env)

.PHONY: build run tidy clean test fmt vet mg migrate-up migrate-down migrate-new sqlc-gen

build: tidy fmt vet
	@mkdir -p bin
	@echo "Building the binary for tvs..."
	@go build -o $(BINARY_NAME) $(ENTRY_FILE)
	@echo "Build completed: $(BINARY_NAME)"

run: build
	@echo "Running tvs..."
	@./$(BINARY_NAME)

compose:
	@echo "Starting Docker Compose..."
	@docker-compose up -d
	@echo "Docker Compose started."

recompose:
	@echo "Starting Docker Compose from scratch..."
	@docker-compose down
	@docker rmi wms-wms
	@docker-compose up -d
	@echo "Docker Compose started."

stop:
	@echo "Stopping Docker Compose..."
	@docker-compose down
	@echo "Docker Compose stopped."

clean:
	@echo "Cleaning up..."
	@rm -rf bin
	@rm -rf log
	@rm -rf tests/unit/log
	@echo "Cleanup completed."

fmt:
	@echo "Formatting code..."
	@go fmt ./...

vet:
	@echo "Running go vet..."
	@go vet ./...

test:
	@echo "Running tests..."
	@go test ./tests/unit -cover
	@go test ./tests/integration -cover

tidy:
	@echo "Tidying up Go modules..."
	@go mod tidy

migrate-new:
ifndef name
	@echo "Error: Please provide a migration name. Usage: make migrate-new name=<name>"
	exit 1
endif
	@$(MIGRATE_CMD) create -ext sql -dir $(MIGRATION_DIR) -seq $(name)

migrate-up:
	@$(MIGRATE_CMD) -path $(MIGRATION_DIR) -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" up

migrate-down:
	@$(MIGRATE_CMD) -path $(MIGRATION_DIR) -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" down
	@make dump-schema

sqlc-gen:
	@echo "Generating SQLC..."
	@sqlc generate
