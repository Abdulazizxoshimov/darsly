# Telegram arxivini ulash — qadamma-qadam

Bu qo'llanma **bir marta** bajariladi. Tugagach dars yozuvlari avtomatik
Telegram guruhingizga saqlanadi va serverdan 30 kundan keyin o'chsa ham
Telegramda abadiy qoladi.

> **Hozir ulanmagan bo'lsa nima bo'ladi?** Hech narsa buzilmaydi. Yozib olish
> avvalgidek ishlaydi, yozuvlar serverda turadi. Telegram — qo'shimcha
> ishonchlilik qatlami; uni istalgan vaqtda yoqasiz.

Jami vaqt: ~15 daqiqa.

---

## 1-qadam. Botni yaratish (5 daqiqa)

1. Telegramda **@BotFather** ni oching.
2. `/newbot` yuboring.
3. Bot **nomini** so'raydi (odamlar ko'radigan nom): masalan `Jonly Arxiv`.
4. Bot **username** ini so'raydi — `bot` bilan tugashi shart:
   masalan `jonly_arxiv_bot`.
5. BotFather javobida shunday satr bo'ladi:

   ```
   Use this token to access the HTTP API:
   7123456789:AAF-abcdefghIJKLMNOPqrstuvwxyz123456
   ```

   Bu **`TELEGRAM_BOT_TOKEN`**. Uni nusxa oling va hech kimga bermang —
   token botni to'liq boshqarish huquqi.

6. Shu yerda yana ikki buyruq yuboring (ixtiyoriy, lekin tavsiya etiladi):
   - `/setprivacy` → botni tanlang → **Disable**
     (bo'lmasa bot guruhdagi buyruqlarni ko'rmaydi);
   - `/setjoingroups` → botni tanlang → **Enable**
     (bo'lmasa botni guruhga qo'sha olmaysiz).

---

## 2-qadam. api_id va api_hash olish (5 daqiqa)

Bu **50 MB chegarasini 2 GB ga ko'tarish** uchun kerak. Ularsiz 1 soatlik
dars yozuvi Telegramga umuman yuborilmaydi.

1. Brauzerda **my.telegram.org** ni oching.
2. Telefon raqamingizni xalqaro formatda kiriting: `+998901234567`.
3. Telegram ilovasiga **kod** keladi — uni saytga kiriting.
4. **API development tools** bo'limini bosing.
5. Forma to'ldiring (istalgan qiymat bo'lishi mumkin):
   - *App title*: `Jonly`
   - *Short name*: `jonly`
   - *Platform*: `Other`
6. **Create application** bosing.
7. Chiqqan sahifada ikki qiymat bo'ladi:
   - **App api_id** — masalan `1234567` → bu **`TELEGRAM_API_ID`**
   - **App api_hash** — masalan `0123456789abcdef...` → bu **`TELEGRAM_API_HASH`**

> Bu qiymatlar **shaxsiy akkauntingizga** bog'langan va boshqa
> ko'rsatilmaydi — darhol saqlab qo'ying.

---

## 3-qadam. Arxiv guruhini yasash (3 daqiqa)

Bu **sizning shaxsiy** guruhingiz — barcha darslar avtomatik shu yerga
tushadi (ikkinchi nusxa, ishonchlilik uchun). O'quvchilar bu guruhda
bo'lmaydi.

1. Telegramda yangi **guruh** yarating: nomi masalan `Video darslar`.
2. A'zo sifatida faqat o'zingizni qoldiring (Telegram kamida bitta odam
   qo'shishni so'raydi — botni qo'shsangiz bo'ladi).
3. Guruhga **botingizni qo'shing** va uni **administrator** qiling
   (Guruh nomi → *Administrators* → *Add Administrator* → botni tanlang).
   Kamida **Post Messages** huquqi bo'lsin.
4. Endi **chat ID** ni bilish kerak. Eng oson yo'l:
   - guruhga **@getidsbot** yoki **@RawDataBot** ni vaqtincha qo'shing;
   - u guruh ID sini yozadi: masalan `-1001234567890`;
   - ID ni nusxa olib, o'sha yordamchi botni guruhdan **chiqaring**.

   Bu **`TELEGRAM_ARCHIVE_CHAT_ID`**. **Minus belgisi bilan** yozing.

---

## 4-qadam. Serverga yozish (2 daqiqa)

Serverga kiring va `.env` faylini oching:

```bash
ssh root@194.163.139.242
cd /root/darsly/deploy/server
nano .env
```

Quyidagi qatorlarni **to'ldiring** (ular allaqachon bo'sh holda turadi):

```
TELEGRAM_BOT_TOKEN=7123456789:AAF-abcdefghIJKLMNOPqrstuvwxyz123456
TELEGRAM_API_ID=1234567
TELEGRAM_API_HASH=0123456789abcdef0123456789abcdef
TELEGRAM_ARCHIVE_CHAT_ID=-1001234567890
```

