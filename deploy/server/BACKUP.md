# Darsly — DB backup & tiklash (P0-3)

RPO'ni ∞ dan **~24 soatga** tushiradi. Ikki skript:

| Skript | Vazifa |
|---|---|
| `backup.sh` | `darsly-postgres` → `pg_dump -Fc` → MinIO `darsly-backups` bucket. Retention: lokal 14 nusxa, MinIO 30 kun. |
| `backup-restore-test.sh` | Backup'ni vaqtinchalik `darsly_restore_test` bazasiga tiklaydi va tekshiradi (asosiy bazaga tegmaydi). |

## O'rnatish

`remote-setup.sh` **avtomatik** kunlik cron o'rnatadi (03:17 UTC):
```
17 3 * * *  /root/darsly/deploy/server/backup.sh >> /var/log/darsly-backup.log 2>&1
```
Qo'lda tekshirish: `crontab -l | grep backup.sh`

## Qo'lda backup olish
```bash
cd /root/darsly/deploy/server
./backup.sh
```

## ⚠️ Tiklashni sinash — MAJBURIY

**Sinalmagan backup — backup emas.** Deploy'dan keyin kamida bir marta:
```bash
./backup-restore-test.sh              # MinIO'dagi eng so'nggini tiklab tekshiradi
./backup-restore-test.sh backups/darsly-20260815-031700.dump   # aniq fayl
```
Natijada jadval soni + `users` yozuvlar soni chiqadi, so'ng vaqtinchalik baza o'chadi.
**Bu qadam bajarilmaguncha P0-3 yopilmagan hisoblanadi.**

## Halokatdan real tiklash (production DB)
```bash
# 1) Backup'ni oling
docker run --rm --network darsly_net -v "$PWD:/out" \
  -e MC_HOST_local="http://$KEY:$SECRET@minio:9000" --entrypoint sh minio/mc \
  -c "mc cp local/darsly-backups/darsly-<STAMP>.dump /out/restore.dump"

# 2) Backend'ni to'xtating (yozuvlar to'xtasin)
docker compose stop backend

# 3) Bazani tiklang (mavjud ma'lumot ustiga — EHTIYOT BO'LING)
docker exec -i darsly-postgres pg_restore -U darsly -d darsly --clean --if-exists < restore.dump

# 4) Backend'ni qayta ishga tushiring
docker compose start backend
```
> `--clean --if-exists` mavjud obyektlarni tashlab, qaytadan yaratadi. Butunlay yangi
> serverga tiklashda esa avval `CREATE DATABASE darsly;` qilib, `--clean`siz tiklang.
