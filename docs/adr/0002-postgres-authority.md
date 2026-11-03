# ADR 0002: PostgreSQL owns stock

Redis is deliberately excluded from inventory reads and writes. Row-level conditional updates in PostgreSQL prevent negative stock and provide durable tenant-scoped idempotency.
