.PHONY: build run test clean docker-build docker-run

# Build variables
BINARY_NAME=admin-api
BUILD_DIR=bin

# Go commands
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./admin-api/cmd/api

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)
	go clean

# Docker commands
docker-build:
	docker compose build

docker-run:
	docker compose up

docker-down:
	docker compose down

# Development helpers
dev: docker-down docker-build docker-run

# Add these commands
check-services:
	@echo "\nRunning containers:"
	docker ps -a
	@echo "\nDocker images:"
	docker images
	@echo "\nDocker networks:"
	docker network ls

check-api-logs:
	@echo "\nAPI Logs:"
	docker-compose logs api

check-postgres-logs:
	@echo "\nPostgres Logs:"
	docker-compose logs postgres

# Add new debug command
debug:
	@echo "Full system check..."
	@make check-services
	@make check-api-logs
	@make check-postgres-logs
	@echo "\nDocker compose config:"
	docker-compose config

# Add a quick start command for testing
start-test:
	@echo "Starting services in foreground mode..."
	docker-compose up --build 

# Add these test commands
test-flow:
	@echo "\n1. Creating account..."
	@curl -v -X POST http://localhost:8080/api/v1/accounts \
		-H "Content-Type: application/json" \
		-d '{"name":"Test Account","email":"test@example.com"}'

	@echo "\n2. Creating clinician..."
	@curl -v -X POST http://localhost:8080/api/v1/clinicians \
		-H "Content-Type: application/json" \
		-d '{"email":"doctor@example.com","name":"Dr. Smith","password":"password123"}'

	@echo "\n3. Testing login..."
	@curl -v -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"doctor@example.com","password":"password123"}'

	@echo "\nNote: Save the access_token from the login response for authenticated requests"

# Add this to test authenticated endpoints
test-auth:
	@echo "\nTesting authenticated endpoint (replace TOKEN with actual token)..."
	@curl -v -X GET http://localhost:8080/api/v1/accounts \
		-H "Authorization: Bearer YOUR_TOKEN_HERE"

# Add a helper to save token
save-token:
	@curl -s -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"doctor@example.com","password":"password123"}' \
		| grep -o '"access_token":"[^"]*' | cut -d'"' -f4 > .token

# Use saved token
test-with-token:
	@TOKEN=$$(cat .token); \
	curl -v -X GET http://localhost:8080/api/v1/accounts \
		-H "Authorization: Bearer $$TOKEN" 

# Add these new test commands
test-organization:
	@echo "\nCreating organization for account..."
	@TOKEN=$$(cat .token); \
	ACCOUNT_ID="72285c94-a49d-417b-8a52-26a4f06afc8c"; \
	curl -v -X POST "http://localhost:8080/api/v1/accounts/$$ACCOUNT_ID/organizations" \
		-H "Authorization: Bearer $$TOKEN" \
		-H "Content-Type: application/json" \
		-d '{"name":"Test Organization","status":"active"}'

test-refresh:
	@echo "\n=== Testing Token Refresh ==="
	@REFRESH_TOKEN=$$(cat .refresh_token); \
	curl -s -X POST http://localhost:8080/api/v1/auth/refresh \
		-H "Content-Type: application/json" \
		-d "{\"refresh_token\":\"$$REFRESH_TOKEN\"}" | jq

test-full-flow: test-flow
	@echo "\nWaiting for token to be saved..."
	@sleep 2
	@make save-token
	@make test-with-token
	@make test-organization
	@make test-refresh

# Add error case testing
test-errors:
	@echo "\nTesting invalid login..."
	@curl -v -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"wrong@example.com","password":"wrong"}'

	@echo "\nTesting unauthorized access..."
	@curl -v -X GET http://localhost:8080/api/v1/accounts

	@echo "\nTesting invalid token..."
	@curl -v -X GET http://localhost:8080/api/v1/accounts \
		-H "Authorization: Bearer invalid.token.here" 

# API Testing Suite
.PHONY: test-api-* test-all ensure-api

