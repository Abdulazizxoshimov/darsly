# Darsly — hujjatlar

Bu katalogda **faqat barqaror ma'lumot** turadi: reja va todo ro'yxatlari emas,
o'zgarmaydigan yoki sekin o'zgaradigan bilim.

## Nima qayerda

| Fayl | Nima uchun |
|---|---|
| [BACKLOG.md](BACKLOG.md) | **Ochiq ishlar** — yagona ro'yxat. Yangi ish shu yerga yoziladi |
| [api-contract.md](api-contract.md) | API kontrakti — frontend/mobil uchun yagona haqiqat manbai |
| [api-test-coverage.md](api-test-coverage.md) | Qaysi endpoint qaysi test bilan qoplangan |
| [acceptance-criteria.md](acceptance-criteria.md) | Mahsulot qabul mezonlari (web) |
| [mobile-acceptance-criteria.md](mobile-acceptance-criteria.md) | Mobil qabul mezonlari |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Infratuzilma va **masshtab** talablari (PgBouncer, multi-node LiveKit, Redis HA) |
| [load-test-report.md](load-test-report.md) | Yuklama sinovi o'lchovlari (2026-07-24) — tarixiy yozuv |
| [darsly-frontend-prompt.md](darsly-frontend-prompt.md) | Frontend arxitektura spetsifikatsiyasi (manba talab) |
| [MOBILE-DEVICE-TEST.md](MOBILE-DEVICE-TEST.md) | Qurilma sinovi yozuvi: **o'lchov metodikasi**, platforma cheklovlari, takrorlanmasligi kerak bo'lgan xatolar |

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

`MOBILE-DEVICE-TEST.md` esa ATAYLAB saqlandi: u todo emas — o'lchov usullari,
platforma cheklovlari va "o'lchovni buzgan omillar" yozilgan empirik yozuv.
Bunday bilim qayta topilmaydi, faqat qayta xato qilish orqali o'rganiladi.

**Qoida:** yangi reja hujjati yaratmang. Ish — `BACKLOG.md` ga, barqaror bilim —
shu jadvaldagi tegishli faylga yoki `CLAUDE.md` ga.
