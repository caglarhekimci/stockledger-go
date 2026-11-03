# StockLedger

StockLedger is a tenant-isolated inventory reservation API. PostgreSQL is the source of truth: each reservation, stock decrement, idempotency record, and outbox event are committed in one transaction. Redis is only a per-tenant rate limiter. Kafka is fed by a retrying transactional-outbox worker, giving at-least-once delivery; consumers must deduplicate on event ID.

## Run

Start dependencies with `docker compose up -d`, apply `migrations/001_init.sql` with `psql "$DATABASE_URL" -f migrations/001_init.sql`, then set `DATABASE_URL`, `REDIS_ADDR`, and `API_KEYS=local-key:demo-tenant`. Run `go run ./cmd/stockledger`. This repository commits no usable credentials; compose values are local development defaults only.

## Verify

`go test -race ./...`, `go vet ./...`, and (with PostgreSQL) `DATABASE_URL=... go test -tags=integration -race ./internal/postgres`. CI runs these on Linux; its service-backed integration check is not claimed as locally executed.

## Boundaries

This is a portfolio service, not a production deployment. It has no key rotation, TLS termination, consumer implementation, database migration framework, or distributed tracing. See `docs/runbook.md` and `docs/threat-model.md`.
