# stockledger-go implementation plan

> Agentic workers: use executing-plans, TDD and verification-before-completion; user delegated all routine decisions.

**Goal:** Implement the adjacent approved-by-delegation design.
**Architecture:** See spec. Keep domain independent and adapters replaceable.
**Tech Stack:** Go, PostgreSQL, Redis, Kafka
**Spec:** ../specs/2026-11-03-design.md

## Global constraints
- Never commit secrets or generated runtime data.
- Demonstrate behavior through real adapters; label any unexecuted checks.
- No deployment or paid external integrations.

## Review focus
Tenant bypass; duplicate/replayed mutations; simultaneous conflicting actions; invalid numeric boundaries; infrastructure failures and partial transactions.

## Task 1
- [ ] Reservation domain and small application ports; RED then GREEN for invalid quantities, transition rules and idempotency contracts.
- [ ] Observe relevant test failure, implement, run suite, record evidence.

## Task 2
- [ ] PostgreSQL persistence and migrations; RED then GREEN for atomic stock changes, tenant isolation, replay/conflict, rollback and outbox.
- [ ] Observe relevant test failure, implement, run suite, record evidence.

## Task 3
- [ ] HTTP adapter, API-key auth and Redis tenant rate limiting; test malformed input, unauthorized/cross-tenant access, errors and bounded bodies.
- [ ] Observe relevant test failure, implement, run suite, record evidence.

## Task 4
- [ ] Kafka outbox worker, metrics, configuration and shutdown; verify retry and duplicate-delivery contract.
- [ ] Observe relevant test failure, implement, run suite, record evidence.

## Task 5
- [ ] Compose, CI integration, OpenAPI, architecture/ADRs and runbook; run all checks and independent final review.
- [ ] Observe relevant test failure, implement, run suite, record evidence.
