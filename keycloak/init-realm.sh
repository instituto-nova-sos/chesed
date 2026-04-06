#!/usr/bin/env bash
# Post-startup script for Keycloak development environment.
# Configures: User Profile, SMTP (Mailpit), MFA (disabled in dev), and test users.
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
SMTP_HOST="${SMTP_HOST:-mailpit}"
SMTP_PORT="${SMTP_PORT:-1025}"

# --- Helper functions ---

get_admin_token() {
  curl -sf -X POST "$KC_URL/realms/master/protocol/openid-connect/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "grant_type=password&client_id=admin-cli&username=$KC_ADMIN&password=$KC_ADMIN_PASSWORD" \
    | python3 -c "import sys,json;print(json.load(sys.stdin)['access_token'])"
}

create_user() {
  local username="$1" email="$2" first="$3" last="$4" role="$5" token="$6"

  local count
  count=$(curl -sf "$KC_URL/admin/realms/$KC_REALM/users?username=$username&exact=true" \
    -H "Authorization: Bearer $token" | python3 -c "import sys,json;print(len(json.load(sys.stdin)))")

  if [ "$count" != "0" ]; then
    echo "    $username — already exists, skipping"
    return
  fi

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

echo "==> [1/5] Configuring User Profile..."
TOKEN=$(get_admin_token)

PROFILE=$(curl -sf "$KC_URL/admin/realms/$KC_REALM/users/profile" \
  -H "Authorization: Bearer $TOKEN")

if echo "$PROFILE" | python3 -c "import sys,json;attrs=[a['name'] for a in json.load(sys.stdin)['attributes']];sys.exit(0 if 'campus_id' in attrs else 1)" 2>/dev/null; then
  echo "    User Profile already configured. Skipping."
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
    echo "    User Profile updated (campus_id + person_id)."
  else
    echo "    ERROR: User Profile update failed (HTTP $HTTP_CODE)"
    exit 1
  fi
fi

# --- Step 2: SMTP Configuration (Mailpit for dev) ---

echo "==> [2/5] Configuring SMTP (Mailpit)..."
CURRENT_SMTP=$(curl -sf "$KC_URL/admin/realms/$KC_REALM" \
  -H "Authorization: Bearer $TOKEN" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('smtpServer',{}).get('host',''))")

if [ "$CURRENT_SMTP" = "$SMTP_HOST" ]; then
  echo "    SMTP already configured ($SMTP_HOST). Skipping."
else
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT \
    "$KC_URL/admin/realms/$KC_REALM" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"smtpServer\": {
        \"host\": \"$SMTP_HOST\",
        \"port\": \"$SMTP_PORT\",
        \"from\": \"noreply@chesed.test\",
        \"fromDisplayName\": \"Instituto Nova SOS\",
        \"ssl\": \"false\",
        \"starttls\": \"false\",
        \"auth\": \"false\"
      }
    }")

  if [ "$HTTP_CODE" = "204" ]; then
    echo "    SMTP configured (host=$SMTP_HOST, port=$SMTP_PORT)."
  else
    echo "    WARNING: SMTP configuration failed (HTTP $HTTP_CODE). Emails will not be sent."
  fi
fi

# --- Step 3: Disable conditional MFA for dev ---

echo "==> [3/5] Disabling conditional MFA (dev mode)..."
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
    echo "    WARNING: Could not disable MFA (HTTP $HTTP_CODE)."
  fi
fi

# --- Step 4: Disable email verification for test users ---
# realm-export.json has verifyEmail: true for production.
# For dev, we disable it so test users can login immediately.

echo "==> [4/5] Disabling email verification (dev mode)..."
VERIFY_EMAIL=$(curl -sf "$KC_URL/admin/realms/$KC_REALM" \
  -H "Authorization: Bearer $TOKEN" | python3 -c "import sys,json;print(json.load(sys.stdin).get('verifyEmail',False))")

if [ "$VERIFY_EMAIL" = "False" ]; then
  echo "    Email verification already disabled. Skipping."
else
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT \
    "$KC_URL/admin/realms/$KC_REALM" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"verifyEmail\": false}")

  if [ "$HTTP_CODE" = "204" ]; then
    echo "    Email verification disabled for dev."
    echo "    NOTE: Production must enable verifyEmail: true."
  else
    echo "    WARNING: Could not disable email verification (HTTP $HTTP_CODE)."
  fi
fi

# --- Step 5: Create test users ---

echo "==> [5/5] Creating test users (password: $TEST_PASSWORD)..."
TOKEN=$(get_admin_token)

create_user "volunteer"    "volunteer@chesed.test"    "Test" "Volunteer"    "VOLUNTEER"    "$TOKEN"
create_user "secretary"    "secretary@chesed.test"    "Test" "Secretary"    "SECRETARY"    "$TOKEN"
create_user "professional" "professional@chesed.test" "Test" "Professional" "PROFESSIONAL" "$TOKEN"
create_user "coordinator"  "coordinator@chesed.test"  "Test" "Coordinator"  "COORDINATOR"  "$TOKEN"
create_user "admin"        "admin@chesed.test"        "Test" "Admin"        "ADMIN"        "$TOKEN"

echo ""
echo "==> Done! Development environment ready."
echo ""
echo "  | Username     | Password   | Role         |"
echo "  |--------------|------------|--------------|"
echo "  | volunteer    | $TEST_PASSWORD | VOLUNTEER    |"
echo "  | secretary    | $TEST_PASSWORD | SECRETARY    |"
echo "  | professional | $TEST_PASSWORD | PROFESSIONAL |"
echo "  | coordinator  | $TEST_PASSWORD | COORDINATOR  |"
echo "  | admin        | $TEST_PASSWORD | ADMIN        |"
echo ""
echo "  Frontend:  http://localhost:5173"
echo "  API:       http://localhost:8080/api/v1/health"
echo "  Keycloak:  http://localhost:8180/admin (admin/admin)"
echo "  Mailpit:   http://localhost:8025 (email inbox)"
echo ""
echo "  NOTE: Email verification and MFA are DISABLED in dev."
echo "  Production must enable: verifyEmail=true, browserFlow='Browser with Conditional MFA'"
