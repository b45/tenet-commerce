#!/usr/bin/env bash
# ==============================================================================
# Tenet Commerce - Safe Demo Database Reset Script
# Guarded reset requiring explicit demo environment confirmation
# ==============================================================================
set -euo pipefail

TARGET_CONTAINER="${TENET_POSTGRES_CONTAINER:-tenet_postgres}"
TARGET_DB="${TENET_POSTGRES_DB:-tenet_commerce}"
INIT_SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/init_dev_db.sql"

# Safety Check 1: Target DB must be explicitly named tenet_commerce or contain demo/test
if [[ "$TARGET_DB" != "tenet_commerce" && "$TARGET_DB" != *"demo"* && "$TARGET_DB" != *"test"* ]]; then
    echo "ERROR: Unsafe target database name: '$TARGET_DB'." >&2
    echo "Reset is strictly restricted to local demo or test databases." >&2
    exit 1
fi

# Safety Check 2: Require explicit confirmation variable or user prompt
if [[ "${CONFIRM_DEMO_RESET:-}" != "true" && "${CONFIRM_DEMO_RESET:-}" != "1" ]]; then
    if [[ -t 0 ]]; then
        read -r -p "WARNING: This will reset and re-seed '$TARGET_DB' on container '$TARGET_CONTAINER'. Proceed? [y/N] " response
        if [[ "$response" != "y" && "$response" != "Y" ]]; then
            echo "Reset aborted by user."
            exit 0
        fi
    else
        echo "ERROR: Reset rejected. Automated reset requires CONFIRM_DEMO_RESET=true." >&2
        exit 1
    fi
fi

if [[ ! -f "$INIT_SCRIPT" ]]; then
    echo "ERROR: Canonical init script not found at '$INIT_SCRIPT'." >&2
    exit 1
fi

echo "==> Resetting database '$TARGET_DB' on container '$TARGET_CONTAINER'..."
docker exec -i "$TARGET_CONTAINER" psql -U postgres -d "$TARGET_DB" < "$INIT_SCRIPT"
echo "==> Database '$TARGET_DB' reset and seeded with deterministic demo fixtures successfully!"
