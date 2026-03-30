.PHONY: run build test migrate-up migrate-down sqlc seed lint docker-up docker-down

# ─── Dev ────────────────────────────────────────────────────────────────────
run:
	go run ./cmd/api/main.go

run-watch:
	air -c .air.toml

build:
	go build -ldflags="-s -w" -o bin/api ./cmd/api/main.go

# ─── Database ────────────────────────────────────────────────────────────────
migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

migrate-drop:
	migrate -path ./migrations -database "$(DATABASE_URL)" drop -f

migrate-new:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir ./migrations -seq $$name

# ─── SQLC ────────────────────────────────────────────────────────────────────
sqlc:
	sqlc generate

# ─── Testing ────────────────────────────────────────────────────────────────
test:
	go test ./... -v -race -coverprofile=coverage.out

test-cover:
	go tool cover -html=coverage.out

# ─── Code Quality ────────────────────────────────────────────────────────────
lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

# ─── Docker ──────────────────────────────────────────────────────────────────
docker-up:
	docker compose -f ../docker-compose.yml up -d postgres

docker-down:
	docker compose -f ../docker-compose.yml down

# ─── Seed ────────────────────────────────────────────────────────────────────
seed:
	go run ./cmd/seed/main.go

# ─── Setup (first time) ──────────────────────────────────────────────────────
setup: docker-up
	@echo "Waiting for Postgres..."
	@sleep 3
	$(MAKE) migrate-up
	$(MAKE) seed
	@echo "✓ Backend ready. Run: make run"
