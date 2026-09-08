# ADR 003: Runtime operability boundaries

## Status

Accepted

## Context

The demo runtime needs a cheap liveness signal and a dependency-aware readiness signal without exposing database or Redis details. Startup failures must be bounded and diagnosable, and shutdown must stop accepting work before the process exits.

## Decision

- `/health` reports process liveness only.
- `/ready` checks PostgreSQL and Redis with a bounded timeout and returns only `ready` or `not_ready`.
- API startup uses a bounded initialization context and validates authentication configuration before accepting traffic.
- SIGINT and SIGTERM trigger a bounded HTTP graceful shutdown; incomplete shutdown is logged as an error.
- The Compose API image is multi-stage and runs as a non-root user. Compose waits for healthy PostgreSQL and Redis dependencies.
- `APP_MAINTENANCE_MODE=true` makes readiness unavailable and returns a bounded `503 MAINTENANCE_MODE` response for mutation methods with `Retry-After: 300`; liveness and read-only requests remain available.

## Consequences

Readiness can fail while liveness remains healthy, allowing an orchestrator or operator to distinguish process failure from dependency failure. The current Compose setup is a reproducible development/demo runtime, not a production SLA or Kubernetes deployment. Dynamic, audited maintenance control through a shared control plane is intentionally a follow-up; environment mode is the fail-safe for planned restarts and must be rolled out consistently across instances.

## Evidence

- `backend/cmd/api/router.go`
- `backend/cmd/api/main.go`
- `backend/Dockerfile`
- `docker-compose.yml`
- CI backend and browser jobs in `.github/workflows/ci.yml`
