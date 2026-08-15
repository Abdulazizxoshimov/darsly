#!/usr/bin/env bash
# Darsly — backup TIKLASH sinovi (P0-3). "Sinalmagan backup — backup emas."
#
# Nima qiladi:
#   1) Berilgan dump faylni (argument) yoki MinIO'dagi ENG SO'NGGISINI oladi.
#   2) `darsly-postgres` ichida VAQTINCHALIK `darsly_restore_test` bazasiga tiklaydi
#      (ASOSIY `darsly` bazasiga TEGMAYDI).
#   3) Sanity: jadval soni + `users` yozuvlar sonini chiqaradi.
#   4) Vaqtinchalik bazani o'chiradi.
#
# Ishlatish:
#   ./backup-restore-test.sh                      # MinIO'dagi eng so'nggi backup
#   ./backup-restore-test.sh backups/darsly-*.dump  # aniq lokal fayl
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"
[[ -f .env ]] || { echo "!! .env yo'q"; exit 1; }
env_get() { grep "^$1=" .env | head -1 | cut -d= -f2-; }

PG_CONTAINER="darsly-postgres"
NET="darsly_net"
BACKUP_BUCKET="darsly-backups"
TEST_DB="darsly_restore_test"

DUMP="${1:-}"
TMP=""
if [[ -z "$DUMP" ]]; then
  echo ">> MinIO'dan eng so'nggi backup olinmoqda..."
  MINIO_ACCESS_KEY="$(env_get MINIO_ACCESS_KEY)"; MINIO_SECRET_KEY="$(env_get MINIO_SECRET_KEY)"
  TMP="$(mktemp -d)"
  LATEST="$(docker run --rm --network "$NET" \
    -e MC_HOST_local="http://${MINIO_ACCESS_KEY}:${MINIO_SECRET_KEY}@minio:9000" \
    --entrypoint sh minio/mc -c \
    "mc ls local/${BACKUP_BUCKET}/ | awk '{print \$NF}' | sort | tail -1")"
  [[ -n "$LATEST" ]] || { echo "!! MinIO'da backup topilmadi"; exit 1; }
  echo ">> eng so'nggi: $LATEST"
  docker run --rm --network "$NET" -v "$TMP:/out" \
    -e MC_HOST_local="http://${MINIO_ACCESS_KEY}:${MINIO_SECRET_KEY}@minio:9000" \
    --entrypoint sh minio/mc -c "mc cp local/${BACKUP_BUCKET}/${LATEST} /out/dump"
  DUMP="$TMP/dump"
fi
[[ -f "$DUMP" ]] || { echo "!! dump fayl topilmadi: $DUMP"; exit 1; }
echo ">> tiklanadigan fayl: $DUMP ($(du -h "$DUMP" | cut -f1))"

cleanup() {
  docker exec "$PG_CONTAINER" psql -U darsly -d darsly -c "DROP DATABASE IF EXISTS $TEST_DB;" >/dev/null 2>&1 || true
  [[ -n "$TMP" ]] && rm -rf "$TMP"
}
trap cleanup EXIT

echo ">> vaqtinchalik baza yaratilmoqda: $TEST_DB"
docker exec "$PG_CONTAINER" psql -U darsly -d darsly -c "DROP DATABASE IF EXISTS $TEST_DB;" >/dev/null
docker exec "$PG_CONTAINER" psql -U darsly -d darsly -c "CREATE DATABASE $TEST_DB;" >/dev/null

echo ">> tiklanmoqda (pg_restore)..."
START=$(date +%s)
docker exec -i "$PG_CONTAINER" pg_restore -U darsly -d "$TEST_DB" --no-owner < "$DUMP"
ELAPSED=$(( $(date +%s) - START ))

TABLES="$(docker exec "$PG_CONTAINER" psql -U darsly -d "$TEST_DB" -tAc \
  "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';")"
USERS="$(docker exec "$PG_CONTAINER" psql -U darsly -d "$TEST_DB" -tAc \
  "SELECT count(*) FROM users;" 2>/dev/null || echo 'N/A')"

echo ""
echo "==================== TIKLASH SINOVI NATIJASI ===================="
echo "  Fayl:            $(basename "$DUMP")"
echo "  Tiklash vaqti:   ${ELAPSED}s"
echo "  Jadvallar (public): $TABLES"
echo "  users yozuvlari:    $USERS"
echo "================================================================"
[[ "$TABLES" -gt 0 ]] || { echo "!! 0 jadval — tiklash MUVAFFAQIYATSIZ"; exit 1; }
echo ">> TIKLASH ISHLADI ✓  (vaqtinchalik baza o'chirilmoqda)"
