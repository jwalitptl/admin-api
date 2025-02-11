.PHONY: test test-worker test-auth test-patient test-clinic test-coverage dev

# Test configuration
TEST_FLAGS := -v -race
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Application settings
APP_NAME := admin-api
DB_NAME := admin_db

# Docker settings
DOCKER_COMPOSE := docker-compose
DOCKER_COMPOSE_FILE := docker-compose.yml
PORT := 8081
DB_USER := postgres
DB_PASS := postgres

# Main test targets
test: ## Run all tests
	go test $(TEST_FLAGS) ./...

test-coverage: ## Run tests with coverage
	go test $(TEST_FLAGS) -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)

# Individual test groups
test-worker: ## Run worker-related tests
	go test $(TEST_FLAGS) ./cmd/worker/... ./pkg/worker/...

test-auth: ## Run authentication-related tests
	go test $(TEST_FLAGS) ./internal/service/auth/... ./internal/handler/auth/...

test-patient: ## Run patient-related tests
	go test $(TEST_FLAGS) ./internal/service/patient/... ./internal/handler/patient/...

test-clinic: ## Run clinic-related tests
	go test $(TEST_FLAGS) ./internal/service/clinic/... ./internal/handler/clinic/...

# Development helpers
mock-gen: ## Generate mocks for testing
	mockgen -source=internal/repository/interfaces.go -destination=internal/mocks/repository_mocks.go -package=mocks
	mockgen -source=pkg/messaging/broker.go -destination=internal/mocks/broker_mocks.go -package=mocks
	mockgen -source=internal/service/auth/service.go -destination=internal/mocks/auth_service_mock.go -package=mocks

test-clean: ## Clean test cache
	go clean -testcache

# New commands
all: dev

dev: ## Run everything in development mode
	@echo "🚀 Starting development environment..."
	
	@echo "🧹 Cleaning up previous instances..."
	@$(DOCKER_COMPOSE) down -v 2>/dev/null || true
	@pkill -f "$(APP_NAME)" 2>/dev/null || true
	@lsof -ti:$(PORT) | xargs kill -9 2>/dev/null || true
	
	@echo "🐳 Starting Docker services..."
	@$(DOCKER_COMPOSE) up -d postgres redis
	
	@echo "⏳ Waiting for database..."
	@until docker exec admin-api-postgres-1 pg_isready -U $(DB_USER) > /dev/null 2>&1; do \
		sleep 1; \
	done
	
	@echo "🔄 Running migrations..."
	@docker exec -i admin-api-postgres-1 psql -U $(DB_USER) -c "DROP DATABASE IF EXISTS $(DB_NAME);" 2>/dev/null || true
	@docker exec -i admin-api-postgres-1 psql -U $(DB_USER) -c "CREATE DATABASE $(DB_NAME);"
	@for f in migrations/*.up.sql; do \
		echo "Applying $$f..."; \
		docker exec -i admin-api-postgres-1 psql -U $(DB_USER) -d $(DB_NAME) < $$f; \
	done
	
	@echo "🚀 Starting API server..."
	@go run cmd/api/main.go > api.log 2>&1 & echo $$! > api.pid
	@echo "🚀 Starting worker..."
	@go run cmd/worker/main.go > worker.log 2>&1 & echo $$! > worker.pid
	
	@echo "⏳ Waiting for API server..."
	@for i in {1..30}; do \
		if curl -s http://localhost:$(PORT)/health > /dev/null; then \
			echo "✅ API server is ready!"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then \
			echo "❌ API server failed to start"; \
			cat api.log; \
			exit 1; \
		fi; \
		sleep 1; \
	done
	
	@echo "🧪 Running API tests..."
	@chmod +x scripts/test_api.sh
	@./scripts/test_api.sh
	
	@echo "✨ Development environment is ready!"
	@echo "📝 Logs available in api.log and worker.log"
	@echo "💡 Use 'make stop' to stop all services"

stop: ## Stop all services
	@if [ -f $(DOCKER_COMPOSE_FILE) ]; then \
		$(DOCKER_COMPOSE) down || true; \
	fi
	@pkill -f "$(APP_NAME)" || true

clean: stop ## Clean up everything
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@lsof -ti:8081 | xargs kill -9 2>/dev/null || true
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML) api.log worker.log || true
	@if [ -n "$$(docker volume ls -q)" ]; then \
		$(DOCKER_COMPOSE) down -v || true
		docker volume rm $$(docker volume ls -q) 2>/dev/null || true; \
	fi

api-test: ## Run API tests
	@echo "Running API tests..."
	@chmod +x scripts/test_api.sh
	@./scripts/test_api.sh

run-all: clean start api-test ## Run everything from scratch
	@echo "All tests completed"
	@make stop 