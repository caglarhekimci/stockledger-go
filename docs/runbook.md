# Runbook

**Readiness:** `/healthz` is liveness. `/readyz` pings PostgreSQL and Redis before accepting traffic. If Kafka is unavailable, reservations still commit and events remain pending in `outbox_events`; repair Kafka then restart the worker. Do not delete pending events to clear a backlog. Duplicate Kafka delivery is expected; consumers deduplicate event IDs. Investigate unexpected 409 responses with the tenant, idempotency key, and reservation ID without logging API keys.
