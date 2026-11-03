# Runbook

**Readiness:** `/healthz` is liveness. `/readyz` pings PostgreSQL and Redis before accepting traffic. If Kafka is unavailable, reservations still commit and events remain pending in `outbox_events`; repair Kafka then restart the worker. Do not delete pending events to clear a backlog. Duplicate Kafka delivery is expected; consumers deduplicate event IDs. Investigate unexpected 409 responses with the tenant, idempotency key, and reservation ID without logging API keys.

## Startup and smoke check

Run `docker compose up --build` for the local stack. Migration must complete before API/worker start. Check `/readyz`, create an item, list its generated UUID, reserve with a unique key, replay that key, then release. Available stock must decrease once and return once. See README for example requests. Compose's worker restarts after startup failures while Kafka becomes ready.

## Failure triage

| Symptom | Check / response |
|---|---|
| 401 | Check configured key-to-tenant mapping; never print the key in logs |
| 400 | Inspect UUID, quantity/stock bounds, SKU/key length and strict JSON shape |
| 409 on reserve | Check stock and whether the same key was reused with a changed payload |
| 409 on confirm/release | Read the existing decision; completed reservations cannot transition again |
| 429 | Tenant exceeded the Redis window; allow expiry rather than resetting counters blindly |
| 503 / readiness failure | Check PostgreSQL and Redis connectivity before routing requests |
| Growing unpublished outbox | Check worker logs and Kafka listeners/topic; preserve pending rows for retry |

Metrics are exposed at `/metrics`. Production ingress must restrict probe/metric endpoints appropriately. Alert thresholds require workload measurement; no arbitrary latency/throughput SLO is asserted here.

## Persistence and recovery

PostgreSQL uses the Compose named volume. `docker compose down` preserves it; removing volumes destroys local data. Back up with PostgreSQL-aware tooling and test restoration before relying on a deployment. The schema file is initial setup rather than a general migration runner. A production rollout needs versioned forward migrations and a reviewed rollback/restore plan.

Do not use shared or production databases for integration tests: the fixture resets its configured schema. Tests expect disposable PostgreSQL/Redis/Kafka instances.

## Event recovery

Leases expire after a crash. Retrying may produce a duplicate if Kafka accepted an event before the worker lost acknowledgement. Consumers must deduplicate on the stable event ID; never mark unpublished rows as published merely to shrink a backlog. Compare event payload reservation IDs with persisted reservation state when investigating correlation problems.
