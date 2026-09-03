# Database bootstrap source

`init_dev_db.sql` is the canonical baseline schema and development fixture for this repository. It is the only SQL script run by Docker Compose, `make db-reset`, and the Testcontainers integration suite.

The numbered SQL files remain as historical implementation artifacts. They must not be added to a bootstrap command: their effects have already been consolidated into the canonical baseline. New schema changes must be introduced as versioned migrations as part of the Phase 2 hardening work; do not copy schema fragments into test fixtures.
