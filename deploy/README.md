# Darsly — deployment stacklari

Repoda **uchta** docker-compose stacki bor va bu auditda (M17) konfiguratsiya
drifti xavfi sifatida belgilangan: bir joyda tuzatilgan narsa boshqasida
tuzatilmasdan qolardi. Quyida qaysi biri nima uchun ekani va **qaysi biri
kanonik** ekani qayd etilgan.

## 1. `deploy/server/` — KANONIK ✅

Hozir ishlatiladigan yagona stack. Test/staging serveri (Contabo VPS) shu bilan
ko'tarilgan va **barcha yangi o'zgarishlar shu yerga qo'shiladi**.

- O'zi yetarli: postgres + redis + minio + backend + livekit + egress + caddy
  + prometheus + grafana + loki
- Domensiz HTTPS: `sslip.io` + Caddy avtomatik Let's Encrypt
- Sirlar serverda `remote-setup.sh` bilan yasaladi (git'ga tushmaydi)
- Kuzatuv: `observability/` (M16) — Grafana faqat `127.0.0.1:3030`, SSH tunnel orqali

```bash
ssh root@<IP>
cd /opt/darsly && ./remote-setup.sh && docker compose up -d --build
# Grafana: ssh -L 3030:localhost:3030 root@<IP>  → http://localhost:3030
```

## 2. `backend/deployments/` — faqat `Dockerfile` ishlatiladi ⚠️

`Dockerfile` **kerak**: kanonik stack uni build uchun ishlatadi
(`context: ../../backend`, `dockerfile: deployments/Dockerfile`). Unga tegmang.

`docker-compose.yml`, `grafana/`, `loki/`, `nginx/` esa **eskirgan**: bular
domenli (`*.darsly.uz`), ikkiga bo'lingan deployment uchun yozilgan muqobil
variant. Ular ishlab turgan serverda ISHLATILMAYDI.

Real domenga o'tilganda bu variant asos bo'lishi mumkin, lekin unda ham
kanonik stack yangilanishlari (xavfsizlik sarlavhalari, healthcheck,
kuzatuv, `METRICS_TOKEN`) ko'chirilishi shart.

## 3. `services/livekit/` — LiveKit konfiguratsiyasi 📎

`livekit.yaml` / `egress.yaml` shablonlari va **lokal dev** uchun compose.
`docker-compose.prod.yml` + `livekit.prod.yaml` — yuqoridagi (2) domenli
variantning bo'lagi, kanonik stackda ishlatilmaydi (kanonik stack LiveKit'ni
o'z ichida ko'taradi).

---

## Qoida

> Deploy bilan bog'liq har qanday o'zgarish **avval `deploy/server/` da**
> bajariladi. Boshqa stacklarga faqat domenli deploymentga o'tish qarori
> qabul qilinganda tegiladi — va o'shanda ular kanonik stackdan qayta
> yig'iladi, teskarisi emas.
