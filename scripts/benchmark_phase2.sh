#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
# No configurable target URL: TestMain always creates disposable dependencies.
OUTPUT_DIR="${1:-${ROOT_DIR}/docs/local/benchmark-evidence}"
mkdir -p "${OUTPUT_DIR}"
OUTPUT_DIR="$(cd "${OUTPUT_DIR}" && pwd)"
git -C "${ROOT_DIR}" rev-parse HEAD > "${OUTPUT_DIR}/commit.txt"
git -C "${ROOT_DIR}" diff --binary HEAD > "${OUTPUT_DIR}/candidate.patch"
git -C "${ROOT_DIR}" status --short > "${OUTPUT_DIR}/worktree.txt"
uname -sm > "${OUTPUT_DIR}/platform.txt"
go version >> "${OUTPUT_DIR}/platform.txt"
if command -v sysctl >/dev/null 2>&1; then
  sysctl -n machdep.cpu.brand_string hw.memsize >> "${OUTPUT_DIR}/platform.txt" 2>/dev/null || true
fi
shasum -a 256 "${ROOT_DIR}/backend/integration/benchmark_evidence_test.go" "${ROOT_DIR}/backend/integration/infrastructure_test.go" "${ROOT_DIR}/scripts/init_dev_db.sql" > "${OUTPUT_DIR}/source-sha256.txt"
printf 'postgres_tmpfs=%s\n' "${TENET_TEST_POSTGRES_TMPFS:-0}" >> "${OUTPUT_DIR}/platform.txt"
cd "${ROOT_DIR}/backend"
for attempt in 1 2; do
  go test -json ./integration -run '^TestBenchmarkEvidence$' -count=1 > "${OUTPUT_DIR}/run-${attempt}.jsonl"
done
printf 'Evidence saved to %s\n' "${OUTPUT_DIR}"
