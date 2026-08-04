#!/bin/bash
# Emulyatorda USTOZ oqimi: kirish → dars yaratish → xona → ekran ulashish → sahifa.
#
# Nega skript: qadamlar ko'p va koordinatalar aniq bo'lishi kerak; qo'lda
# takrorlanganda har safar boshqa natija chiqadi va o'lchov taqqoslanmaydi.
#
# ⚠️ Koordinatalar `darsly34` AVD (1080x2400) uchun. Boshqa ekranda ular
# ishlamaydi — `adb shell wm size` bilan tekshiring.
#
# Ishlatish:
#   PASSWORD=<parol> TITLE="Sinov" PAGE="http://10.0.2.2:8099/slide.html" \
#     tests/manual/emulator-lesson.sh
#
# Kirish shart bo'lmasa (allaqachon kirilgan) SKIP_LOGIN=1 bering.
set -u
export PATH="$PATH:$HOME/Android/Sdk/platform-tools"

EMAIL="${EMAIL:-admin@darsly.uz}"
PASSWORD="${PASSWORD:-}"
TITLE="${TITLE:-Sinov}"
PAGE="${PAGE:-http://10.0.2.2:8099/slide.html}"
SKIP_LOGIN="${SKIP_LOGIN:-0}"

say() { echo "  · $*"; }
tap() { adb shell input tap "$1" "$2"; }

adb wait-for-device
[ "$(adb shell getprop sys.boot_completed | tr -d '\r')" = "1" ] || { echo "emulyator hali yuklanmadi"; exit 1; }

say "ilovani qayta ishga tushiraman (holat toza bo'lsin)"
adb shell am force-stop uz.darsly.mentor; sleep 2
adb shell am start -n uz.darsly.mentor/.MainActivity >/dev/null 2>&1; sleep 10

if [ "$SKIP_LOGIN" != "1" ]; then
  [ -n "$PASSWORD" ] || { echo "PASSWORD kerak (yoki SKIP_LOGIN=1)"; exit 1; }
  say "kirish"
  tap 540 1166; adb shell input text "$EMAIL"
  tap 540 1367; adb shell input text "$PASSWORD"
  # ⚠️ Klaviatura OCHIQ turganda "Kirish" tugmasi bosilmaydi — tugma
  # klaviatura ostida qoladi va tap boshqa joyga tushadi. Avval IME'ning
  # ✓ tugmasi bilan yopamiz.
  tap 994 2172; sleep 2
  tap 540 1561; sleep 14
fi

say "dars yaratish: $TITLE"
tap 126 2247; sleep 4          # «Darslar» tabi
tap 833 2012; sleep 5          # «Dars yaratish»
tap 540 432; adb shell input text "$TITLE"
adb shell input keyevent 4; sleep 2   # klaviaturani yopish
tap 956 212; sleep 20          # «Boshlash»

# Ruxsat dialoglari faqat birinchi o'rnatishdan keyin chiqadi; bosilsa zarari yo'q.
tap 540 1227; sleep 3          # kamera
tap 540 1227; sleep 3          # mikrofon
tap 540 1306; sleep 12         # bildirishnoma

say "ekran ulashish"
tap 476 2231; sleep 4          # «Ekran»
tap 678 1565; sleep 3          # «Ramkasiz davom etish» (efir-ramka dialogi chiqsa)
tap 667 1708; sleep 4          # «Tushunarli, davom etish»
tap 853 1583; sleep 8          # tizim dialogi: «Start now»

say "sahifa ochilmoqda: $PAGE"
adb shell am start -a android.intent.action.VIEW -d "$PAGE" >/dev/null 2>&1
sleep 8
adb shell input swipe 540 1600 540 1400 400   # bir oz skroll — harakat bo'lsin

echo "  · USTOZ TAYYOR. Dars ma'lumotini olish:"
echo "      SELECT id, join_slug FROM lessons ORDER BY created_at DESC LIMIT 1;"
