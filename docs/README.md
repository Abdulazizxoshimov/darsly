# Darsly — hujjatlar

Bu katalogda **faqat barqaror ma'lumot** turadi: reja va todo ro'yxatlari emas,
o'zgarmaydigan yoki sekin o'zgaradigan bilim.

## Nima qayerda

| Fayl | Nima uchun |
|---|---|
| [BACKLOG.md](BACKLOG.md) | **Ochiq ishlar** — yagona ro'yxat. Yangi ish shu yerga yoziladi |
| [LAUNCH-PLAN.md](LAUNCH-PLAN.md) | Ishga tushirish rejasi — fazalar (P0/P1/P2), audit natijasi |
| [PRODUCT.md](PRODUCT.md) | Mahsulot qarorlari (asoschi intervyusi asosida) |
| [api-contract.md](api-contract.md) | API kontrakti — frontend/mobil uchun yagona haqiqat manbai |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Infratuzilma va **masshtab** talablari (PgBouncer, multi-node LiveKit, Redis HA) |
| [PRIVACY.md](PRIVACY.md) | Maxfiylik siyosati (Play Store majburiyati, M5) |
| [telegram-setup.md](telegram-setup.md) | Telegram arxiv integratsiyasini sozlash |

Boshqa joylarda:

- **`CLAUDE.md`** (ildizda) — loyiha manuali: tuzilma, portlar, arxitektura qoidalari
- **`deploy/README.md`** — qaysi deploy stacki kanonik
- **`deploy/DEPLOY-CHECKLIST.md`** — deploy qadamlari va keyingi majburiy tekshiruvlar
- **`mobile/README.md`** — mobil ilova tafsilotlari

## Nega bu katalog kichraydi

Ilgari bu yerda 22 ta fayl bor edi va ularning ko'pi **reja/todo** hujjatlari edi:
`mvp-todo`, `OPTIMAL-PLAN`, `PERFECTION_PLAN`, `OPTIMIZATION_PLAN`,
`PRODUCTION-READINESS`, `MOBILE-STATUS`, `darsly-mobile-plan`,
`darsly-mobile-roadmap`, `darsly-backend-plan`, `final-report`,
`orchestration-log`, `PROJECT_STRUCTURE`.

Muammo shundaki, ular **eskirgan** edi: 2026-07-28 dagi tekshiruvda
`PRODUCTION-READINESS.md` ning ochiq bandlaridan bir nechtasi allaqachon
bajarilgan bo'lib chiqdi (dublikat CI fayli o'chirilgan, `.gitignore` bor,
git remote ulangan), `mvp-todo.md` ning esa yarmi audit ishida yopilgan edi.

Eskirgan ro'yxat — foydasizdan ham yomon: unga qarab qaror qabul qilinadi.
Shuning uchun ular o'chirildi (git tarixida qoladi), haqiqiy ochiq ishlar esa
tekshirilib [BACKLOG.md](BACKLOG.md) ga ko'chirildi.

Keyingi tozalashda (2026-08) yana bir nechta eskirgan/tarixiy hujjat o'chirildi
(`acceptance-criteria.md`, `mobile-acceptance-criteria.md`, `load-test-report.md`,
`MOBILE-DEVICE-TEST.md`, `api-test-coverage.md`, `darsly-frontend-prompt.md`) —
ular git tarixida qoladi. Bu jadval endi faqat **mavjud** fayllarni ko'rsatadi.

**Qoida:** yangi reja hujjati yaratmang. Ish — `BACKLOG.md` ga, barqaror bilim —
shu jadvaldagi tegishli faylga yoki `CLAUDE.md` ga.
