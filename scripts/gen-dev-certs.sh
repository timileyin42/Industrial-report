#!/usr/bin/env bash
# Generates a self-signed CA + server certificate for LOCAL DEV VERIFICATION
# ONLY. This proves the TLS code path (Mosquitto's 8883 listener, Postgres
# sslmode, the API's optional TLS listener) actually works end to end — it
# does NOT satisfy "TLS everywhere" for a real deployment. A real deployment
# needs a CA-issued cert for a real hostname; see docs/tls.md.
#
# DEV-ONLY, NOT FOR PRODUCTION.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CERT_DIR="$REPO_ROOT/mosquitto/config/certs"
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

DAYS="${CERT_DAYS:-825}"

echo "Generating dev-only CA + server certificate in $CERT_DIR (valid $DAYS days)..."

openssl genrsa -out ca.key 4096 2>/dev/null
openssl req -x509 -new -nodes -key ca.key -sha256 -days "$DAYS" \
  -subj "/CN=Zgnis Dev CA/O=Zgnis Dev/C=NG" -out ca.crt 2>/dev/null

openssl genrsa -out server.key 4096 2>/dev/null
openssl req -new -key server.key -subj "/CN=localhost/O=Zgnis Dev/C=NG" -out server.csr 2>/dev/null
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -days "$DAYS" -sha256 -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1") \
  -out server.crt 2>/dev/null

rm -f server.csr ca.srl
chmod 600 ca.key server.key

echo "Done:"
ls -la "$CERT_DIR"
echo
echo "DEV-ONLY certificate — never use this for a real deployment."
echo "Next: uncomment the TLS listener in mosquitto/config/mosquitto.conf"
echo "(already pointed at these paths) and restart Mosquitto. See docs/tls.md."
