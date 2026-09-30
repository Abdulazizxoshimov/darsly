Bu katalog Caddy tomonidan https://app.169.58.104.245.sslip.io/download/ da
statik tarqatiladi (deploy/server/Caddyfile -> handle /download/*).

⚠️ ESKI APK'LAR: bu yerga oldin joylangan v1.2.0 APK'lar ESKI/O'LIK IP
(194.163.139.242.sslip.io) ni backend sifatida ichiga singdirgan. O'sha IP
qayta taqsimlansa MITM xavfi bor — eski versiyali APK nusxalarini serverdan
OLIB TASHLANG va faqat yangi IP (yoki jonly.uz) bilan qurilgan relizni tarqating.

Fayllar:
  darsly-mentor.apk          — DOIM eng oxirgi reliz (app-config'dagi apk_url
                               shu nomga ishora qiladi; nom o'zgarmaydi)
  darsly-mentor-v<X.Y.Z>.apk — versiyalangan arxiv nusxasi
  darsly-mentor-qr.svg       — yuklab olish havolasining QR kodi

Yangi reliz chiqarish:
  1. mobile/app/build.gradle.kts da versionCode va versionName oshiriladi
  2. cd mobile && ./gradlew assembleRelease
  3. APK ikki nom bilan shu katalogga nusxalanadi (versiyali + darsly-mentor.apk)
  4. rsync bilan serverga: /opt/darsly/deploy/server/dl/
  5. serverdagi .env da APP_ANDROID_LATEST_VERSION yangilanadi va
     `docker compose up -d backend` (restart .env'ni QAYTA O'QIMAYDI)

DIQQAT — imzo kaliti:
  APK mobile/darsly-release.jks bilan imzolanadi (parollar keystore.properties'da,
  ikkalasi ham git'ga tushmaydi). Kalit yo'qolsa keyingi APK'lar o'rnatilgan
  ilova ustiga o'rnatilmaydi. ZAXIRA NUSXASINI SAQLANG.

Katalog mazmuni git'ga TUSHMAYDI (deploy/server/.gitignore).
