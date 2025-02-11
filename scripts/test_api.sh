#!/bin/bash

BASE_URL="http://localhost:8081/api/v1"
ACCESS_TOKEN=""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Helper function for assertions
assert() {
    if [ $1 -ne 0 ]; then
        echo -e "${RED}❌ $2 failed${NC}"
        exit 1
    else
        echo -e "${GREEN}✅ $2 passed${NC}"
    fi
}

echo "🚀 Starting API Tests..."

# Test Account Creation
echo -e "\n${GREEN}1. Testing Account Management${NC}"
# Test invalid account creation
echo "Testing invalid account creation..."
INVALID_RESPONSE=$(curl -s -X POST "${BASE_URL}/accounts" \
  -H "Content-Type: application/json" \
  -d '{"name": ""}')
echo "$INVALID_RESPONSE" | grep "error" > /dev/null
assert $? "Invalid account validation"

# Create valid account
echo "Creating valid account..."
ACCOUNT_RESPONSE=$(curl -s -X POST "${BASE_URL}/accounts" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Account",
    "email": "account@example.com",
    "status": "active"
  }')
ACCOUNT_ID=$(echo $ACCOUNT_RESPONSE | jq -r '.data.id')
[ ! -z "$ACCOUNT_ID" ] && [ "$ACCOUNT_ID" != "null" ]
assert $? "Valid account creation"

# Test Organization Management
echo -e "\n${GREEN}2. Testing Organization Management${NC}"
# Create organization
ORG_RESPONSE=$(curl -s -X POST "${BASE_URL}/organizations" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Organization",
    "account_id": "'${ACCOUNT_ID}'",
    "status": "active"
  }')
ORG_ID=$(echo $ORG_RESPONSE | jq -r '.data.id')
[ ! -z "$ORG_ID" ] && [ "$ORG_ID" != "null" ]
assert $? "Organization creation"

# Test User Registration and Authentication
echo -e "\n${GREEN}3. Testing User Authentication${NC}"

# Test invalid registration
echo "Testing invalid registration..."
INVALID_REG_RESPONSE=$(curl -s -X POST "${BASE_URL}/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "invalid-email",
    "password": "short"
  }')
echo "$INVALID_REG_RESPONSE" | grep "error" > /dev/null
assert $? "Invalid registration validation"

# Test valid registration
echo "Testing valid registration..."
REGISTER_RESPONSE=$(curl -s -X POST "${BASE_URL}/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "password": "password123",
    "first_name": "John",
    "last_name": "Doe",
    "phone": "+1234567890",
    "type": "admin",
    "status": "active",
    "organization_id": "'${ORG_ID}'"
  }')
# Print response for debugging
echo "Register Response: $REGISTER_RESPONSE"
USER_ID=$(echo $REGISTER_RESPONSE | jq -r '.data.id')
# Check if registration was successful (should NOT contain error)
echo "$REGISTER_RESPONSE" | grep "error" > /dev/null && exit 1
[ ! -z "$USER_ID" ] && [ "$USER_ID" != "null" ]
assert $? "User registration"

# Test invalid login
echo "Testing invalid login..."
INVALID_LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "password": "wrongpassword"
  }')
echo "$INVALID_LOGIN_RESPONSE" | grep "error" > /dev/null
assert $? "Invalid login validation"

# Test valid login
echo "Testing valid login..."
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "password": "password123"
  }')
ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.access_token')
[ ! -z "$ACCESS_TOKEN" ] && [ "$ACCESS_TOKEN" != "null" ]
assert $? "Valid login"

# Test Protected Endpoints
echo -e "\n${GREEN}4. Testing Protected Endpoints${NC}"

# Test without token
echo "Testing endpoint without token..."
NO_TOKEN_RESPONSE=$(curl -s -X GET "${BASE_URL}/users/me")
echo "$NO_TOKEN_RESPONSE" | grep "error" > /dev/null
assert $? "Unauthorized access validation"

# Test with invalid token
echo "Testing endpoint with invalid token..."
INVALID_TOKEN_RESPONSE=$(curl -s -X GET "${BASE_URL}/users/me" \
  -H "Authorization: Bearer invalid_token")
echo "$INVALID_TOKEN_RESPONSE" | grep "error" > /dev/null
assert $? "Invalid token validation"

# Test with valid token
echo "Testing endpoint with valid token..."
ME_RESPONSE=$(curl -s -X GET "${BASE_URL}/users/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN")
# Print response for debugging
echo "Protected Endpoint Response: $ME_RESPONSE"
# Success response should contain "data" and not contain "error"
if echo "$ME_RESPONSE" | jq -e '.data' > /dev/null; then
  echo -e "${GREEN}✅ Got valid user data${NC}"
  true
else
  echo -e "${RED}❌ Invalid response format${NC}"
  false
fi
assert $? "Protected endpoint access"

# Test User Profile Update
echo -e "\n${GREEN}5. Testing User Profile Management${NC}"
echo "Updating profile for user: ${USER_ID}"
UPDATE_RESPONSE=$(curl -s -X PUT "${BASE_URL}/users/${USER_ID}" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Updated",
    "last_name": "Name",
    "email": "newuser@example.com",
    "type": "admin",
    "status": "active",
    "organization_id": "'$ORG_ID'",
    "phone": "+1234567890",
    "settings": {}
  }')
# Print response for debugging
echo "Update Response: $UPDATE_RESPONSE"
# Check for successful update
if echo "$UPDATE_RESPONSE" | jq -e '.data' > /dev/null; then
  echo -e "${GREEN}✅ Profile updated successfully${NC}"
  true
else
  echo -e "${RED}❌ Profile update failed: $(echo $UPDATE_RESPONSE | jq -r '.error')${NC}"
  false
fi
assert $? "Profile update"

# Verify updated profile
VERIFY_UPDATE=$(curl -s -X GET "${BASE_URL}/users/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN")
echo "$VERIFY_UPDATE" | grep "Updated" > /dev/null
assert $? "Profile update verification"

echo -e "\n${GREEN}✨ All tests completed successfully!${NC}" 