#!/usr/bin/env bash
# Darsly — PostgreSQL backup → MinIO (P0-3, RPO'ni ∞ dan ~24 soatga tushiradi).
#
# Nima qiladi:
#   1) `darsly-postgres` konteyneridan `pg_dump -Fc` (custom format — pg_restore
#      bilan tiklanadi, plain SQL'dan ishonchliroq va siqilgan).
#   2) Natijani MinIO'ning ALOHIDA `darsly-backups` bucket'iga yuklaydi
#      (asosiy `darsly` bucket'i — yozuvlar; backup boshqa joyda tursin).
#   3) Retention: lokalda oxirgi 14 nusxa, MinIO'da 30 kundan eskisi o'chiriladi.
#
# ⚠️ Sinalmagan backup — backup EMAS. Har o'zgarishdan keyin `backup-restore-test.sh`
#    ni ishga tushirib, fayl haqiqatan tiklanishini tekshiring.
#
# Cron (remote-setup.sh o'rnatadi): har kuni 03:17 da.
#   17 3 * * *  /root/darsly/deploy/server/backup.sh >> /var/log/darsly-backup.log 2>&1
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

[[ -f .env ]] || { echo "!! .env yo'q ($DIR)"; exit 1; }

# .env dan faqat kerakli kalitlarni xavfsiz o'qiymiz (butun faylni source qilmasdan).
env_get() { grep "^$1=" .env | head -1 | cut -d= -f2-; }
MINIO_ACCESS_KEY="$(env_get MINIO_ACCESS_KEY)"
MINIO_SECRET_KEY="$(env_get MINIO_SECRET_KEY)"
[[ -n "$MINIO_ACCESS_KEY" && -n "$MINIO_SECRET_KEY" ]] || { echo "!! .env da MinIO kaliti yo'q"; exit 1; }

PG_CONTAINER="darsly-postgres"
NET="darsly_net"
BACKUP_BUCKET="darsly-backups"
LOCAL_DIR="$DIR/backups"
LOCAL_KEEP=14
REMOTE_KEEP_DAYS=30

mkdir -p "$LOCAL_DIR"
STAMP="$(date -u +%Y%m%d-%H%M%S)"
FILE="darsly-${STAMP}.dump"
OUT="$LOCAL_DIR/$FILE"

echo ">> [$(date -u +%FT%TZ)] pg_dump boshlandi → $FILE"
# -Fc: custom format, ichida siqilgan. Konteyner ichida bajariladi (host'da pg kerak emas).
docker exec "$PG_CONTAINER" pg_dump -U darsly -d darsly -Fc > "$OUT"

SIZE=$(stat -c%s "$OUT" 2>/dev/null || echo 0)
[[ "$SIZE" -gt 1000 ]] || { echo "!! dump juda kichik ($SIZE bayt) — buzuq bo'lishi mumkin"; exit 1; }
echo ">> dump tayyor: $(du -h "$OUT" | cut -f1)"

# MinIO'ga yuklash + retention — mc'ni bir martalik konteynerda ishlatamiz.
echo ">> MinIO'ga yuklanmoqda ($BACKUP_BUCKET)..."
docker run --rm --network "$NET" \
  -v "$LOCAL_DIR:/backup:ro" \
  -e MC_HOST_local="http://${MINIO_ACCESS_KEY}:${MINIO_SECRET_KEY}@minio:9000" \
  --entrypoint sh minio/mc -c "
    set -e
    mc mb -p local/${BACKUP_BUCKET} 2>/dev/null || true
    mc cp /backup/${FILE} local/${BACKUP_BUCKET}/${FILE}
    mc rm --recursive --force --older-than ${REMOTE_KEEP_DAYS}d local/${BACKUP_BUCKET}/ 2>/dev/null || true
  "
echo ">> MinIO'ga yuklandi: $BACKUP_BUCKET/$FILE"

# Lokal retention — oxirgi $LOCAL_KEEP nusxa qoladi.
ls -1t "$LOCAL_DIR"/darsly-*.dump 2>/dev/null | tail -n +$((LOCAL_KEEP + 1)) | while read -r old; do
  rm -f "$old" && echo ">> eski lokal nusxa o'chirildi: $(basename "$old")"
done

echo ">> [$(date -u +%FT%TZ)] backup TUGADI ✓"
