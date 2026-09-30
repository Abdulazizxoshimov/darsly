# Mobil QA bo'shliqlari (GAPS)

> Python black-box QA mobil **UI**ni hayday olmaydi — faqat mobil ilova bog'liq
> bo'lgan **backend kontraktini** tekshiradi (`test_mobile_contract.py`,
> `test_mentor_only.py`). Quyidagilar shu qatlam qoplay OLMAYDIGAN, alohida
> asboblar (Espresso/Appium yoki qo'lda) talab qiladigan bo'shliqlar.

## 1. Instrumentation / UI (Espresso/Compose — hozir YO'Q)
Mavjud mobil testlar 535+ JVM unit (parser/repo/pure-logic) — qarang
[[darsly-mobile-arch]]. Instrumented (androidTest) UI/oqim testi **yo'q**:
- Login ekrani → dashboard oqimi (haqiqiy ViewModel + tarmoq).
- Room ekrani: LiveKit SDK connect, kamera/mikrofon, mute/hand/reaction UI.
- `LocalRecorder` (MediaCodec/muxer) — yozuv fayli buzuq/bo'sh bo'lib qolishi
  (audit M-2: eng jimgina yiqiladigan feature, avtomatik guard'i yo'q).
- Recording upload xizmati (`PendingUploadResumer` scan/upload/delete orkestratsiyasi).
- Forced-update gate (`app-config` `min_version`/`force_update` → ilova bloklanishi).

## 2. Client-side rol guard bo'shlig'i
`AuthRepository.kt` FAQAT `role=="admin"` ni rad etadi. **Student** login'i
client-side rad ETILMAYDI. Backend mentor-endpointlarda baribir 403 beradi
(`../backend/test_rbac.py` bilan qoplangan), lekin mobil UI student bilan qanday
ko'rinishi (bo'sh dashboard? xato?) — instrumentation testi kerak.

## 3. WS/realtime mobilda
`RealtimeParser` maydon nomlari backend bilan mos ekani kontrakt testida bilvosita
tekshiriladi (`waiting_room.request` → `request_id`, `admitted` → RoomToken,
`notification` → `body`|`message`). Lekin haqiqiy WS ulanish + reconnect + backoff
mobil qurilmada — JVM unit testda (mavjud) va qo'lda; Python bu WS'ni backend
tomondan tekshiradi (`../backend/test_ws.py`, `../e2e/`).

## 4. Tarmoq chekka holatlari
Token refresh (401 → `me()` → retry), offline kesh, sekin 3G — mobil JVM unit
testlarida (mavjud) qoplangan; qurilmada uchma-uch emas.

---
**Xulosa:** backend kontrakti QA'da qoplangan; mobil **UI/SDK/media** qatlami
Espresso/Appium bosqichini kutadi (P7, README «Bosqichlar»).
