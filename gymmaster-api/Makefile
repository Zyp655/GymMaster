.DEFAULT_GOAL := help
BIN := bin/gymmaster

include hack/hack.mk

help: ## Liệt kê các lệnh
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

run: ## Chạy API (đọc .env)
	go run . http

dev: ## Chạy API với hot reload (cần: go install github.com/air-verse/air@latest)
	air

build: ## Build binary vào bin/
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN) .

test: ## Chạy unit test
	go test ./... -race -count=1

itest: ## Chạy integration test (cần GYMMASTER_TEST_DSN, ví dụ postgres://gymmaster:gymmaster@localhost:5432/gymmaster?sslmode=disable)
	go test -tags integration ./internal/... -count=1

cover: ## Test + báo cáo coverage
	go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1

lint: ## golangci-lint
	golangci-lint run ./...

migrate-up: ## Áp dụng migration
	go run . migrate up

migrate-down: ## Rollback 1 migration (make migrate-down n=2)
	go run . migrate down $(or $(n),1)

migrate-version: ## Xem version schema
	go run . migrate version

migrate-create: ## Tạo cặp file migration mới (make migrate-create name=add_members)
	@test -n "$(name)" || (echo "Thiếu name=..."; exit 1)
	@n=$$(ls migrations/*.up.sql 2>/dev/null | wc -l | tr -d ' '); v=$$(printf "%06d" $$((n+1))); \
	touch migrations/$${v}_$(name).up.sql migrations/$${v}_$(name).down.sql; \
	echo "Đã tạo migrations/$${v}_$(name).{up,down}.sql"

seed: ## Tạo dữ liệu khởi tạo (OWNER + chi nhánh mẫu)
	go run . seed

.PHONY: help run dev build test itest cover lint migrate-up migrate-down migrate-version migrate-create seed
