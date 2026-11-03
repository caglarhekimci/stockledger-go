# Test evidence

2026-11-03 RED: `go test ./internal/domain` failed with undefined `NewReservation`, `ErrInvalidQuantity`, and `ErrInvalidTransition`. After the minimal domain implementation, it passed.

2026-11-03 RED: after dependency resolution, `go test ./internal/...` failed because `Item`, `ReserveInput`, `Service`, and `ErrIdempotencyKeyRequired` did not exist. After application service implementation, internal tests passed.

2026-11-03 GREEN: `go test ./...`, `go vet ./...`, `go build ./...`, and `git diff --check` completed with exit code 0. `go test -tags=integration -run '^$' ./internal/...` compiled every service-backed test package without running services.

`go test -race ./...` could not run locally because this Windows environment has no C compiler and Go reported `-race requires cgo; enable cgo by setting CGO_ENABLED=1`. PostgreSQL, Redis, and Kafka containers were unavailable locally, so their behavior tests are provided for Linux CI and are not represented as locally executed.

2026-11-03 review-fix regression coverage adds: synchronized same-key reservation replay with one stock decrement/outbox event and payload correlation; concurrent release and confirm-vs-release state-transition tests; real Redis tenant/TTL behavior; Kafka stable-key publication; and an outbox worker test proving failure is not acknowledged and a later pass succeeds. These real-service tests compile locally under the `integration` tag and are run by Linux CI; they were not executed against services locally.

Final review-fix coverage adds a worker lease-budget regression: it asserts sequential processing claims exactly one event, so the 5-second publish timeout remains inside the 30-second lease regardless of backlog size. PostgreSQL integration tests invoke the JSON outbox expressions against a real database in CI.

2026-11-03 RED/GREEN: the HTTP malformed/range-input test initially received `201` for out-of-range stock; after transport validation it passes. The snake_case response contract test initially returned Go field names such as `ItemID`; after explicit JSON tags it passes.

After compatible dependency upgrades, `govulncheck ./...` reported zero reachable vulnerabilities; it reported one vulnerability in a required module that this code does not call.

Primary verification (2026-11-03, Go1.27.3 Windows): full unit suite, vet, build and pinned govulncheck v1.9.0 passed after final adapter/validation fixes. Vulnerability scan found0 reachable and1 in a required but uncalled module. CI uses current1.27.x patches and exercises the real service tests on Linux. Independent review findings were fixed with regression coverage; local Docker/race limits remain as above.
