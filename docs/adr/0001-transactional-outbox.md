# ADR 0001: transactional outbox

Reservation state and an outbox event are written in the same PostgreSQL transaction. Publishing directly to Kafka would permit a committed reservation with a lost event, or an event for a rolled-back reservation. The worker publishes at least once and consumers must deduplicate by stable event ID.
