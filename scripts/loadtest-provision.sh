#!/usr/bin/env bash
# Provisions N synthetic devices for cmd/loadtest: creates a dedicated
# load-test site (if it doesn't exist), registers N devices through the
# real API (so the app-level registry is exercised, not bypassed), and
# syncs each device's secret into the Mosquitto password file — the same
# manual-sync step every real device registration requires (AGENTS.md
# item 1: broker credential provisioning is deliberately not automated
# beyond this).
#
# Writes device_id,secret pairs to loadtest-devices.csv (gitignored) for
# cmd/loadtest to read.
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8080/v1}"
OPERATOR_EMAIL="${OPERATOR_EMAIL:-admin@zgnis.test}"
OPERATOR_PASSWORD="${OPERATOR_PASSWORD:?set OPERATOR_PASSWORD}"
SITE_ID="${LOADTEST_SITE_ID:-SITE-LOADTEST}"
DEVICE_COUNT="${1:?usage: loadtest-provision.sh <device_count>}"
MOSQUITTO_CONTAINER="${MOSQUITTO_CONTAINER:-zgnis-mosquitto}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_FILE="$REPO_ROOT/loadtest-devices.csv"

echo "Logging in as $OPERATOR_EMAIL..."
TOKEN="$(curl -s -X POST "$API_BASE/auth/login" -H "Content-Type: application/json" \
  -d "{\"email\":\"$OPERATOR_EMAIL\",\"password\":\"$OPERATOR_PASSWORD\"}" | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')"
if [ -z "$TOKEN" ]; then
  echo "Login failed." >&2
  exit 1
fi

echo "Ensuring load-test site $SITE_ID exists..."
curl -s -X POST "$API_BASE/sites" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d "{\"site_id\":\"$SITE_ID\",\"name\":\"Load Test Site\",\"timezone\":\"Africa/Lagos\",\"system_size_kw\":1000}" \
  > /dev/null || true # 400 if it already exists — fine, continue

echo "Registering $DEVICE_COUNT devices..."
echo "device_id,secret" > "$OUT_FILE"

REGISTER_DELAY="${REGISTER_DELAY:-0.6}" # paced under the API's registration rate limiter (2 req/s) — see docs/load-test-results.md
for i in $(seq 1 "$DEVICE_COUNT"); do
  DEVICE_ID="ZG-LOAD-$(printf '%05d' "$i")"
  RESP="$(curl -s -X POST "$API_BASE/devices" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
    -d "{\"device_id\":\"$DEVICE_ID\",\"site_id\":\"$SITE_ID\"}")"
  sleep "$REGISTER_DELAY"
  SECRET="$(echo "$RESP" | python3 -c 'import sys,json;print(json.load(sys.stdin).get("secret",""))')"
  if [ -z "$SECRET" ]; then
    echo "  failed to register $DEVICE_ID: $RESP" >&2
    continue
  fi
  echo "$DEVICE_ID,$SECRET" >> "$OUT_FILE"
  docker exec "$MOSQUITTO_CONTAINER" mosquitto_passwd -b /mosquitto/config/passwd "$DEVICE_ID" "$SECRET" > /dev/null
  if [ $((i % 100)) -eq 0 ]; then
    echo "  provisioned $i/$DEVICE_COUNT"
  fi
done

echo "Restarting Mosquitto to pick up new credentials..."
docker restart "$MOSQUITTO_CONTAINER" > /dev/null
sleep 2

echo "Done. $DEVICE_COUNT devices written to $OUT_FILE"
