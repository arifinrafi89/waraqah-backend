# Waraqah backend commands (BACKEND_PLAN.md §22.4). Needs GNU make and bash.
-include .env
export

FRONTEND_DIR ?= ../waraqah-frontend
SQLC ?= sqlc

.PHONY: dev test check migrate migrate-down-up sqlc seed frontend-sync contract-export set-role smoke logs fmt lint

dev:
	docker compose up -d db
	MIGRATE_ON_START=true go run ./cmd/api

test:
	go test ./...

fmt:
	gofmt -l -w .

lint:
	golangci-lint run ./...

check:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo "gofmt needed" && exit 1)
	go vet ./...
	./scripts/check-file-length.sh
	@if command -v golangci-lint >/dev/null; then golangci-lint run ./...; else echo "golangci-lint not installed, skipped"; fi
	@if command -v $(SQLC) >/dev/null && [ -f db/sqlc.yaml ]; then $(SQLC) diff -f db/sqlc.yaml; else echo "sqlc diff skipped"; fi
	go test ./...

migrate:
	go run ./cmd/api -migrate up

migrate-down-up:
	go run ./cmd/api -migrate down-up

sqlc:
	$(SQLC) generate -f db/sqlc.yaml

seed:
	go run ./cmd/seed

frontend-sync:
	git -C $(FRONTEND_DIR) pull

contract-export:
	./scripts/export-contract.sh

set-role:
	go run ./cmd/admin set-role $(EMAIL) $(ROLE)

smoke:
	./scripts/smoke.sh

logs:
	@echo "run 'make dev' in a terminal; logs go to stdout"
