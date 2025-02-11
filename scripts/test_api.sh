#!/bin/bash

BASE_URL="http://localhost:8081/api/v1"
ACCESS_TOKEN=""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "Testing Authentication APIs..."

# Create Account first
echo -e "\n${GREEN}Creating Account${NC}"
ACCOUNT_RESPONSE=$(curl -s -X POST "${BASE_URL}/accounts" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Account",
    "email": "account@example.com",
    "password": "password123",
    "status": "active"
  }')
echo "Account Response: $ACCOUNT_RESPONSE"
ACCOUNT_ID=$(echo $ACCOUNT_RESPONSE | jq -r '.data.id')
if [ -z "$ACCOUNT_ID" ] || [ "$ACCOUNT_ID" = "null" ]; then
    echo -e "${RED}Failed to create account${NC}"
    exit 1
fi

# Create Organization
echo -e "\n${GREEN}Creating Organization${NC}"
ORG_RESPONSE=$(curl -s -X POST "${BASE_URL}/organizations" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Organization",
    "account_id": "'${ACCOUNT_ID}'",
    "status": "active"
  }')
echo "Organization Response: $ORG_RESPONSE"
ORG_ID=$(echo $ORG_RESPONSE | jq -r '.data.id')
if [ -z "$ORG_ID" ] || [ "$ORG_ID" = "null" ]; then
    echo -e "${RED}Failed to create organization${NC}"
    exit 1
fi

# Test Register
echo -e "\n${GREEN}Testing Register${NC}"
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
echo "Register Response: $REGISTER_RESPONSE"

# Test Login after registration
echo -e "\n${GREEN}Testing Login${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "password": "password123"
  }')
echo "Login Response: $LOGIN_RESPONSE"

# Store token from login response
ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.access_token')

# Test Protected Endpoint
echo -e "\n${GREEN}Testing Protected Endpoint${NC}"
curl -X GET "${BASE_URL}/users/me" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" 