# Main test command that runs all tests in sequence
test-all: check-dependencies
	@echo "\n=== Starting Test Suite ==="
	
	@echo "\n=== 1. Initial Setup ==="
	# Create admin clinician first
	@curl -s -X POST http://localhost:8080/api/v1/clinicians \
		-H "Content-Type: application/json" \
		-d '{"email":"admin@example.com","name":"Admin","password":"password123"}' | jq

	# Login to get token
	@curl -s -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"admin@example.com","password":"password123"}' | tee auth.json | jq

	@echo "\n=== Saving tokens ==="
	@cat auth.json | jq -r '.data.access_token' > .token
	@cat auth.json | jq -r '.data.refresh_token' > .refresh_token

	# Create account and organization with token
	@TOKEN=$$(cat .token); \
	echo "\nCreating account..." && \
	curl -s -X POST http://localhost:8080/api/v1/accounts \
		-H "Authorization: Bearer $$TOKEN" \
		-H "Content-Type: application/json" \
		-d '{"name":"Test Account","email":"account@example.com"}' | tee account.json | jq && \
	ACCOUNT_ID=$$(cat account.json | jq -r '.data.id'); \
	echo "\nCreating organization..." && \
	curl -s -X POST "http://localhost:8080/api/v1/accounts/$$ACCOUNT_ID/organizations" \
		-H "Authorization: Bearer $$TOKEN" \
		-H "Content-Type: application/json" \
		-d '{"name":"Test Organization","status":"active"}' | tee org.json | jq
	
	@echo "\n=== Testing Protected Endpoints ==="
	@make test-api-accounts
	@make test-api-clinics
	@make test-api-clinicians
	@make test-api-rbac
	@make test-api-errors
	
	@echo "\n=== Test Suite Completed ==="
	@rm -f auth.json .token .refresh_token account.json org.json

# Authentication Tests
test-api-auth:
	@echo "\n=== Testing Authentication ==="
	@echo "\n1. Login with invalid credentials..."
	@curl -s -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"wrong@example.com","password":"wrong"}' | jq

	@echo "\n2. Create test clinician..."
	@curl -s -X POST http://localhost:8080/api/v1/clinicians \
		-H "Content-Type: application/json" \
		-d '{"email":"doctor@example.com","name":"Dr. Smith","password":"password123"}' | jq

	@echo "\n3. Login with valid credentials..."
	@curl -s -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"doctor@example.com","password":"password123"}' | jq

	@echo "\n4. Test token refresh..."
	@make test-refresh

# Account Tests
test-api-accounts:
	@echo "\n=== Testing Accounts ==="
	@TOKEN=$$(cat .token); \
	curl -s -X GET http://localhost:8080/api/v1/accounts \
		-H "Authorization: Bearer $$TOKEN" | jq

	@TOKEN=$$(cat .token); \
	ACCOUNT_ID=$$(curl -s -X GET http://localhost:8080/api/v1/accounts \
		-H "Authorization: Bearer $$TOKEN" | jq -r '.data[0].id'); \
	curl -s -X POST "http://localhost:8080/api/v1/accounts/$$ACCOUNT_ID/organizations" \
		-H "Authorization: Bearer $$TOKEN" \
		-H "Content-Type: application/json" \
		-d '{"name":"Test Organization","status":"active"}' | jq

# Helper function to get organization ID
define get_org_id
$(shell curl -s -X GET http://localhost:8080/api/v1/accounts \
	-H "Authorization: Bearer $$(cat .token)" | jq -r '.data[0].organizations[0].id')
endef

# Update test-api-clinics with better organization ID handling
test-api-clinics:
	@echo "\n=== Testing Clinics ==="
	@TOKEN=$$(cat .token); \
	ORG_ID=$$(curl -s -X GET http://localhost:8080/api/v1/accounts \
		-H "Authorization: Bearer $$TOKEN" | jq -r '.data[0].organizations[0].id'); \
	echo "Using Organization ID: $$ORG_ID" && \
	echo "\n2. Create clinic..." && \
	curl -s -X POST "http://localhost:8080/api/v1/clinics" \
		-H "Authorization: Bearer $$TOKEN" \
		-H "Content-Type: application/json" \
		-d "{\"organization_id\":\"$$ORG_ID\",\"name\":\"Test Clinic\",\"location\":\"Test Location\"}" | jq && \
	echo "\n3. List clinics..." && \
	curl -s -X GET "http://localhost:8080/api/v1/clinics?organization_id=$$ORG_ID" \
		-H "Authorization: Bearer $$TOKEN" | jq

# Update test-api-clinicians with better organization ID handling
test-api-clinicians:
	@echo "\n=== Testing Clinicians ==="
	@TOKEN=$$(cat .token); \
	ORG_ID=$$(curl -s -X GET http://localhost:8080/api/v1/accounts \
		-H "Authorization: Bearer $$TOKEN" | jq -r '.data[0].organizations[0].id'); \
	echo "Using Organization ID: $$ORG_ID" && \
	echo "\n2. Create another clinician..." && \
	curl -s -X POST http://localhost:8080/api/v1/clinicians \
		-H "Authorization: Bearer $$TOKEN" \
		-H "Content-Type: application/json" \
		-d "{\"organization_id\":\"$$ORG_ID\",\"email\":\"doctor2@example.com\",\"name\":\"Dr. Jones\",\"password\":\"password123\"}" | jq && \
	echo "\n3. List clinicians..." && \
	curl -s -X GET "http://localhost:8080/api/v1/clinicians?organization_id=$$ORG_ID" \
		-H "Authorization: Bearer $$TOKEN" | jq

