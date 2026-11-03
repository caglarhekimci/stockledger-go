# ADR 0001: transactional outbox

Reservation state and an outbox event are written in the same PostgreSQL transaction. Publishing directly to Kafka would permit a committed reservation with a lost event, or an event for a rolled-back reservation. Workers atomically lease bounded batches; a lease expires after a crash, failed publication delays the next attempt, and acknowledgement requires the lease token. Delivery is at least once and consumers must deduplicate by stable event ID.
