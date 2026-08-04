# Qo'lda sinov to'plami — emulyator + brauzer + server

Bu papka **serverga qarshi** o'lchov o'tkazish uchun. Lokal run ishlatilmaydi:
sinovning butun ma'nosi — foydalanuvchi oladigan APK va foydalanuvchi
ulanadigan server bilan aynan bir xil sharoitni takrorlash.

---

## ⭐ 1-STSENARIY: «Bitta ustozga bu server yetadimi?»

**Bu hali O'TKAZILMAGAN** — asoschi 2026-08-04 da emulyatorni to'xtatgan, sinov
keyinga qoldirilgan. Quyida to'liq tartib va nimani o'lchash kerakligi.

### Nima uchun kerak

Hozirgacha yozuv **bitta ishtirokchi** bilan sinalgan. Haqiqiy dars esa
boshqacha: ustoz ekran ulashadi, **yozuv ketadi** va bir vaqtda **N o'quvchi**
oqimni oladi. Bularning uchalasi bitta 4 yadroli serverda (u yerda
postgres/redis/minio/caddy/prometheus/grafana/loki va ikki Telegram boti ham
bor). Yozuv allaqachon chegarada ishlaydi — 1280×1280 sinovida egress
«pipeline frozen» bilan yiqilib, **yozuv butunlay yo'qolgan**
(`docs/PRODUCT.md` → «Server sig'imi»).

Javob kerak bo'lgan savol: **o'quvchilar qo'shilganda yozuv omon qoladimi?**

### Tayyorgarlik

```bash
# 1) Etalon sahifalar serveri (xostda)
python3 tests/manual/clock/serve.py          # 8099-portda

# 2) Emulyator (apparat GPU SHART — dasturiy GPU bilan 6 fps chiqadi va
#    o'lchov emulyatorning sekinligini o'lchab qo'yadi)
~/Android/Sdk/emulator/emulator -avd darsly34 -gpu host -memory 3072 \
  -no-boot-anim -no-snapshot -camera-back emulated -camera-front emulated

# 3) APK — SERVERDAN (lokal build EMAS). Emulyator x86_64, shuning uchun
#    universal APK kerak: jonly-arm64.apk emulyatorga o'rnatilmaydi.
curl -o /tmp/jonly.apk https://app.169.58.104.245.sslip.io/download/jonly-mentor.apk
adb install -r /tmp/jonly.apk
```

### Yurish tartibi

```bash
# A) Ustoz: dars + ekran ulashish + etalon slayd
PASSWORD='<admin paroli>' TITLE="Yuklama" \
  PAGE="http://10.0.2.2:8099/slide.html" tests/manual/emulator-lesson.sh

# B) Dars ma'lumotini olish
ssh root@169.58.104.245 "docker exec darsly-postgres psql -U darsly -d darsly \
  -tAc \"SELECT id||'|'||join_slug FROM lessons ORDER BY created_at DESC LIMIT 1;\""

# C) Yozuv haqiqatan ketayotganini TASDIQLASH (aks holda sinov ma'nosiz)
ssh root@169.58.104.245 "docker exec darsly-postgres psql -U darsly -d darsly \
  -tAc \"SELECT status FROM recordings ORDER BY created_at DESC LIMIT 1;\""
#   kutilgan: recording

# D) O'quvchilarni qo'shish — MAVJUD darsga (`-slug` bayrog'i shuning uchun bor:
#    o'z darsini yasasa o'quvchilar hech narsaga obuna bo'lmaydi va SFU
#    deyarli ish qilmaydi, ya'ni o'lchov yolg'on chiqadi).
LKS=$(ssh root@169.58.104.245 'grep ^LIVEKIT_API_SECRET /opt/darsly/deploy/server/.env | cut -d= -f2')
cd backend && go run ./tests/load/livekit_load \
  -base https://app.169.58.104.245.sslip.io \
  -slug <join_slug> -lesson <lesson_id> \
  -lk wss://livekit.169.58.104.245.sslip.io -lk-key darslykey -lk-secret "$LKS" \
  -n 25 -batch 8 -batch-delay 2s

# E) SHU PAYTDA serverni o'lchash (alohida terminalda)
ssh root@169.58.104.245 'for i in $(seq 1 20); do
  echo -n "$(date +%H:%M:%S) yuk=$(cut -d" " -f1 /proc/loadavg)  ";
  docker stats --no-stream --format "{{.Name}}={{.CPUPerc}}" \
    darsly-egress darsly-livekit darsly-backend | tr "\n" " "; echo; sleep 15; done'
```

### O'quvchi tomoni + kechikish (ixtiyoriy, alohida)

```bash
cp tests/manual/student-latency.mjs frontend/zz-student.mjs
cd frontend && SLUG=<join_slug> OUT=/tmp/student node zz-student.mjs
rm frontend/zz-student.mjs
```

Skript 8 ta kadr suratini oladi (`frame_<i>_<xostVaqti>.png`) va `getStats`
namunalarini yozadi. Kechikish qo'lda hisoblanadi:

```
kechikish = (fayl nomidagi xost vaqti mod 100000) − (kadrdagi raqam)
```

Ustoz ekranida `http://10.0.2.2:8099/` (soat) turishi kerak, slayd emas.

### Nimani yozib olish kerak

| O'lchov | Qayerdan | Chegara / kutilgan |
|---|---|---|
| Yozuv omon qoldimi | `recordings.status` | **`ready` bo'lishi SHART**; `failed` → server yetmaydi |
| Egress CPU | `docker stats` | 25 fps'da 1027% da yiqilgan; 15 fps'da ~400-600% kutiladi |
| Yuk (load avg) | `/proc/loadavg` | 4 yadro → **8 dan oshsa xavfli** |
| Ulangan o'quvchi | yuklama chiqishi | so'ralgan `-n` ga teng bo'lsin |
| «high cpu» tezligi | `docker logs darsly-egress` | daqiqasiga 50 dan oshsa chegarada |
| Yakuniy kadr | `ffprobe` | 458×1022 (1024 kanvas), 15 fps, 48 kHz stereo |

### Qaror mezoni

- Yozuv `ready` + yuk < 8 → **bitta ustozga yetadi**, N gacha o'quvchi bilan.
- Yozuv `failed` yoki yuk > 10 → **server kuchaytirilishi kerak**
  (`docs/PRODUCT.md` backlog №29).

`-n` ni bosqichma-bosqich oshiring: **10 → 25 → 50**. Har bosqichda yozuv
holatini tekshiring; birinchi `failed` chiqqan son — shu serverning chegarasi.

### ⚠️ Sinovdan keyin TOZALASH

Yuklama vositasi to'xtaganda dars **jonli qolib ketadi** va yozuv CPU yeyaveradi.

```bash
TOK=$(curl -s -X POST https://app.169.58.104.245.sslip.io/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@darsly.uz","password":"<parol>"}' \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['access_token'])")
curl -X POST https://app.169.58.104.245.sslip.io/api/v1/lessons/<id>/end \
  -H "Authorization: Bearer $TOK"
```

Keyin `SELECT count(*) FROM lessons WHERE status='live';` → **0** bo'lishi kerak.

---

## 2-STSENARIY: Yozuv sifati (kadr va matn)

Ustozni `PAGE="http://10.0.2.2:8099/slide.html"` bilan ishga tushiring, 60-90 s
slaydni ko'rsating, darsni yakunlang, transkod tugashini kuting va o'lchang:

```bash
# MinIO'dan olish (object_key ni `recordings` jadvalidan oling)
ffprobe -v error -show_entries stream=width,height,avg_frame_rate,sample_rate,channels \
  -show_entries format=duration -of default=noprint_wrappers=1 rec.mp4
ffmpeg -ss 60 -i rec.mp4 -frames:v 1 kadr.png    # matnni ko'z bilan baholash
```

**Etalon slaydda uch xil matn bor va har biri boshqa savolga javob beradi:**
sarlavha (har doim o'qilishi shart) · asosiy matn (ASOSIY mezon) · mayda
formulalar (sozlama yetarlimi — matematika ustozi uchun aynan shu muhim).

O'lchangan tarix (`docs/PRODUCT.md` → «Yozuv sifati»):

| Sozlama | Kadr | Mayda formulalar |
|---|---|---|
| 1280×720 (eski) | 324×718 | xira |
| **1024 kvadrat (hozirgi)** | **458×1022** | o'qiladi |
| 1280 kvadrat (kuchli server) | 576×1280 | to'liq parite |

⚠️ **Aralash darsda kesish ishlamaydi.** Agar darsda ham ekran ulashish, ham
faqat-kamera davri bo'lsa, kesish qarori barcha namunalarning BIRLASHMASINI
oladi va hech narsa kesilmaydi (aks holda kameraning bir qismi qirqilardi).
Natija kvadrat bo'lib qoladi — rezolyutsiya saqlanadi, kompozitsiya emas.
Sof sinov uchun butun dars davomida ekranni ulashib turing.

---

## Tuzoqlar (ilgari vaqt yo'qotgan joylar)

- **APK ABI.** `jonly-arm64.apk` emulyatorga o'rnatilmaydi (emulyator x86_64).
  Faqat `jonly-mentor.apk` (universal) ichida `x86_64` bor.
- **Klaviatura.** `adb input text` dan keyin klaviatura ochiq qoladi va keyingi
  `tap` tugmaga emas, klaviaturaga tushadi. Har doim `keyevent 4` yoki IME'ning
  ✓ tugmasi bilan yoping.
- **Bitta faol sessiya.** API orqali login qilish emulyatordagi sessiyani
  BEKOR QILADI. Sinov davomida API'ga kirmang yoki oxirida qayta kiring.
- **Emulyator GPU.** `-gpu swiftshader_indirect` bilan 6 fps, `-gpu host`
  bilan 11 fps. Kechikish o'lchovi buni to'g'ridan-to'g'ri sezadi.
- **Dars yakunlash dialogi.** Tugma joyi ekran ulashish yoniq/o'chiq holatda
  SILJIYDI. Bosishdan oldin `screencap` bilan tekshiring.
- **Jonli dars ro'yxatdan yakunlanmaydi** — faqat xona ichidan (backlog №30).
