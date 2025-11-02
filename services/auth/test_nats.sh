#!/bin/bash

# NATS API Testing Script for Auth Service
# This script tests the main NATS topics of the Auth service

NATS_URL="${NATS_URL:-nats://localhost:4222}"

echo "Testing Auth Service via NATS at $NATS_URL"
echo "=================================="

# Check if nats CLI is installed
if ! command -v nats &> /dev/null; then
    echo "Error: nats CLI is not installed"
    echo "Install it with: go install github.com/nats-io/natscli/nats@latest"
    exit 1
fi

# Test 1: Registration
echo -e "\n1. Testing Registration..."
REGISTRATION_RESPONSE=$(nats request auth.register '{
  "username": "testuser",
  "email": "test@example.com",
  "password": "SecurePassword123",
  "metadata": {
    "reason": "Testing"
  }
}' --server="$NATS_URL" 2>&1)

echo "$REGISTRATION_RESPONSE"
REQUEST_ID=$(echo "$REGISTRATION_RESPONSE" | grep -o '"request_id":"[^"]*"' | cut -d'"' -f4)

echo "Request ID: $REQUEST_ID"

# Note: In a real scenario, an admin would approve the registration
# For testing, you would need to:
# 1. Publish an admin.registration.approved event via NATS, OR
# 2. Manually insert a user into the database

echo -e "\nTo approve registration, an admin needs to publish:"
echo "nats pub admin.registration.approved '{\"request_id\":\"$REQUEST_ID\",\"approver_id\":\"admin-uuid\"}'"

# Test 2: Login (will fail unless user is created)
echo -e "\n2. Testing Login (may fail if user not approved)..."
LOGIN_RESPONSE=$(nats request auth.login '{
  "email": "test@example.com",
  "password": "SecurePassword123",
  "device_id": "test-device",
  "user_agent": "test-script",
  "ip_address": "127.0.0.1"
}' --server="$NATS_URL" 2>&1)

echo "$LOGIN_RESPONSE"
ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
REFRESH_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$ACCESS_TOKEN" ] && [ "$ACCESS_TOKEN" != "" ]; then
  # Test 3: Validate Token
  echo -e "\n3. Testing Token Validation..."
  nats request auth.validate "{
    \"token\": \"$ACCESS_TOKEN\"
  }" --server="$NATS_URL"

  # Test 4: Refresh Token
  echo -e "\n4. Testing Token Refresh..."
  REFRESH_RESPONSE=$(nats request auth.refresh "{
    \"refresh_token\": \"$REFRESH_TOKEN\",
    \"device_id\": \"test-device\",
    \"user_agent\": \"test-script\",
    \"ip_address\": \"127.0.0.1\"
  }" --server="$NATS_URL" 2>&1)
  
  echo "$REFRESH_RESPONSE"
  NEW_ACCESS_TOKEN=$(echo "$REFRESH_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

  # Test 5: Change Password (needs user_id)
  echo -e "\n5. Testing Change Password..."
  echo "Skipping - requires user_id from validated token"

  # Test 6: Logout (needs user_id)
  echo -e "\n6. Testing Logout..."
  echo "Skipping - requires user_id from validated token"

else
  echo -e "\nSkipping authenticated tests (login failed or user not approved)"
  echo "To test full flow:"
  echo "1. Approve the registration request with ID: $REQUEST_ID"
  echo "2. Or manually create a user in the database"
  echo "3. Then run this script again"
fi

echo -e "\n=================================="
echo "Testing complete!"
echo ""
echo "Available NATS topics:"
echo "  - auth.register (public)"
echo "  - auth.login (public)"
echo "  - auth.refresh (public)"
echo "  - auth.validate (public)"
echo "  - auth.logout (requires authentication)"
echo "  - auth.logout_all (requires authentication)"
echo "  - auth.change_password (requires authentication)"
