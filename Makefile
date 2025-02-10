.PHONY: test-api ensure-api test-cleanup test-user-api test-events test-patient test-clinic

# Start services and test account creation API
test-api: ensure-api
	@echo "🚀 Testing Account Creation API..."
	@echo "\n1. Rebuilding and starting services..."
	@docker compose build --no-cache admin-api
	@docker compose up -d postgres redis admin-api
	@echo "⏳ Waiting for services to be ready..."
	@echo "Waiting for API to be healthy..."
	@for i in {1..30}; do \
	  if curl -s http://localhost:8080/health/live > /dev/null; then \
		echo "API is ready!"; \
		break; \
	  fi; \
	  echo "Waiting... $$i/30"; \
	  if [ $$i -eq 15 ]; then \
		echo "\n🔍 Checking API logs:"; \
		docker compose logs admin-api; \
	  fi; \
	  sleep 1; \
	done

	@echo "\n2. Running migrations..."
	@docker compose exec admin-api migrate -path=/app/migrations -database "postgres://postgres:postgres@postgres:5432/aiclinic?sslmode=disable" up

	@echo "\n3. Testing Account Creation endpoint..."
	@echo "Creating account..."
	@ACCOUNT_RESPONSE=$$(curl -s -X POST http://localhost:8080/api/v1/accounts \
		-H "Content-Type: application/json" \
		-d '{ \
			"name": "Test Hospital", \
			"email": "admin@hospital.com", \
			"password": "password123", \
			"status": "active" \
		}') \
	&& echo "Account creation response: $$ACCOUNT_RESPONSE" \
	&& echo $$ACCOUNT_RESPONSE | jq -r '.data.id' > account_id.txt

	@echo "\nVerifying account in database..."
	@docker compose exec postgres psql -U postgres -d aiclinic -c "SELECT id, email, status FROM accounts;"

	@echo "\n4. Creating Organization..."
	@ACCOUNT_ID=$$(cat account_id.txt) && \
	ORG_RESPONSE=$$(curl -s -X POST http://localhost:8080/api/v1/accounts/$$ACCOUNT_ID/organizations \
		-H "Content-Type: application/json" \
		-d '{ \
			"name": "Test Organization", \
			"status": "active" \
		}') \
	&& echo "Organization creation response: $$ORG_RESPONSE" \
	&& echo $$ORG_RESPONSE | jq -r '.data.id' > org_id.txt

	@echo "\nVerifying organization in database..."
	@docker compose exec postgres psql -U postgres -d aiclinic -c "SELECT id, name, status FROM organizations;"

	@echo "\nCreated organization with ID: $$(cat org_id.txt)"

# Ensure clean environment and dependencies
ensure-api:
	@echo "🧹 Cleaning up previous instances..."
	@docker compose down -v
	@echo "📦 Checking dependencies..."
	@which migrate || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "🔄 Starting required services..."
	@docker compose up -d postgres redis
	@echo "⏳ Waiting for databases to be ready..."
	@sleep 5

# Cleanup after testing
test-cleanup:
	@echo "🧹 Cleaning up test environment..."
	@rm -f account_id.txt org_id.txt token.txt
	@docker compose down -v 

test-user-api:
	@echo "🚀 Testing User Creation API..."
	@echo "\n1. Getting auth token..."
	@TOKEN=$$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"admin@hospital.com","password":"password123"}' | jq -r '.data.access_token') \
	&& if [ "$$TOKEN" = "null" ]; then \
		echo "Failed to get token"; \
		exit 1; \
	fi \
	&& echo "Got token: $$TOKEN" \
	&& curl -v -X POST http://localhost:8080/api/v1/users \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $$TOKEN" \
		-d "{ \
			\"organization_id\": \"$$(cat org_id.txt)\", \
			\"name\": \"Test User\", \
			\"email\": \"test@example.com\", \
			\"password\": \"password123\", \
			\"type\": \"admin\" \
		}"

