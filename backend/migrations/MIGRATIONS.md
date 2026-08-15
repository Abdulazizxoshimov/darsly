# Migratsiya siyosati (B6)

golang-migrate, `up`/`down` juftlari. Server ishga tushganda `m.Up()` **avtomatik**
bajariladi (`internal/pkg/postgres/migrate.go`).

## Asosiy qoida: expand → migrate → contract

Deploy paytida qisqa fursat **eski kod + yangi sxema** birga ishlaydi (rolling
restart yoki eski konteyner hali tirik). Shuning uchun har migratsiya
**orqaga-mos** bo'lishi kerak — eski kod yangi sxemada yiqilmasligi shart.

**Buzadigan o'zgarishlarni ikki bosqichga bo'ling:**

| Amal | ❌ Bir bosqichda (xavfli) | ✅ Ikki bosqichda (xavfsiz) |
|---|---|---|
| Ustun o'chirish | `DROP COLUMN` | (1) kodni ustunni ishlatmaydigan qilib deploy → (2) keyingi relizda `DROP` |
| Ustun nomini o'zgartirish | `RENAME COLUMN` | (1) yangi ustun qo'shish + ikkalasiga yozish → (2) eski o'qishni ko'chirish → (3) eski ustunni tashlash |
| `NOT NULL` qo'shish | to'g'ridan-to'g'ri | (1) nullable qo'shish + default → (2) backfill → (3) `SET NOT NULL` |
| Ustun turi | joyida `ALTER TYPE` | yangi ustun + backfill + almashtirish |

**Qo'shish** (yangi jadval/ustun/indeks) odatda xavfsiz — eski kod ularni bilmaydi.
Katta jadvalga indeks qo'shishda `CREATE INDEX CONCURRENTLY` (lock'siz).

## Har migratsiya uchun

- **Har doim `.down.sql` yozing** va uni real sinang (`migrate down 1` → `up 1`).
- Migratsiyani **idempotent** qiling (`IF NOT EXISTS` / `IF EXISTS`).
- Bitta migratsiya = bitta mantiqiy o'zgarish (aралашtirmang).

## Rollback

golang-migrate startup'da faqat `Up` yuritadi. Orqaga qaytarish **qo'lda**:

```bash
# Konteyner ichida yoki DATABASE_URL bilan:
migrate -path ./migrations -database "$DATABASE_URL" down 1     # bitta qadam orqaga
migrate -path ./migrations -database "$DATABASE_URL" version    # joriy versiya
```

> ⚠️ `down` ma'lumot yo'qotishi mumkin (masalan `DROP COLUMN` ni qaytarish —
> ustun qaytadi, lekin ma'lumot yo'q). Shuning uchun **production'da rollback'dan
> ko'ra "oldinga tuzatuvchi migratsiya"** (yangi `up`) afzal. `down` asosan
> lokal/staging uchun.

## "Dirty" holat

Migratsiya o'rtada uzilsa `schema_migrations.dirty = true` bo'ladi va keyingi
`Up` rad etiladi. Tuzatish: muammoni qo'lda hal qilib, so'ng
`migrate -path ./migrations -database "$DATABASE_URL" force <VERSION>`.

## Deploy oldidan

- Migratsiya orqaga-mos ekaniga ishonch hosil qiling (yuqoridagi jadval).
- **DB backup oling** (`deploy/server/backup.sh`) — migratsiya kutilmaganda ketsa.
