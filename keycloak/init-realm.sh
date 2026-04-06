#!/usr/bin/env bash
# Post-startup script for Keycloak development environment.
# Configures User Profile, disables MFA for dev, and creates test users (one per role).
#
# Run after Keycloak is healthy:
#   ./keycloak/init-realm.sh
#
# Idempotent — safe to run multiple times.

set -euo pipefail

KC_URL="${KC_URL:-http://localhost:8180}"
KC_ADMIN="${KC_ADMIN:-admin}"
KC_ADMIN_PASSWORD="${KC_ADMIN_PASSWORD:-admin}"
KC_REALM="${KC_REALM:-chesed}"
CAMPUS_ID="a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
TEST_PASSWORD="Test1234!"

# --- Helper functions ---

get_admin_token() {
  curl -sf -X POST "$KC_URL/realms/master/protocol/openid-connect/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "grant_type=password&client_id=admin-cli&username=$KC_ADMIN&password=$KC_ADMIN_PASSWORD" \
    | python3 -c "import sys,json;print(json.load(sys.stdin)['access_token'])"
}

create_user() {
  local username="$1" email="$2" first="$3" last="$4" role="$5" token="$6"

  # Check if user exists
  local count
  count=$(curl -sf "$KC_URL/admin/realms/$KC_REALM/users?username=$username&exact=true" \
    -H "Authorization: Bearer $token" | python3 -c "import sys,json;print(len(json.load(sys.stdin)))")

  if [ "$count" != "0" ]; then
    echo "    $username — already exists, skipping"
    return
  fi

  # Create user
  curl -sf -o /dev/null -X POST "$KC_URL/admin/realms/$KC_REALM/users" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "{
      \"username\": \"$username\",
      \"email\": \"$email\",
      \"firstName\": \"$first\",
      \"lastName\": \"$last\",
      \"enabled\": true,
      \"emailVerified\": true,
      \"requiredActions\": [],
      \"credentials\": [{\"type\": \"password\", \"value\": \"$TEST_PASSWORD\", \"temporary\": false}],
      \"attributes\": {\"campus_id\": [\"$CAMPUS_ID\"]}
    }"

  # Get user ID and assign role
  local user_id
  user_id=$(curl -sf "$KC_URL/admin/realms/$KC_REALM/users?username=$username&exact=true" \
    -H "Authorization: Bearer $token" | python3 -c "import sys,json;print(json.load(sys.stdin)[0]['id'])")

  local role_repr
  role_repr=$(curl -sf "$KC_URL/admin/realms/$KC_REALM/roles/$role" \
    -H "Authorization: Bearer $token")

  curl -sf -o /dev/null -X POST \
    "$KC_URL/admin/realms/$KC_REALM/users/$user_id/role-mappings/realm" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "[$role_repr]"

  echo "    $username — created with role $role"
}

# --- Step 1: User Profile ---

echo "==> [1/3] Configuring User Profile..."
TOKEN=$(get_admin_token)

PROFILE=$(curl -sf "$KC_URL/admin/realms/$KC_REALM/users/profile" \
  -H "Authorization: Bearer $TOKEN")

if echo "$PROFILE" | python3 -c "import sys,json;attrs=[a['name'] for a in json.load(sys.stdin)['attributes']];sys.exit(0 if 'campus_id' in attrs else 1)" 2>/dev/null; then
  echo "    User Profile already configured (campus_id exists). Skipping."
else
  UPDATED=$(echo "$PROFILE" | python3 -c "
import sys, json
profile = json.load(sys.stdin)
profile['attributes'].extend([
    {'name': 'campus_id', 'displayName': 'Campus ID', 'validations': {}, 'permissions': {'view': ['admin'], 'edit': ['admin']}, 'multivalued': False},
    {'name': 'person_id', 'displayName': 'Person ID', 'validations': {}, 'permissions': {'view': ['admin'], 'edit': ['admin']}, 'multivalued': False}
])
print(json.dumps(profile))
")

  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT \
    "$KC_URL/admin/realms/$KC_REALM/users/profile" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$UPDATED")

  if [ "$HTTP_CODE" = "200" ]; then
    echo "    User Profile updated (campus_id + person_id added)."
  else
    echo "    ERROR: User Profile update failed (HTTP $HTTP_CODE)"
    exit 1
  fi
fi

# --- Step 2: Disable conditional OTP for dev ---

echo "==> [2/3] Disabling conditional OTP (dev mode)..."
CURRENT_FLOW=$(curl -sf "$KC_URL/admin/realms/$KC_REALM" \
  -H "Authorization: Bearer $TOKEN" | python3 -c "import sys,json;print(json.load(sys.stdin).get('browserFlow',''))")

if [ "$CURRENT_FLOW" = "browser" ]; then
  echo "    Browser flow already set to 'browser' (no MFA). Skipping."
else
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT \
    "$KC_URL/admin/realms/$KC_REALM" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"browserFlow\": \"browser\"}")

  if [ "$HTTP_CODE" = "204" ]; then
    echo "    Browser flow switched to 'browser' (MFA disabled for dev)."
  else
    echo "    WARNING: Could not disable MFA (HTTP $HTTP_CODE). ADMIN login may require TOTP setup."
  fi
fi

# --- Step 3: Create test users ---

echo "==> [3/3] Creating test users (password: $TEST_PASSWORD)..."
# Refresh token in case it expired
TOKEN=$(get_admin_token)

create_user "volunteer"    "volunteer@chesed.test"    "Test" "Volunteer"    "VOLUNTEER"    "$TOKEN"
create_user "secretary"    "secretary@chesed.test"    "Test" "Secretary"    "SECRETARY"    "$TOKEN"
create_user "professional" "professional@chesed.test" "Test" "Professional" "PROFESSIONAL" "$TOKEN"
create_user "coordinator"  "coordinator@chesed.test"  "Test" "Coordinator"  "COORDINATOR"  "$TOKEN"
create_user "admin"        "admin@chesed.test"        "Test" "Admin"        "ADMIN"        "$TOKEN"

echo ""
echo "==> Done! Test users ready:"
echo ""
echo "  | Username     | Password   | Role         |"
echo "  |--------------|------------|--------------|"
echo "  | volunteer    | $TEST_PASSWORD | VOLUNTEER    |"
echo "  | secretary    | $TEST_PASSWORD | SECRETARY    |"
echo "  | professional | $TEST_PASSWORD | PROFESSIONAL |"
echo "  | coordinator  | $TEST_PASSWORD | COORDINATOR  |"
echo "  | admin        | $TEST_PASSWORD | ADMIN        |"
echo ""
echo "  Frontend: http://localhost:5173"
echo "  API:      http://localhost:8080/api/v1/health"
echo "  Keycloak: http://localhost:8180/admin (admin/admin)"