test-events: ensure-api
	@echo "🚀 Testing Event System..."
	@echo "\n1. Waiting for account setup..."
	@sleep 5  # Add delay to ensure account is ready
	@echo "\n2. Getting auth token..."
	@echo "Attempting login with admin@hospital.com..."
	@TOKEN=$$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"admin@hospital.com","password":"password123"}') \
	&& echo "Login response: $$TOKEN" \
	&& TOKEN_VALUE=$$(echo "$$TOKEN" | jq -r '.data.access_token') \
	&& if [ "$$TOKEN_VALUE" = "null" ]; then \
		echo "Failed to get token. Full response: $$TOKEN"; \
		exit 1; \
	fi \
	&& echo "Got token: $$TOKEN_VALUE" \
	&& echo "\n3. Creating user..." \
	&& ORG_ID=$$(cat org_id.txt) \
	&& echo "Using org ID: $$ORG_ID" \
	&& curl -s -X POST http://localhost:8080/api/v1/users \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $$TOKEN_VALUE" \
		-d "{ \
			\"organization_id\": \"$$ORG_ID\", \
			\"name\": \"Test User\", \
			\"email\": \"test@example.com\", \
			\"password\": \"password123\", \
			\"type\": \"admin\" \
		}"

	@echo "\n4. Checking outbox events in database..."
	@docker compose exec postgres psql -U postgres -d aiclinic -c "SELECT event_type, status, error FROM outbox_events WHERE event_type IN ('CLINIC_CREATE', 'CLINIC_UPDATE', 'CLINIC_DELETE', 'PATIENT_CREATE', 'PATIENT_UPDATE', 'PATIENT_DELETE') ORDER BY created_at DESC LIMIT 5;"

	@echo "\n5. Checking events in Redis..."
	@echo "All Redis keys:"
	@docker compose exec redis redis-cli KEYS "*"
	@echo "\nEvent details (events list):"
	@docker compose exec redis redis-cli --raw LRANGE events 0 -1 || echo "No events found"
	@echo "\nEvent details (event.* keys):"
	@docker compose exec redis redis-cli --raw KEYS "event.*" || echo "No event.* keys found"
	@echo "\nEvent details (outbox.* keys):"
	@docker compose exec redis redis-cli --raw KEYS "outbox.*" || echo "No outbox.* keys found"

	@echo "\nTest complete! 🎉"

# Regular test without logs
test-all: ensure-api test-api test-events test-clinic test-patient
	@make test-cleanup

# Test with logs
test-all-debug: ensure-api test-api test-events test-clinic test-patient
	@echo "\n🔍 Checking container logs..."
	@docker compose logs
	@echo "\nDone checking logs"
	@make test-cleanup

# Fix the token issue in test-patient and test-clinic
test-patient:
	@echo "\n6. Creating patient..."
	@TOKEN=$$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"admin@hospital.com","password":"password123"}' | jq -r '.data.access_token') \
	&& echo "Using clinic ID: $$(cat clinic_id.txt)" \
	&& REQUEST='{ \
			"organization_id": "'$$(cat org_id.txt)'", \
			"clinic_id": "'$$(cat clinic_id.txt)'", \
			"name": "Test Patient", \
			"email": "patient@example.com", \
			"phone": "+1234567890", \
			"dob": "1990-01-01T00:00:00.000Z", \
			"address": "123 Patient St", \
			"status": "active" \
		}' \
	&& echo "Request: $$REQUEST" \
	&& curl -v -X POST http://localhost:8080/api/v1/patients \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $$TOKEN" \
		-d "$$REQUEST"

test-clinic:
	@echo "\n7. Creating clinic..."
	@TOKEN=$$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"admin@hospital.com","password":"password123"}' | jq -r '.data.access_token') \
	&& RESPONSE=$$(curl -s -X POST http://localhost:8080/api/v1/clinics \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $$TOKEN" \
		-d "{ \
			\"organization_id\": \"$$(cat org_id.txt)\", \
			\"name\": \"Test Clinic\", \
			\"address\": \"123 Test St\", \
			\"location\": \"Building A\", \
			\"status\": \"active\" \
		}") \
	&& echo $$RESPONSE \
	&& echo $$RESPONSE | jq -r '.data.id' > clinic_id.txt 