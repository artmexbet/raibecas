#!/bin/bash

# API Testing Script for Auth Service
# This script tests the main endpoints of the Auth service

BASE_URL="${BASE_URL:-http://localhost:8081}"

echo "Testing Auth Service at $BASE_URL"
echo "=================================="

# Test 1: Health Check
echo -e "\n1. Testing Health Check..."
curl -X GET "$BASE_URL/health" \
  -H "Content-Type: application/json" \
  -s | jq '.'

# Test 2: Registration
echo -e "\n2. Testing Registration..."
REGISTRATION_RESPONSE=$(curl -X POST "$BASE_URL/api/v1/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "SecurePassword123",
    "metadata": {
      "reason": "Testing"
    }
  }' \
  -s)

echo "$REGISTRATION_RESPONSE" | jq '.'
REQUEST_ID=$(echo "$REGISTRATION_RESPONSE" | jq -r '.request_id')

# Note: In a real scenario, an admin would approve the registration
# For testing, you would need to:
# 1. Publish an admin.registration.approved event via NATS, OR
# 2. Manually insert a user into the database

# Test 3: Login (will fail unless user is created)
echo -e "\n3. Testing Login (may fail if user not approved)..."
LOGIN_RESPONSE=$(curl -X POST "$BASE_URL/api/v1/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "SecurePassword123"
  }' \
  -s)

echo "$LOGIN_RESPONSE" | jq '.'
ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token // empty')
REFRESH_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.refresh_token // empty')

if [ -n "$ACCESS_TOKEN" ] && [ "$ACCESS_TOKEN" != "null" ]; then
  # Test 4: Validate Token
  echo -e "\n4. Testing Token Validation..."
  curl -X POST "$BASE_URL/api/v1/validate" \
    -H "Content-Type: application/json" \
    -d "{
      \"token\": \"$ACCESS_TOKEN\"
    }" \
    -s | jq '.'

  # Test 5: Refresh Token
  echo -e "\n5. Testing Token Refresh..."
  REFRESH_RESPONSE=$(curl -X POST "$BASE_URL/api/v1/refresh" \
    -H "Content-Type: application/json" \
    -d "{
      \"refresh_token\": \"$REFRESH_TOKEN\"
    }" \
    -s)
  
  echo "$REFRESH_RESPONSE" | jq '.'
  NEW_ACCESS_TOKEN=$(echo "$REFRESH_RESPONSE" | jq -r '.access_token // empty')

  # Test 6: Change Password
  echo -e "\n6. Testing Change Password..."
  curl -X POST "$BASE_URL/api/v1/change-password" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $NEW_ACCESS_TOKEN" \
    -d '{
      "old_password": "SecurePassword123",
      "new_password": "NewSecurePassword456"
    }' \
    -s | jq '.'

  # Test 7: Logout
  echo -e "\n7. Testing Logout..."
  curl -X POST "$BASE_URL/api/v1/logout" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $NEW_ACCESS_TOKEN" \
    -s | jq '.'

else
  echo -e "\nSkipping authenticated tests (login failed or user not approved)"
  echo "To test full flow:"
  echo "1. Approve the registration request with ID: $REQUEST_ID"
  echo "2. Or manually create a user in the database"
  echo "3. Then run this script again"
fi

echo -e "\n=================================="
echo "Testing complete!"
