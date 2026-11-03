# Test evidence

2026-11-03 RED: `go test ./internal/domain` failed with undefined `NewReservation`, `ErrInvalidQuantity`, and `ErrInvalidTransition`. After the minimal domain implementation, it passed.

2026-11-03 RED: after dependency resolution, `go test ./internal/...` failed because `Item`, `ReserveInput`, `Service`, and `ErrIdempotencyKeyRequired` did not exist. After application service implementation, internal tests passed.

2026-11-03 GREEN: `go test ./...`, `go vet ./...`, `go build ./...`, and `git diff --check` completed with exit code 0. `go test -tags=integration -run '^$' ./internal/...` compiled every service-backed test package without running services.

`go test -race ./...` could not run locally because this Windows environment has no C compiler and Go reported `-race requires cgo; enable cgo by setting CGO_ENABLED=1`. PostgreSQL, Redis, and Kafka containers were unavailable locally, so their behavior tests are provided for Linux CI and are not represented as locally executed.