Qolgan uchtasi allaqachon to'g'ri turadi, tegmang:

```
TELEGRAM_API_URL=http://telegram-bot-api:8081
TELEGRAM_UPLOAD_MAX_MB=1900
TELEGRAM_FILE_ROOT=/var/lib/telegram-bot-api
```

Saqlang (`Ctrl+O`, `Enter`, `Ctrl+X`) va stack'ni qayta ko'taring:

```bash
docker compose --profile telegram up -d --build
```

> **`--profile telegram` ni unutmang.** Usiz Local Bot API Server ishga
> tushmaydi va chegara 50 MB bo'lib qoladi (darslar yuborilmaydi).

Tekshirish:

```bash
docker compose logs backend | grep -i telegram
```

Shunday satr chiqishi kerak:

```
telegram configured  bot=jonly_arxiv_bot  local_api=true  archive_chat=-1001234567890
```

---

## 5-qadam. Hisobingizni botga bog'lash (2 daqiqa)

Bu bot sizni **taniy olishi** uchun: dars tugagach u aynan sizga
«qaysi guruhga yuboray?» deb yozadi.

1. Ilovaga (`app.194.163.139.242.sslip.io`) mentor sifatida kiring.
2. **Sozlamalar → Telegram bilan bog'lash** ni bosing.
3. Ekranda kod chiqadi: masalan `ABCD2345` (15 daqiqa amal qiladi).
4. Telegramda **botingizni** oching va yuboring:

   ```
   /start ABCD2345
   ```

5. Bot javob beradi: *«Tayyor, Ali! Hisobingiz bog'landi.»*

---

## 6-qadam. O'quvchilar guruhini ulash (1 daqiqa)

1. O'quvchilaringiz bo'lgan guruhni oching.
2. Botni **administrator** qilib qo'shing (Post Messages huquqi bilan).
3. Bot sizga shaxsiy chatda tasdiq yozadi:
   *«9-A sinf guruhi qo'shildi.»*

Shuni har bir sinf/guruh uchun takrorlang.

---

## Endi qanday ishlaydi

1. Dars tugaydi → yozuv tayyorlanadi.
2. Bot uni **avtomatik** arxiv guruhingizga yuboradi.
3. Bot sizga shaxsiy chatda yozadi:

   > «Algebra» darsi tugadi va yozuv arxivga saqlandi.
   > O'quvchilar guruhiga ham yuboraymi?
   > [9-A sinf] [9-B sinf] [Yubormayman]

4. Guruhni tanlasangiz — video o'sha guruhga bir zumda ketadi
   (qayta yuklanmaydi, shuning uchun tez).
5. Keyin so'raydi: *«Chat tarixi ham yuborilsinmi?»* → Ha/Yo'q.
   Shaxsiy xabarlar hech qachon yuborilmaydi — faqat ochiq chat.
6. 30 kundan keyin server nusxasi o'chadi, Telegramdagi **qoladi**.
   Ilovada eski darsni ochsangiz — bot uni qaytarib oladi (30-60 soniya).

---

## Muammolar

**Bot javob bermayapti**
`docker compose logs backend | grep -i telegram` — token noto'g'ri bo'lsa
`telegram: bot tokeni tekshirilmadi` yozuvi chiqadi.

**«Guruh qo'shildi, lekin akkauntingiz bog'lanmagan»**
5-qadamni bajaring (`/start <kod>`), keyin botni guruhdan chiqarib qayta
qo'shing — shunda guruh sizga biriktiriladi.

**Video yuborilmadi, ilovada «Yozuv Telegramga yuborilmadi» xabari**
Server nusxasi **o'chirilmagan** — video xavfsiz. Sabablari odatda:
bot guruhdan chiqarilgan, unga Post Messages huquqi berilmagan, yoki
`--profile telegram` unutilgan (u holda 50 MB chegara ishlaydi).
Tuzatgandan keyin yozuv keyingi darsda emas, **avtomatik** qayta urinmaydi —
ilovadan yozuvni qo'lda yuklab olib, guruhga tashlang yoki
`docker compose restart backend` qiling.

**Guruh ro'yxati bo'sh**
Bot guruhda **administrator** emas yoki `/setjoingroups` o'chiq.
1-qadamning 6-bandiga qarang.

---

## Nima qayerda saqlanadi

| Nima | Qayerda | Qancha |
|------|---------|--------|
| Video (asl) | Server (MinIO) | 30 kun |
| Video (nusxa) | Telegram arxiv guruhi | abadiy |
| Video (o'quvchilarga) | Siz tanlagan guruh | abadiy |
| Chat tarixi | Server DB | abadiy |
| Chat fayli (TXT) | Siz tanlagan guruh | abadiy (agar yuborsangiz) |
| Darsdagi fayllar | Faqat serverda | — |
