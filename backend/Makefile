# ========== Variables ==========
APP_NAME    := darsly-backend
LOG_FILE    := logs/app.log
BUILD_DIR   := bin
MAIN_PATH   := cmd/main.go
SWAG_MAIN   := cmd/main.go
MIGRATE_DIR := migrations
DB_URL      ?= $(shell grep DATABASE_URL .env 2>/dev/null | cut -d= -f2)
DB_HOST     ?= $(shell grep ^DB_HOST     .env 2>/dev/null | cut -d= -f2)
DB_PORT     ?= $(shell grep ^DB_PORT     .env 2>/dev/null | cut -d= -f2)
DB_NAME     ?= $(shell grep ^DB_NAME     .env 2>/dev/null | cut -d= -f2)
DB_USER     ?= $(shell grep ^DB_USER     .env 2>/dev/null | cut -d= -f2)
DB_PASSWORD ?= $(shell grep ^DB_PASSWORD .env 2>/dev/null | cut -d= -f2)
MIGRATE     := $(shell go env GOPATH)/bin/migrate
SCRIPTS_DIR := scripts

.PHONY: all build run test test-unit test-integration test-e2e test-load \
        lint fmt vet swagger migrate-up migrate-down \
        docker-up docker-down docker-logs livekit-up livekit-up-recording livekit-down \
        clean deps seed seed-test api-test \
        db-reset dev logs logs-f logs-errors logs-clear help

# ========== Test infra (integration) ==========
TEST_DB_CONTAINER    := darsly-test-db
TEST_DB_PORT         ?= 5459
TEST_DB_USER         ?= postgres
TEST_DB_PASSWORD     ?= postgres
TEST_DB_NAME         ?= darsly_test
TEST_DB_IMAGE        ?= postgres:16-alpine
TEST_REDIS_CONTAINER := darsly-test-redis
TEST_REDIS_PORT      ?= 6399
TEST_REDIS_IMAGE     ?= redis:7-alpine

#make api-test — scripts/test_apis.sh ni ishga tushiradi
#make dev — build qilib serverni ishga tushiradi
#make seed — seed ma'lumotlarini DB ga yuklaydi
#make db-reset — DB ni tozalab qayta migrate + seed qiladi

# ========== Build ==========
build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

run:
	go run $(MAIN_PATH)

# ========== Quality ==========
fmt:
	gofmt -s -w .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

# Default `test` = fast unit tests (no external services, no build tags).
test: test-unit

test-unit:
	go test -race -coverprofile=coverage.out ./...

test-cover: test-unit
	go tool cover -html=coverage.out -o coverage.html

# Integration tests: boot a throwaway Postgres container, apply migrations
# (done by the harness), run `-tags=integration`, then tear the container down.
test-integration:
	@echo "→ Starting throwaway Postgres ($(TEST_DB_PORT)) + Redis ($(TEST_REDIS_PORT))..."
	@docker rm -f $(TEST_DB_CONTAINER) $(TEST_REDIS_CONTAINER) >/dev/null 2>&1 || true
	@docker run -d --name $(TEST_DB_CONTAINER) \
	  -e POSTGRES_USER=$(TEST_DB_USER) \
	  -e POSTGRES_PASSWORD=$(TEST_DB_PASSWORD) \
	  -e POSTGRES_DB=$(TEST_DB_NAME) \
	  -p $(TEST_DB_PORT):5432 $(TEST_DB_IMAGE) >/dev/null
	@docker run -d --name $(TEST_REDIS_CONTAINER) \
	  -p $(TEST_REDIS_PORT):6379 $(TEST_REDIS_IMAGE) >/dev/null
	@echo "→ Waiting for Postgres to become ready..."
	@for i in $$(seq 1 30); do \
	  docker exec $(TEST_DB_CONTAINER) pg_isready -U $(TEST_DB_USER) >/dev/null 2>&1 && break; \
	  sleep 1; \
	done
	@echo "→ Running integration tests..."
	@DARSLY_TEST_DB_HOST=localhost \
	 DARSLY_TEST_DB_PORT=$(TEST_DB_PORT) \
	 DARSLY_TEST_DB_USER=$(TEST_DB_USER) \
	 DARSLY_TEST_DB_PASSWORD=$(TEST_DB_PASSWORD) \
	 DARSLY_TEST_DB_NAME=$(TEST_DB_NAME) \
	 DARSLY_TEST_REDIS_HOST=localhost \
	 DARSLY_TEST_REDIS_PORT=$(TEST_REDIS_PORT) \
	 DARSLY_MIGRATIONS_PATH=$(abspath $(MIGRATE_DIR)) \
	 go test -tags=integration -race -p 1 ./... ; \
	 status=$$? ; \
	 echo "→ Tearing down test containers..." ; \
	 docker rm -f $(TEST_DB_CONTAINER) $(TEST_REDIS_CONTAINER) >/dev/null 2>&1 || true ; \
	 exit $$status

# E2E tests (Playwright) live in the frontend package.
test-e2e:
	@echo "→ Running frontend E2E tests (Playwright)..."
	cd ../frontend && npm run test:e2e

