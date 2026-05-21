.PHONY: help db-up db-down db-logs db-shell migrate-create migrate-up migrate-down migrate-version run test build tidy

-include .env
export

# Default migration source
MIGRATIONS_PATH := ./migrations
DATABASE_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

help:
	@echo "Available targets:"
	@echo "  db-up           Start PostgreSQL in Docker"
	@echo "  db-down         Stop PostgreSQL"
	@echo "  db-logs         Tail PostgreSQL logs"
	@echo "  db-shell        Open a psql shell to the database"
	@echo "  migrate-up      Apply all pending migrations"
	@echo "  migrate-down    Roll back the most recent migration"
	@echo "  migrate-create  Create new migration files (use NAME=...)"
	@echo "  run             Run the application"
	@echo "  test            Run all tests with race detector"

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

db-logs:
	docker compose logs -f postgres

db-shell:
	docker compose exec postgres psql -U $(DB_USER) -d $(DB_NAME)

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

migrate-version:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" version

run:
	@trap 'exit 0' INT; go run ./cmd/server

test:
	go test -race -cover ./...

build:
	go build -o bin/server ./cmd/server

tidy:
	go mod tidy