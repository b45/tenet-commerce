#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Tenet Commerce — Phase 2 Real-Database Benchmark Harness
# Exercises full ACID POS checkout transaction throughput against real
# PostgreSQL 16 and Redis 7 testcontainers.
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "========================================================================"
echo " Tenet Commerce: Phase 2 Real-Database Transaction Benchmark"
echo "========================================================================"
echo "Target: Full POS Checkout Flow (Redis SetNX + PG Row Lock + Stock"
echo "        Decrement + POS TXN + Double-Entry Ledger + Balance Trigger + WAL)"
echo "------------------------------------------------------------------------"

cd "${ROOT_DIR}/backend"

# Run Benchmark with 5-second duration and memory allocations report
go test -v ./integration \
  -run "^$" \
  -bench "BenchmarkE2E_RealDatabase_POSCheckout" \
  -benchtime=5s \
  -count=1

echo "========================================================================"
echo " Benchmark completed successfully."
echo "========================================================================"
