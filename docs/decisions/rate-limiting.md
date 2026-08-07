# Decision record: rate limiting is single-instance only

## Status

Accepted, as of Phase 4. This is a documented, permanent limitation for
this project's phased plan — there is no Phase 5 to defer it to.

## Context

`internal/httpapi/router.go` rate-limits `/auth/login` and `POST /devices`
using Echo's built-in `middleware.RateLimiter` backed by
`NewRateLimiterMemoryStore`. That store is a plain in-process map: each
running API process has its own independent bucket.

This has been flagged since Phase 2/3 as a known gap, and Phase 4's own
load test confirmed the registration limiter is real and effective — a
provisioning script that ignored it started failing within the first
second of bulk device registration (see `docs/load-test-results.md`).

## The actual gap

If this API is ever run as more than one replica behind a load balancer,
each replica enforces the limit independently — an attacker (or a buggy
client) distributed across replicas effectively gets `limit × replica
count` requests through, not the configured limit.

## Decision

**Do not fix this now.** The standard fix is a shared, external rate-limit
store (Redis or equivalent) that every replica reads/writes to. That is a
new infrastructure dependency this project has not confirmed it needs —
CLAUDE.md is explicit that a new external/SaaS dependency requires
justification, not speculative addition.

This deployment, as scoped through Phase 4, runs a single API instance.
The in-memory limiter is **correct and sufficient** for that shape. The
moment a real decision is made to run more than one API replica (for
availability or throughput), that is the trigger to revisit this record
and add a shared-store limiter — not before.

## What would change this decision

- A confirmed requirement to run 2+ API replicas (availability target,
  throughput need discovered via further load testing, etc.)
- At that point: introduce a Redis-backed (or equivalent) limiter store,
  update `router.go`'s `middleware.RateLimiter` construction, and add
  Redis to `docker-compose.yml` / real infra — a small, well-scoped change
  once the need is real, not before.
