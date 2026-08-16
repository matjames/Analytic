#!/bin/bash
# ═══════════════════════════════════════════════════════════════════
# STATGATE AUTOMATED DATABASE BACKUP SCRIPT (Linux/Docker)
# ═══════════════════════════════════════════════════════════════════

set -euo pipefail

BACKUP_DIR="${1:-./backups}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-Kaggle}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
TARGET_DIR="${BACKUP_DIR}/${TIMESTAMP}"

mkdir -p "${TARGET_DIR}"

echo "═══════════════════════════════════════════════════════════"
echo "  STATGATE MULTI-DATABASE BACKUP ENGINE"
echo "  Timestamp: ${TIMESTAMP}"
echo "═══════════════════════════════════════════════════════════"

DATABASES=("statgate_ml_staging" "statgate" "pms" "rms" "statchat" "statgate_enterprise")

for db in "${DATABASES[@]}"; do
    DUMP_FILE="${TARGET_DIR}/${db}_${TIMESTAMP}.sql.gz"
    echo " [BACKUP] Dumping database: ${db} -> ${DUMP_FILE}"
    PGPASSWORD="${KAGGLE_DB_PASSWORD:-}" pg_dump -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${db}" | gzip > "${DUMP_FILE}" || true
    if [ -f "${DUMP_FILE}" ]; then
        echo "   -> OK: $(du -h "${DUMP_FILE}" | cut -f1)"
    fi
done

echo "═══════════════════════════════════════════════════════════"
echo "  BACKUP SEQUENCE COMPLETE: ${TARGET_DIR}"
echo "═══════════════════════════════════════════════════════════"