# Load / stress tests (k6). Requires the app + stack running.
test-load:
	@echo "→ Running load tests (k6)..."
	@command -v k6 >/dev/null 2>&1 || { echo "k6 not installed: https://k6.io/docs/get-started/installation/"; exit 1; }
	@for s in tests/load/*.js; do echo "== $$s =="; k6 run $$s; done

# ========== Swagger ==========
swagger:
	swag init -g $(SWAG_MAIN) --output docs --parseDependency

# ========== Database ==========
migrate-up:
	$(MIGRATE) -path $(MIGRATE_DIR) -database "$(DB_URL)" up

migrate-down:
	$(MIGRATE) -path $(MIGRATE_DIR) -database "$(DB_URL)" down

migrate-create:
	$(MIGRATE) create -ext sql -dir $(MIGRATE_DIR) -seq $(name)

# Test ma'lumotlarini yuklash (o'chirmasdan, ON CONFLICT qoidasi bilan)
seed:
	@echo "→ Test seed ma'lumotlarini yuklamoqda..."
	PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) \
	  -U $(DB_USER) -d $(DB_NAME) \
	  -f $(SCRIPTS_DIR)/seed_test_data.sql
	@echo "✓ Seed bajarildi"

# API test skriptini ishga tushirish
api-test:
	@echo "→ API testlar ishga tushmoqda..."
	bash $(SCRIPTS_DIR)/test_apis.sh

# Build qilib serverni ishga tushirish
dev: build
	./$(BUILD_DIR)/$(APP_NAME)

# DB ni migration gacha reset qilib, qayta seed yuklash
db-reset:
	@echo "→ Migratsiyalar qayta ishlatilmoqda..."
	$(MIGRATE) -path $(MIGRATE_DIR) -database "$(DB_URL)" drop -f
	$(MIGRATE) -path $(MIGRATE_DIR) -database "$(DB_URL)" up
	@echo "→ Seed yuklanmoqda..."
	PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) \
	  -U $(DB_USER) -d $(DB_NAME) \
	  -f $(SCRIPTS_DIR)/seed_test_data.sql
	@echo "✓ DB reset va seed bajarildi"

# ========== Logs ==========
logs:
	@test -f $(LOG_FILE) || { echo "Log fayl topilmadi: $(LOG_FILE)"; exit 1; }
	tac $(LOG_FILE) | less -R

logs-f:
	tail -f $(LOG_FILE)

logs-errors:
	@test -f $(LOG_FILE) || { echo "Log fayl topilmadi: $(LOG_FILE)"; exit 1; }
	tac $(LOG_FILE) | grep '"level":"ERROR"' | less -R

logs-clear:
	@read -p "Log faylni tozalash. Ishonchingiz komilmi? [y/N] " ans && [ "$$ans" = "y" ]
	> $(LOG_FILE)
	@echo "✓ Log fayl tozalandi"

# ========== Docker (local dev infra: postgres:5442 redis:6399 minio:9020/21 rabbitmq:5682) ==========
DEV_COMPOSE := docker-compose.dev.yml

docker-up:
	docker compose -f $(DEV_COMPOSE) up -d

docker-down:
	docker compose -f $(DEV_COMPOSE) down

docker-logs:
	docker compose -f $(DEV_COMPOSE) logs -f

# LiveKit SFU (services/livekit): video (7880) + TURN (3478)
livekit-up:
	docker compose -f ../services/livekit/docker-compose.yml up -d

livekit-up-recording:
	docker compose -f ../services/livekit/docker-compose.yml --profile recording up -d

livekit-down:
	docker compose -f ../services/livekit/docker-compose.yml down

# ========== Cleanup ==========
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

deps:
	go mod tidy
	go mod verify

# ========== Default ==========
all: fmt vet build

help:
	@echo "Available targets:"
	@echo "  build          - Build binary to bin/"
	@echo "  run            - Run server directly"
	@echo "  fmt            - Format code (gofmt)"
	@echo "  vet            - Run go vet"
	@echo "  lint           - Run golangci-lint"
	@echo "  test           - Run unit tests with race detector (alias: test-unit)"
	@echo "  test-unit      - Fast unit tests (mocks, no external services)"
	@echo "  test-integration - Boot throwaway Postgres + run -tags=integration tests"
	@echo "  test-e2e       - Frontend E2E tests (Playwright)"
	@echo "  test-load      - Load/stress tests (k6, requires running stack)"
	@echo "  test-cover     - Unit tests + HTML coverage report"
	@echo "  swagger        - Generate Swagger docs"
	@echo "  migrate-up     - Apply all migrations"
	@echo "  migrate-down   - Rollback last migration"
	@echo "  migrate-create name=xxx - Create new migration"
	@echo "  docker-up      - Start dev infra (docker-compose.dev.yml: pg/redis/minio/rabbitmq)"
	@echo "  docker-down    - Stop dev infra"
	@echo "  livekit-up     - Start LiveKit SFU (services/livekit)"
	@echo "  livekit-up-recording - Start LiveKit + Egress (recording profile)"
	@echo "  livekit-down   - Stop LiveKit SFU"
	@echo "  deps           - go mod tidy + verify"
	@echo "  clean          - Remove build artifacts"
	@echo "  seed           - Load test seed data into DB"
	@echo "  db-reset       - Drop + migrate up + seed"
	@echo "  api-test       - Run API test script (scripts/test_apis.sh)"
	@echo "  dev            - Build and run server"
	@echo "  logs           - View log file newest first (less viewer)"
	@echo "  logs-f         - Follow log file in real-time (tail -f)"
	@echo "  logs-errors    - View only ERROR logs, newest first"
	@echo "  logs-clear     - Clear the log file (with confirmation)"
