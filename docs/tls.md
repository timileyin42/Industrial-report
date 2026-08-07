# TLS

CLAUDE.md: *"TLS on every hop once anything leaves localhost... do not
carry that 'disabled' state into any environment a real device or
external user touches."*

Phase 4 wires up the TLS code paths and activates a **self-signed,
dev-only** certificate so those paths are actually exercised locally.
**This does not satisfy "TLS everywhere" for a real deployment** — a
self-signed cert isn't trusted by any real device, browser, or CA-aware
client. What's still needed before this touches a real device or public
internet is listed in the checklist at the bottom.

## What's built

| Hop | What | Env var |
|---|---|---|
| Device → Mosquitto | TLS listener on 8883, cert from `scripts/gen-dev-certs.sh` | n/a (broker config) |
| Ingestor → Mosquitto | Optional TLS client, verifies the broker's cert against a CA file | `MQTT_TLS_CA_CERT` (+ use an `ssl://` broker URL) |
| API/Ingestor → Postgres | `sslmode` appended to the connection string if not already present | `DATABASE_SSLMODE` (default `disable`) |
| Dashboard → API | Optional Echo TLS listener | `API_TLS_CERT_FILE`, `API_TLS_KEY_FILE` |

All four are **off by default** — unset env vars mean Phase 0-3's exact
prior behavior, so this is additive, not a breaking change.

## Generating dev certs

```bash
./scripts/gen-dev-certs.sh
```

Writes `ca.crt`, `ca.key`, `server.crt`, `server.key` into
`mosquitto/config/certs/` (gitignored — never commit these, even though
they're dev-only). Run this **before** starting Mosquitto, since
`mosquitto.conf`'s 8883 listener now points at these paths unconditionally
and will fail to start without them.

## Verifying locally

```bash
# Regenerate certs, then bring Mosquitto up
./scripts/gen-dev-certs.sh
docker compose up -d

# TLS handshake against the broker
openssl s_client -connect localhost:8883 -CAfile mosquitto/config/certs/ca.crt </dev/null

# Ingestor over TLS
export MQTT_BROKER_URL="ssl://localhost:8883"
export MQTT_TLS_CA_CERT="mosquitto/config/certs/ca.crt"
go run ./cmd/ingestor

# Postgres over TLS (local Timescale doesn't have server-side TLS enabled
# by default in this compose file — DATABASE_SSLMODE=require will fail
# until that's turned on; sslmode=disable, the default, still works)
export DATABASE_SSLMODE=disable

# API over TLS, once dev certs exist
export API_TLS_CERT_FILE="mosquitto/config/certs/server.crt"
export API_TLS_KEY_FILE="mosquitto/config/certs/server.key"
go run ./cmd/api
curl -k https://localhost:8080/v1/auth/login -H "Content-Type: application/json" -d '{}'
# -k because curl doesn't trust our dev-only self-signed cert — expected locally
```

## Before this touches a real device or public internet

- [ ] Replace the self-signed cert with a real CA-issued certificate (Let's Encrypt or a client-purchased cert) for a **real hostname** the client owns — no amount of local work produces this.
- [ ] Decide whether the broker requires mutual TLS (`require_certificate yes` + per-device client certs) — not enabled here; per-device MQTT username/password auth (already in place since Phase 0) is the current device-identity mechanism.
- [ ] Close the plaintext 1883 listener outside localhost — it's kept open here only for local dev continuity.
- [ ] Enable Postgres server-side TLS at the infra layer and set `DATABASE_SSLMODE=require` (or `verify-full` once a real cert chain exists).
- [ ] Point `API_TLS_CERT_FILE`/`API_TLS_KEY_FILE` at the real cert, or terminate TLS at a load balancer/reverse proxy in front of the API instead — either is fine, just pick one deliberately.
