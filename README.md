# StockLedger

StockLedger is a tenant-isolated inventory reservation API. PostgreSQL is the source of truth: each reservation, stock decrement, idempotency record, and outbox event are committed in one transaction. Redis is only a per-tenant rate limiter. Kafka is fed by a retrying transactional-outbox worker, giving at-least-once delivery; consumers must deduplicate on event ID.

## Run

Start the complete local stack with `docker compose up --build`. It runs migrations, API, worker, PostgreSQL, Redis, and Kafka. The API is at [localhost:8080](http://localhost:8080); use `X-API-Key: local-key`. For host-process development, set `DATABASE_URL`, `REDIS_ADDR=localhost:6379`, `KAFKA_BROKERS=localhost:9092`, and `API_KEYS=local-key:demo-tenant`, then run `go run ./cmd/stockledger` and `go run ./cmd/outbox-worker`. Containers use Kafka's internal `kafka:29092` listener. Compose credentials and API keys are public local-development examples; replace them before any network deployment. PostgreSQL data uses a named volume.

## Verify

`go test -race ./...`, `go vet ./...`, and (with PostgreSQL) `DATABASE_URL=... go test -tags=integration -race ./internal/postgres`. CI runs these on Linux; its service-backed integration check is not claimed as locally executed.

## Boundaries

This is a portfolio service, not a production deployment. It has no key rotation, TLS termination, consumer implementation, database migration framework, or distributed tracing. See `docs/runbook.md` and `docs/threat-model.md`.

## Try the workflow

```bash
curl -i http://localhost:8080/v1/items \
  -H 'X-API-Key: local-key' -H 'Content-Type: application/json' \
  -d '{"sku":"design-desk","available":5}'
curl http://localhost:8080/v1/items -H 'X-API-Key: local-key'
```

Copy the item's `id` from the list, then reserve:

```bash
curl -i http://localhost:8080/v1/reservations \
  -H 'X-API-Key: local-key' -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: checkout-001' \
  -d '{"item_id":"YOUR-ITEM-UUID","quantity":2}'
```

The same key and payload replay without another decrement. A changed payload conflicts. Confirm or release with `POST /v1/reservations/{id}/confirm` or `/release`, using the API-key header. Completed reservations are terminal.

## Engineering evidence

- Atomic stock decrement and row-locked transitions prevent overselling and duplicate restoration.
- Tenant-scoped idempotency, reservation and outbox writes share one transaction.
- Redis uses an atomic increment/expiry script and never stores authoritative stock.
- The worker leases one event, publishes to Kafka and acknowledges only its own lease. Failed delivery is retried; stable IDs support consumer deduplication.
- HTTP validation, classified errors, health/readiness, Prometheus metrics and shutdown handling have focused owners.
- Unit tests and service-backed integration tests exercise concurrent reservations, duplicate keys, transition races, tenant isolation and event correlation.

Read the [architecture/SOLID mapping](docs/architecture.md), ADRs in `docs/adr/`, [OpenAPI contract](openapi.yaml), [runbook](docs/runbook.md), [threat model](docs/threat-model.md) and [validation record](docs/test-evidence.md). The [LinkedIn Türkiye report](docs/research/linkedin-turkiye-2026-11-03.md) includes exact URLs, per-posting coding and required/preferred distinctions. It includes closed/cached postings and is **not an open-vacancy census**.

Use the Go version in `go.mod`. Additional checks: `go build ./...` and `govulncheck ./...`. Full integrations use `go test -tags=integration -race ./internal/...` against explicitly disposable services. The PostgreSQL fixture resets its configured schema; never point it at shared/production data. CI provisions disposable services. No production traffic or load benchmark is claimed.