# Update test-api-rbac with better organization ID handling
test-api-rbac:
	@echo "\n=== Testing RBAC ==="
	@TOKEN=$$(cat .token); \
	ORG_ID=$$(curl -s -X GET http://localhost:8080/api/v1/accounts \
		-H "Authorization: Bearer $$TOKEN" | jq -r '.data[0].organizations[0].id'); \
	echo "Using Organization ID: $$ORG_ID" && \
	echo "\n2. Create role..." && \
	curl -s -X POST http://localhost:8080/api/v1/rbac/roles \
		-H "Authorization: Bearer $$TOKEN" \
		-H "Content-Type: application/json" \
		-d "{\"name\":\"Doctor\",\"organization_id\":\"$$ORG_ID\",\"description\":\"Doctor role\"}" | jq && \
	echo "\n3. Create permission..." && \
	curl -s -X POST http://localhost:8080/api/v1/rbac/permissions \
		-H "Authorization: Bearer $$TOKEN" \
		-H "Content-Type: application/json" \
		-d "{\"name\":\"patient:read\",\"organization_id\":\"$$ORG_ID\",\"description\":\"Can read patient data\"}" | jq

# Error Tests
test-api-errors:
	@echo "\n=== Testing Error Cases ==="
	@echo "\n1. Invalid login..."
	@curl -s -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"wrong@example.com","password":"wrong"}' | jq

	@echo "\n2. Missing token..."
	@curl -s -X GET http://localhost:8080/api/v1/accounts | jq

	@echo "\n3. Invalid token..."
	@curl -s -X GET http://localhost:8080/api/v1/accounts \
		-H "Authorization: Bearer invalid.token" | jq

	@echo "\n4. Invalid input data..."
	@curl -s -X POST http://localhost:8080/api/v1/accounts \
		-H "Content-Type: application/json" \
		-d '{"name":"","email":"invalid"}' | jq

# Add jq installation check
check-dependencies:
	@which jq > /dev/null || (echo "Error: jq is not installed. Please install jq first." && exit 1) 

# Add these database-related commands
migrate-reset:
	@echo "Resetting database..."
	docker compose exec admin-api migrate -path=/migrations -database postgres://postgres:postgres@postgres:5432/aiclinic?sslmode=disable drop -f
	docker compose exec admin-api migrate -path=/migrations -database postgres://postgres:postgres@postgres:5432/aiclinic?sslmode=disable up

# Add a simpler migrate command for just running migrations
migrate:
	@echo "Running migrations..."
	docker compose exec admin-api migrate -path=/migrations -database postgres://postgres:postgres@postgres:5432/aiclinic?sslmode=disable up

# Add these test commands
test-permissions:
	@echo "Testing permissions API..."
	@TOKEN=$$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
		-H 'Content-Type: application/json' \
		-d '{"email":"admin@example.com","password":"admin123"}' | jq -r '.data.access_token'); \
	echo "Token: $$TOKEN" && \
	echo "\nTesting roles endpoint..." && \
	curl -v -X GET "http://localhost:8080/api/v1/rbac/roles" \
		-H "Authorization: Bearer $$TOKEN"

test-roles:
	@echo "Testing roles API..."
	@TOKEN=$$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
		-H 'Content-Type: application/json' \
		-d '{"email":"admin@example.com","password":"admin123"}' | jq -r '.data.access_token'); \
	echo "Creating role..." && \
	curl -s -X POST "http://localhost:8080/api/v1/rbac/roles" \
		-H "Authorization: Bearer $$TOKEN" \
		-H "Content-Type: application/json" \
		-d '{"name":"doctor","description":"Doctor role"}' | jq && \
	echo "\nListing roles..." && \
	curl -s -X GET "http://localhost:8080/api/v1/rbac/roles" \
		-H "Authorization: Bearer $$TOKEN" | jq

.PHONY: test-api

ensure-api:
	@echo "Ensuring API is running..."
	@docker compose ps | grep -q "admin-api.*Up" || (echo "Starting API..." && docker compose up -d)
	@sleep 5  # Give the API time to start

test-api: ensure-api
	@echo "Running API tests..."
	@go test ./test/api/... -v

test-api-verbose: ensure-api
	@echo "Running API tests with verbose output..."
	@go test ./test/api/... -v -count=1

.DEFAULT_GOAL := build 