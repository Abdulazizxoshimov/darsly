# Darsly API Kontrakt (haqiqiy backend'dan)

> `integration-agent` va `frontend-agent` uchun yagona haqiqat manbai. Backend `api/router.go` + handler'lardan olingan.
> Base: `/api/v1`. Konvert: muvaffaqiyat `{ "data": T }`; ro'yxat `{ data, total, page, limit, total_pages }`;
> xato `{ "code": string, "message": string }`. Auth: `Authorization: Bearer <access>`.

> Xato konverti BARCHA qatlamlarda bir xil — handler (`hs.Error`) ham, middleware ham
> (401/403/429/503) `{code, message}` qaytaradi. Eskirgan `error` maydoni endi yo'q.

## App (ochiq, auth'siz, rate-limit 20 rps/40 burst per IP)
| Method | Path | Req | Resp |
|---|---|---|---|
| GET | `/app-config` | — | `{android:{min_version,latest_version,apk_url,force_update,release_notes}}` |

Mobil klient ishga tushganda so'raydi: o'z versiyasi `min_version`dan past bo'lsa yoki
`force_update=true` bo'lsa ishlashdan bosh tortadi va `apk_url`ni ko'rsatadi (side-load
APK'da avtomatik yangilanish yo'q). Qiymatlar server env'idan — DB'ga tegmaydi.

## Auth (ochiq, rate-limit 60/min/IP)
| Method | Path | Req | Resp |
|---|---|---|---|
| POST | `/auth/register` | `{full_name,email,password}` | 201 `{access_token,refresh_token}` |
| POST | `/auth/login` | `{email,password}` | 200 `{access_token,refresh_token}` |
| POST | `/auth/refresh` | `{refresh_token}` | 200 `{access_token,refresh_token}` |
| POST | `/auth/forgot-password` | `{email}` | 204 |
| POST | `/auth/reset-password` | `{token,new_password}` | 204 |

`/auth/refresh` idempotent: rotatsiya javobi yo'qolsa (mobil tarmoq) yoki ikki parallel
401-retry ketsa, AYNI o'sha refresh bilan `JWT_REFRESH_GRACE` (default 60s) ichida qayta
so'rov yuborish xuddi o'sha juftlikni qaytaradi va sessiyani o'ldirmaydi. Oyna
tashqarisidagi reuse haliyam 401 + butun sessiya-oilasini bekor qilish.
Sessiya bekor/muddati tugagan bo'lsa refresh 401 beradi (klient toza logout qiladi).
Bir vaqtda yuborilgan N ta refresh so'rovi ham xavfsiz: rotatsiyani atomik CAS bajaradi,
sessiyada har doim BITTA tirik refresh zanjiri qoladi va barcha javoblar bir xil bo'ladi.

### Bitta akkaunt = bitta faol sessiya (PRODUCT.md №1)
Muvaffaqiyatli `POST /auth/login` shu foydalanuvchining BOSHQA barcha sessiyalarini
tugatadi (akkaunt ulashishga qarshi mahsulot qoidasi). Eski qurilma keyingi so'rovda:

| Holat | Status | `code` | Klient nima qilishi kerak |
|---|---|---|---|
| access token bilan himoyalangan so'rov | 401 | `SESSION_REVOKED` | logout + «Boshqa qurilmada kirildi» |
| `POST /auth/refresh` | 401 | `SESSION_REVOKED` | logout + «Boshqa qurilmada kirildi» |
| token buzuq / imzo yaroqsiz | 401 | `TOKEN_INVALID` | logout (sababsiz) |
| access muddati tugagan | 401 | `TOKEN_EXPIRED` | refresh urinish |

`SESSION_REVOKED` faqat "boshqa qurilma" degani EMAS: logout, parol tiklash va
sessiya muddatining tugashi ham shu kodni beradi — uchalasida ham to'g'ri harakat
bir xil (qayta kirish). `JWT_REFRESH_GRACE` oynasiga ta'sir qilmaydi: grace faqat
TIRIK sessiya ichida ishlaydi, tugatilgan sessiyani tiriltirmaydi.

## Auth (himoyalangan)
| POST | `/auth/logout` | `{refresh_token}` | 204 |
| GET | `/auth/me` | — | `User` |

## Joinlink (ochiq, rate-limit 600/min/IP)
| GET | `/joinlink/:slug` | — | `LessonPublic` |
| POST | `/joinlink/:slug` | `{guest_name?,passcode?}` | `JoinLessonResp` |

> Yakunlangan/bekor qilingan dars (№3): GET (preview) 200 qaytaradi (status ichida —
> klient "dars tugagan" sahifasini ko'rsatadi). POST (join) ham 200, lekin
> `next_step:"lesson_ended"` — `room` ham, `request_id` ham YO'Q (token berilmaydi,
> parol/qulf tekshirilmaydi). Aniq holat `lesson.status`da (`ended`|`cancelled`).
> Mentor qora ro'yxatidagi ism (№4) bilan join → 403.

## Waiting room
| GET | `/waitingroom/:id/status` | — (ochiq) | `WaitingRoomStatusResp` |
| GET | `/lessons/:id/waitingroom` | auth(mentor) | `[WaitingRoomRequest]` |
| POST | `/waitingroom/:id/admit` | auth(mentor) | `RoomToken` |
| POST | `/waitingroom/:id/reject` | auth(mentor) | 204 |
| POST | `/lessons/:id/waitingroom/admit-all` | auth(mentor) | `AdmitAllResp` |

> **«Hammasini kiritish»** (`admit-all`) — tanasi YO'Q. Butun navbat bitta so'rovda
> hal bo'ladi: avval N ta alohida `admit` yuborilardi va sekin internetda ustoz
> ro'yxatning yarmi kirib yarmi kirmagan holatni ko'rardi.
>
> **Qisman muvaffaqiyat NORMAL:** har so'rov mustaqil, atomik olinadi. Javob
> **har doim 200** (navbat bo'sh bo'lsa ham — `{total:0,admitted:0,failed:0}`,
> tugma bloklanmasin), sonlar esa rostini aytadi. Ro'yxat o'qilgandan keyin
> alohida admit/reject qilingan so'rov o'tkazib yuboriladi (dublikat token yo'q).
> Begona mentor → **403**, yaroqsiz/mavjud emas dars ID → **404**.
>
> Har bir kiritilgan guest tokenni odatdagi yo'l bilan oladi: WS
> `waiting_room.admitted` yoki `GET /waitingroom/:id/status`.

## Lessons (himoyalangan)
| POST | `/lessons` | `CreateLessonReq` | 201 `Lesson` |
| GET | `/lessons?status=&page=&limit=&search=` | — | `[Lesson]` (list konvert) |
| GET | `/lessons/:id` | — | `Lesson` |
| PATCH | `/lessons/:id` | `UpdateLessonReq` | `Lesson` |
| DELETE | `/lessons/:id` | — | 204 |

## Room / LiveKit (himoyalangan, mentor)
| POST | `/lessons/:id/token` | — | `RoomToken` (host) |
| POST | `/lessons/:id/end` | — | 204 |
| GET | `/lessons/:id/participants` | — | `[RoomParticipant]` |
| POST | `/lessons/:id/mute-all` | `{allow_self_unmute?}` (ixtiyoriy body) | 204 |
| POST | `/lessons/:id/participants/:identity/mute` | — | 204 |
| POST | `/lessons/:id/participants/:identity/remove` | `{scope?:"lesson"\|"mentor"}` (ixtiyoriy body; default `lesson`) | 204 |
| POST | `/lessons/:id/participants/:identity/allow-speak` | — | 204 |
| POST | `/lessons/:id/participants/:identity/revoke-speak` | — | 204 |
| GET | `/blocklist` | — | `[BlocklistEntry]` (mentorning doimiy qora ro'yxati) |
| DELETE | `/blocklist/:id` | — | 204 (unban) |

> **Avto-yakun (PRODUCT.md №2) — YANGI API yo'q, lekin klient xulqi o'zgaradi.**
> Dars endi SERVER tomonidan ham yakunlanishi mumkin, ya'ni `POST /lessons/:id/end`
> chaqirilmagan bo'lsa ham `status` `ended` ga o'tadi. Ikki qoida:
> - **4 soatlik texnik limit** (`LESSON_MAX_DURATION`) — `started_at` dan hisoblanadi;
> - **bo'sh xona** (`LESSON_EMPTY_GRACE`, default 20m) — xonada hech kim qolmagach
>   grace kutiladi va hali ham bo'sh bo'lsa dars yakunlanadi. Mentor 4G uzilishida
>   qaytib ulansa hisob BEKOR bo'ladi.
>
> Ta'siri klientlarga: yakunlangan darsda `POST /lessons/:id/token` **400
> `BAD_REQUEST`** (`"lesson is not active"`), joinlink esa `next_step:"lesson_ended"`.
> Yozuv ham avto-yakunda to'g'ri yopiladi (`processing` → `ready`).
> Reopen YO'Q — yakunlangan dars qayta ochilmaydi.

> **Zoom ovoz nazorati (№11).** O'quvchi tokeni endi `canPublish=true`
> (kamera+mikrofon; ekran ulashish FAQAT ustozda — tokenga imzolangan).
> Dars sozlamalari server webhook'da qo'llanadi: `mute_on_entry=true` → yangi
> ishtirokchining birinchi audio publish'i mute bilan boshlanadi;
> `allow_self_unmute=false` → har audio publish server tomonidan qayta mute
> qilinadi (o'quvchi o'zini ocholmaydi). `mute-all` body'sidagi
> `allow_self_unmute` bayroqni bir yo'la yangilaydi (Zoom checkbox'i);
> bayroqni `PATCH /lessons/:id` bilan ham jonli o'zgartirsa bo'ladi.
>
> **Siyosatni endi SERVER e'lon qiladi (klient emas).** Avval `kind:"policy"`
> data-message'ini faqat web ustoz klienti yuborardi — mobil ustoz hech narsa
> yubormasdi va telefondan o'tilgan darsda o'quvchining mikrofon tugmasi yolg'on
> ko'rsatardi (yoqadi, server jimgina qayta mute qiladi). Uch manba:
> - `POST /lessons/:id/mute-all` dan keyin server xonaga tarqatadi:
>   `{"kind":"policy","mute_on_entry":<bool>,"allow_self_unmute":<bool>}`;
> - `LessonPublic` (joinlink preview/join javobi) — kirish paytidagi qiymat;
> - `GET /rooms/:lessonID/state` — kech ulangan/qayta ulangan klient uchun.
>
> Bu maydonlar faqat **UI ko'rsatkichi**; haqiqiy chegara baribir server
> webhook'ida qo'llanadi.
>
> **Ban tanlovi (№4).** `remove` da `scope:"mentor"` — ishtirokchi mentorning
> HAMMA darslaridan doimiy bloklanadi (`mentor_blocklist`; moslik ko'rsatilgan
> ism bo'yicha, katta-kichik harf farqsiz — o'quvchida akkaunt yo'qligi uchun
> ongli cheklov). `scope` bo'sh/`"lesson"` — eski xulq (faqat shu dars, Redis).

## Chat (himoyalangan, mentor)
| GET | `/lessons/:id/chat` | — | `[ChatMessage]` |
| POST | `/lessons/:id/chat` | `{body,to?}` | 201 `ChatMessage` |
| POST | `/lessons/:id/chat/upload` | `multipart: file,body?,to?` | 201 `ChatMessage` |
| DELETE | `/lessons/:id/chat/:messageID` | — | 204 |
| GET | `/lessons/:id/chat/transcript?format=txt\|html` | — | fayl (`Content-Disposition: attachment`) |

Backend host chat'ni data-channel'ga ham broadcast qiladi (`ChatMessage` shakli).
Ishtirokchi yo'li — pastdagi «Chat — ishtirokchi yo'li» bo'limida.
Transkript — «Chat transkripti (№21)» bo'limida.

## Polls
| GET | `/lessons/:id/polls` | auth(mentor) | `[Poll]` |
| POST | `/lessons/:id/polls` | `{question,options[2..10],results_visibility?}` | 201 `Poll` |
| POST | `/lessons/:id/polls/:pollID/publish` | auth(mentor) | `PollResults` · 400 (mentor_only) |
| POST | `/polls/:id/close` | auth(mentor) | `PollResults` |
| POST | `/polls/:id/vote` | ochiq `{token,option_index}` | 204 |
| GET | `/polls/:id/results` | ochiq `?token=` | `PollResults` · 403 (e'lon qilinmagan) |

## Recording (himoyalangan, mentor)
| POST | `/lessons/:id/recording/start` | — | 201 `Recording` |
| GET | `/lessons/:id/recordings` | — | `[Recording]` |
| POST | `/recordings/:id/stop` | — | 204 |
| GET | `/recordings/:id/download` | — | `RecordingDownload` (presigned, 1h) |
| GET | `/recordings/:id` | — | `Recording` (tiklash holatini poll qilish) |
| POST | `/recordings/:id/restore` | — | **202** `RecordingRestore` |

### Retention — 30 kun (PRODUCT.md №5)
`ready` yozuv `expires_at` maydoni bilan qaytadi (hisoblanadigan:
`ended_at + RECORDING_SERVER_RETENTION_DAYS`) — klient «X kundan keyin
o'chadi» deb ko'rsatadi.

- O'chishga 3 kun qolganda mentor bildirishnoma oladi:
  `type:"recording_expiring"`, `lesson_id` to'ldirilgan. Bir yozuv uchun BIR marta.
- `RECORDING_SERVER_RETENTION_DAYS=0` bo'lsa o'chirish o'chirilgan va
  `expires_at` qaytmaydi.

**Telegram arxivi O'CHIQ bo'lsa** (default): muddat o'tgach fayl MinIO'dan
o'chadi va status `expired` bo'ladi (qator tarix uchun qoladi).
`GET /recordings/:id/download` bunda **400 `BAD_REQUEST`**
(`"recording has expired and was deleted"`).

### Telegram arxivi — `archived` va tiklash (PRODUCT.md 2026-08-01)
`TELEGRAM_BOT_TOKEN` sozlangan bo'lsa retention MA'NOSI o'zgaradi:

- Fayl **faqat** Telegramda tasdiqlangan bo'lsa o'chiriladi (`telegram_sent_at`
  to'lgan) va status **`archived`** bo'ladi — yozuv YO'QOLMAGAN.
- Tasdiqlanmagan yozuv **o'chirilmaydi** (serverda qoladi), `expires_at`
  qaytmaydi va mentor `type:"telegram_upload_failed"` bildirishnoma oladi.
- `archived` bo'lganda mentor `type:"recording_archived"` bildirishnoma oladi.

**Tiklash oqimi** (klient shu ketma-ketlikni bajaradi):

1. `GET /recordings/:id/download` → status `archived` bo'lsa **400**
   (`"recording is archived — restore it first"`). Avtomatik boshlanmaydi:
   tiklash yuzlab megabayt trafik va mentor buni bilib turib boshlasin.
2. `POST /recordings/:id/restore` → **202** `{"status":"restoring","poll_after_s":5}`.
   Idempotent: ikki marta bosilsa ham bitta yuklab olish ketadi.
   Yozuv allaqachon serverda bo'lsa `{"status":"ready"}` (xato emas).
3. Klient `GET /recordings/:id` ni `poll_after_s` oralig'ida so'raydi.
   `status` `restoring` → `ready` bo'lganda `download` ishlaydi.
   Yiqilsa `archived` ga qaytadi va `telegram_error` to'ladi.
4. Tiklangan nusxa `cached_until` gacha turadi (`RECORDING_CACHE_TTL_HOURS`,
   default 24s), keyin yana `archived` bo'ladi.

`restoring` holatida `download` → **400** (`"recording is being restored"`).

## Telegram (himoyalangan, mentor)
| GET | `/me/telegram` | — | `TelegramLinkStatus` |
| POST | `/me/telegram/link` | — | 201 `TelegramLink` |
| DELETE | `/me/telegram` | — | 204 |

- `GET /me/telegram` → `enabled:false` bo'lsa serverda integratsiya
  sozlanmagan: klient «Telegram bilan bog'lash» bo'limini **ko'rsatmasin**
  (DB'ga ham borilmaydi).
- `POST /me/telegram/link` bir martalik kod beradi (15 daqiqa). Mentor botga
  `/start <kod>` yuboradi; `deep_link` bosilsa Telegram buni o'zi qiladi.
  Integratsiya o'chiq bo'lsa **400 `BAD_REQUEST`**.
- Kod BIR MARTALIK: ikkinchi urinish **400**.

## Notifications (himoyalangan)
| GET | `/notifications?unread=&page=&limit=` | — | `[Notification]` (list) |
| GET | `/notifications/unread-count` | — | `{count}` |
| POST | `/notifications/:id/read` | — | 204 |
| POST | `/notifications/read-all` | — | 204 |

## Users (himoyalangan)
| GET | `/users` / `/users/:id` | — | `User` / `[UserShort]` |
| GET | `/users/me` | — | `User` |
| PUT | `/users/me` | `UpdateUserReq` (Role bloklanadi) | `User` |
| PUT | `/users/me/password` | `{current_password,new_password}` | 204 |

## WebSocket
- `GET /api/v1/ws?token=<jwt>` — authed (mentor). Xabar: `{type,room?,payload,created_at}`.
- `GET /api/v1/ws/waitingroom?request_id=<uuid>` — guest (ochiq).
- Turlar: `notification`, `waiting_room.request`, `waiting_room.admitted` (payload=RoomToken), `waiting_room.rejected`.

## Model shakllari (JSON tag)
- **User**: `{id,email,full_name,avatar_url?,color,role,timezone,language,is_active,last_login_at?,created_at,updated_at}`
- **Lesson**: `{id,mentor_id,title,description?,scheduled_at?,duration_min,recurrence_rule?,join_slug,has_passcode,is_locked,is_recording_enabled,is_waiting_room_enabled,mute_on_entry,allow_self_unmute,status,started_at?,ended_at?,created_at,updated_at}`; status: `scheduled|live|ended|cancelled`
- **CreateLessonReq**: `{title,description?,scheduled_at?,duration_min?,recurrence_rule?,passcode?,is_recording_enabled,is_waiting_room_enabled,mute_on_entry?,allow_self_unmute?}` (mute_on_entry/allow_self_unmute berilmasa server default'i **true**;
  `is_waiting_room_enabled` berilmasa **false** — PRODUCT.md «Kutish xonasi: default o'chiq».
  Bu qiymat avval uch joyda zid edi: DB ustuni `TRUE`, web `true`, mobil `false` —
  endi qaror faqat serverda)
- **UpdateLessonReq**: `{title?,description?,scheduled_at?,duration_min?,passcode?,remove_passcode?,is_locked?,is_recording_enabled?,is_waiting_room_enabled?,mute_on_entry?,allow_self_unmute?,status?}`
- **RoomToken**: `{token,ws_url,room_name,identity,role}`; role: `host|participant`
- **RoomParticipant**: `{identity,name,joined_at_ms,active,audio_muted,video_muted}`
- **LessonPublic**: `{id,title,mentor_name,scheduled_at?,status,has_passcode,is_waiting_room_enabled,mute_on_entry,allow_self_unmute}`
- **JoinLessonResp**: `{lesson:LessonPublic,next_step:"waiting_room"|"join"|"lesson_ended",room?:RoomToken,request_id?}`
- **BlocklistEntry**: `{id,identity,display_name,created_at}`
- **WaitingRoomRequest**: `{id,lesson_id,requester_name,status,created_at,decided_at?}`
- **WaitingRoomStatusResp**: `{request_id,status:"pending"|"admitted"|"rejected",room?:RoomToken}`
- **AdmitAllResp**: `{total,admitted,failed}` — `failed = total - admitted` (qisman muvaffaqiyat)
- **ChatMessage**: `{id,lesson_id,sender_identity,sender_name,body,created_at,to_identity?,file?}`
- **ChatFile** (`ChatMessage.file`): `{name,size,mime,url,expires_in_s}` — `url` presigned (1 soat), HAR javobda qayta imzolanadi (bazada saqlanmaydi)
- **Poll**: `{id,lesson_id,question,options[],is_active,created_at,closed_at?,results_visibility,results_published_at?}`; `results_visibility`: `mentor_only|public`
- **PollResults**: `{poll:Poll,counts[],total}`
- **Recording**: `{id,lesson_id,egress_id,status:"recording"|"processing"|"ready"|"failed"|"expired"|"archived"|"restoring",duration_sec,size_bytes,started_at,ended_at?,created_at,expires_at?,telegram_sent_at?,telegram_message_id?,telegram_error?,telegram_attempts?,cached_until?}` — `expires_at` faqat `ready` va Telegramda tasdiqlangan yozuvda; `telegram_file_id`/`telegram_chat_id` TASHQARIGA CHIQMAYDI
- **RecordingDownload**: `{url,expires_in_s,duration_sec,size_bytes}`
- **RecordingRestore**: `{status:"restoring"|"ready",poll_after_s}`
- **TelegramLink**: `{code,deep_link?,expires_in_s}` — `deep_link` bot username ma'lum bo'lsa
- **TelegramLinkStatus**: `{enabled,linked,telegram_username?,linked_at?,chats?:[TelegramChat]}`
- **TelegramChat**: `{chat_id,title,type,is_active,added_at,updated_at}`
- **Notification**: `{id,user_id,type:"lesson_reminder"|"waiting_room"|"system"|"recording_expiring"|"telegram_upload_failed"|"recording_archived",title,body,lesson_id?,read_at?,created_at}`
- **LessonArchive**: `{lesson:Lesson,recording:ArchiveRecording|null,chat:[ArchiveChatMessage],materials:[ArchiveMaterial]}`
- **ArchiveRecording**: `{id,status,duration_sec,size_bytes,url:string|null,expires_at:string|null}` — `url` faqat `status="ready"` da
- **ArchiveChatMessage**: `{id,sender_identity,sender_name,body,to_identity,file,created_at,offset_sec}` — `to_identity`/`file` yo'q bo'lsa `null` (omitempty EMAS)
- **ArchiveMaterial**: `{name,size,mime,url,created_at}`

## Xona holati (roomstate) — qo'l ko'tarish va reaksiyalar

Ochiq endpointlar LiveKit **room-token** bilan autentifikatsiya qilinadi (guest'da JWT yo'q).
Token'ning xonasi dars bilan mos kelishi shart. Token yo'q/yaroqsiz → **401**.

| Method | Yo'l | Auth | Tana / Query | Javob |
|---|---|---|---|---|
| POST | `/api/v1/rooms/:lessonID/hand` | room-token | `{token, raised}` | 204 |
| POST | `/api/v1/rooms/:lessonID/reaction` | room-token | `{token, emoji}` | 204 · 400 (ruxsatsiz emoji) · 429 (10 s da 5) |
| GET | `/api/v1/rooms/:lessonID/state` | room-token | `?token=` | `{data:RoomState}` |
| POST | `/api/v1/lessons/:id/hands/lower` | JWT (mentor) | `{identity}` | 204 |
| POST | `/api/v1/lessons/:id/hands/lower-all` | JWT (mentor) | — | 204 |

**RoomState**: `{hands:[{identity,name,raised_at}],recording,mute_on_entry,allow_self_unmute}`

`hands` — **ko'tarilgan vaqt bo'yicha tartiblangan** (navbat serverda hisoblanadi).
`mute_on_entry`/`allow_self_unmute` — darsning joriy ovoz siyosati: kech ulangan
yoki qayta ulangan klient `mute-all` data-message'ini o'tkazib yuborgan bo'lsa
uni FAQAT shu yerdan tiklaydi. Dars o'qib bo'lmasa ruxsat beruvchi qiymat
qaytadi (`allow_self_unmute:true`) — mavjud bo'lmagan taqiqni ko'rsatmaslik uchun.
Real-vaqt yetkazish: server → LiveKit data-channel → barcha klientlar:

```jsonc
{"kind":"hand","identity":"…","name":"…","raised":true,"at":1730000000000}
{"kind":"hand","act":"lower_all"}
{"kind":"reaction","emoji":"👍","name":"Ali","identity":"…"}  // identity — o'z echo'sini filtrlash uchun
{"kind":"policy","mute_on_entry":true,"allow_self_unmute":false}  // SERVERDAN (mute-all dan keyin)
```

**Reaksiyalar (№14)** — server hech nima SAQLAMAYDI (efemer), faqat tarqatadi.
Ruxsat etilgan to'plam server tomonda ham qulflangan (aks holda ochiq endpoint
moderatsiyasiz matn kanaliga aylanardi):

```
👍 👏 ❤️ 😂 😮 🎉 ✋
```

Ro'yxat klientlarda konstanta (`frontend/src/livekit/Controls.jsx: REACTIONS`) —
backend uni bermaydi, lekin ustidan tekshiradi. Yangi emoji qo'shilsa IKKALA
joyga ham qo'shiladi, aks holda 400 `unsupported reaction`.
Guest tokenida `canPublishData=true` (klient o'zi ham optimistik ko'rsatishi
uchun), lekin **haqiqiy tarqatish server orqali** — shu sababli ban/tezlik/
to'plam cheklovlari chetlab o'tilmaydi.

## Chat — ishtirokchi yo'li va shaxsiy xabar

| Method | Yo'l | Auth | Tana / Query | Javob |
|---|---|---|---|---|
| POST | `/api/v1/rooms/:lessonID/chat` | room-token | `{token, body, to?}` | 201 · 429 (5 s da 5) |
| GET | `/api/v1/rooms/:lessonID/chat` | room-token | `?token=&before=&limit=` | `{data:[ChatMessage]}` |
| POST | `/api/v1/rooms/:lessonID/chat/upload?token=` | room-token | multipart: `file,body?,to?` | 201 · 400 · 429 |
| POST | `/api/v1/lessons/:id/chat` | JWT (mentor) | `{body, to?}` | 201 |
| GET | `/api/v1/lessons/:id/chat` | JWT (mentor) | `?before=&limit=` | `{data:[ChatMessage]}` |
| POST | `/api/v1/lessons/:id/chat/upload` | JWT (mentor) | multipart: `file,body?,to?` | 201 · 400 · 429 |
| DELETE | `/api/v1/lessons/:id/chat/:messageID` | JWT (mentor) | — | 204 · 403 · 404 |

`to` — qabul qiluvchi LiveKit identity'si. Bo'sh → xonaga (ommaviy).
`ChatMessage.to_identity` — `null` bo'lsa ommaviy.

**Ko'rinuvchanlik:** shaxsiy xabarni faqat yuboruvchi va qabul qiluvchi oladi. Filtr SQL'da
qo'llanadi va yetkazish `destination_identities` bilan bo'ladi — begona klientga xabar
umuman bormaydi.

### Moderatsiya — xabarni o'chirish (№6)

Faqat **dars egasi**. Yumshoq o'chirish: qator DB'da qoladi (`deleted_at`/`deleted_by`
— moderatsiya izi), lekin **hech bir tarix so'rovida qaytmaydi** — mentorga ham.
Qabrtosh («xabar o'chirilgan») QOLDIRILMAYDI: u o'chirilgan joyni belgilab,
buzg'unchiga aynan u xohlagan e'tiborni berardi (Zoom ham izsiz olib tashlaydi).

Takroriy o'chirish → **404** (atomik `WHERE deleted_at IS NULL`), begona mentor → **403**,
yaroqsiz ID → **404** (500 emas).

Jonli xonadagi klientlarga data-channel orqali:

```jsonc
{"kind":"chat_deleted","id":"<message_id>","lesson_id":"…","deleted_by":"<mentor_id>"}
```

Xabar MAZMUNI yuborilmaydi. Ommaviy xabar hodisasi butun xonaga, **shaxsiy xabarniki
faqat ikki tomonga** — aks holda DM'ning mavjudligi oshkor bo'lardi.

### Fayl ulashish (№15)

`multipart/form-data`, maydon nomi **`file`**. Mentor VA o'quvchi yubora oladi.

O'quvchi yo'lida token **`?token=` query'da** berilsin (tana `token` maydoni ham
ishlaydi, lekin eskirgan): query'dagi token multipart TANA O'QILMASDAN
tekshiriladi, ya'ni yaroqsiz token darhol 401 oladi va server 20 MB'ni
vaqtinchalik faylga yozmaydi.

- **Maks hajm:** 20 MB (oshsa 400 `file is too large`)
- **Ruxsat etilgan turlar:** `.jpg .jpeg .png .gif .webp .pdf .docx .xlsx .pptx .doc .xls .ppt .txt .csv`
- **Tur tekshiruvi ikki bosqichli:** kengaytma allowlist + mazmun sniff
  (`http.DetectContentType`). Mos kelmasa 400 `file content does not match its extension`.
  Klient yuborgan `Content-Type` ISHONCHSIZ deb qaraladi va e'tiborga olinmaydi;
  MinIO'ga bizning kanonik MIME yoziladi (HTML/JS yuklab XSS qilib bo'lmaydi).
- **Tezlik:** identity bo'yicha daqiqasiga 5 ta (429). Ustozga ham tegadi.
- Fayl nomidagi yo'l tashlanadi (`../../etc/x.png` → `x.png`).

Muvaffaqiyatli javob — oddiy `ChatMessage`, `file` maydoni bilan:

```jsonc
{"data":{
  "id":"…","lesson_id":"…","sender_identity":"guest_ab12","sender_name":"Ali",
  "body":"Uy ishi","created_at":"2026-07-31T09:12:00Z",
  "file":{"name":"uy_ishi.pdf","size":184320,"mime":"application/pdf",
          "url":"https://minio…?X-Amz-Signature=…","expires_in_s":3600}
}}
```

`file.url` **presigned va vaqtinchalik (1 soat)** — bazada saqlanmaydi, har javobda
qayta imzolanadi. Tarixdan kelgan eski xabarlar ham har doim amaldagi havola bilan
qaytadi, ya'ni klient havolani keshlab qo'ymasligi kerak. Obyekt kaliti tashqariga
CHIQMAYDI.

### Poll: ikki rejim + «E'lon qilish» (№7)

`POST /lessons/:id/polls` da `results_visibility`:

| Qiymat | Ma'nosi |
|---|---|
| `mentor_only` (**default**) | Natijani FAQAT mentor ko'radi. E'lon qilib ham bo'lmaydi (publish → **400**). |
| `public` | Natijani o'quvchi ham ko'radi, lekin FAQAT mentor «E'lon qilish» bosgach. |

Maydon yuborilmasa `mentor_only` — yopiq tomon (eski klientlar natijani tasodifan
ochib yubormasin).

`GET /polls/:id/results?token=` javobi:
- **mentor** (host room-token, identity = mentor ID) → doim 200;
- **o'quvchi** → 200 faqat `results_visibility=public` VA `results_published_at != null`;
  aks holda **403** `poll results are not published yet` (bo'sh natija EMAS: «0 ovoz»
  bilan «ko'rsatilmaydi» ni farqlab bo'lmasa klient noto'g'ri diagramma chizardi).

**Yopish ≠ e'lon qilish:** `POST /polls/:id/close` faqat ovoz berishni to'xtatadi,
natijani ochmaydi.

`POST /lessons/:id/polls/:pollID/publish` (mentor) — 200 `PollResults`, idempotent
(takroriy bosish e'lon vaqtini surmaydi). Xonaga data-channel orqali:

```jsonc
{"kind":"poll_published","results":{"poll":{…},"counts":[3,7],"total":10}}
```

Natijaning O'ZI ham yuboriladi — 300 kishilik xonada har biri alohida so'rov
yuborsa bu 300 ta ortiqcha so'rov bo'lardi.

## Dars arxivi (№20) va chat transkripti (№21)

PRODUCT.md «Dars arxivi va Telegram saqlash» (2026-08-01). Ikkalasi ham
**mentor + dars egasi**; begona mentor → **403**, yo'q dars → **404**,
yaroqsiz UUID → **404** (mavjud bo'lmagan UUID bilan bir xil javob — loyihadagi
umumiy qoida).

### `GET /api/v1/lessons/:id/archive`

O'tgan dars sahifasi uchun **bitta so'rov**: video, chat va materiallar.
Uch alohida so'rov emas, chunki `offset_sec` chat bilan videoni bog'laydi va
ikkalasi bir xil `started_at` o'qishidan kelishi shart.

```jsonc
{"data":{
  "lesson": { /* Lesson (yuqoridagi shakl) */ },
  "recording": {
    "id":"a1b2…", "status":"ready", "duration_sec":3600, "size_bytes":128374912,
    "url":"https://minio…/rec/….mp4?X-Amz-…",   // presigned 1 soat; ready BO'LMASA null
    "expires_at":"2026-08-31T10:30:00Z"          // faqat ready da; retention=0 → null
  },
  "chat": [
    {"id":"…","sender_identity":"guest_a","sender_name":"Ali Valiyev",
     "body":"Ustoz, savol bor","to_identity":null,"file":null,
     "created_at":"2026-08-01T09:32:05Z","offset_sec":125}
  ],
  "materials": [
    {"name":"masala.pdf","size":12345,"mime":"application/pdf",
     "url":"https://minio…?X-Amz-…","created_at":"2026-08-01T09:40:03Z"}
  ]
}}
```

- **`recording: null`** — bu dars uchun yozuv umuman yo'q (xato emas; klient
  «yozuv yo'q» deb ko'rsatadi). Bir nechta yozuv bo'lsa ustuvorlik:
  `ready` → `expired` → `recording`/`processing` → `failed`, teng bo'lsa eng yangisi.
- **`recording.url`** faqat `status="ready"` da to'ldiriladi. `expired` da status
  **shundayligicha** qoladi va `url: null` — Telegramdan qaytarib olish oqimi
  (PRODUCT.md, 30 kun + 1 kunlik kesh) aynan shu statusga tayanadi.
- **`offset_sec`** — xabar dars boshidan necha soniyada yozilgani
  (`created_at − lesson.started_at`). Pleyerda vaqtni bosganda sakrash uchun.
  Hech qachon manfiy emas (kutish xonasidagi xabar → `0`). `started_at` bo'lmasa
  yozuvning `started_at` iga tushadi; ikkalasi ham bo'lmasa hammasi `0`.
- **`chat`** — eskidan yangiga, `deleted_at IS NULL` (moderatsiya qilingan xabar
  yo'q). Ko'rinuvchanlik `GET /lessons/:id/chat` bilan bir xil: ommaviy xabarlar
  + ustozning **o'z** shaxsiy yozishmalari (o'quvchilarning bir-biriga yozgani EMAS).
- **`materials`** — chatdagi fayl xabarlaridan yig'iladi (presigned havola bilan).
- Javob `Cache-Control: no-store` bilan keladi (ichida vaqtinchalik havolalar bor).

### `GET /api/v1/lessons/:id/chat/transcript?format=txt|html`

Zoom kabi — dars oxirida chat fayli. Javob **JSON emas**, faylning o'zi:

```
Content-Type: text/plain; charset=utf-8      (html → text/html; charset=utf-8)
Content-Disposition: attachment; filename="algebra-chat-2026-08-01.txt"
Cache-Control: no-store
```

- `format` berilmasa **`txt`**. Boshqa qiymat → **400** `format must be txt or html`.
- Fayl nomi dars sarlavhasidan yasaladi, lekin **faqat ASCII harf/raqam/tire**
  qoldiriladi (sarlavha in'yeksiyasining oldi olinadi); ASCII qolmasa `dars-chat-<sana>.<ext>`.
- Barcha vaqtlar **Asia/Tashkent (UTC+5)** da — DB UTC saqlaydi, hujjatni esa odam o'qiydi.

**TXT namunasi:**
```
Dars: Matematika 5-sinf
Sana: 01.08.2026 14:30
Mentor: Dilnoza Karimova
─────────────────────────────
14:32:05  Ali Valiyev: Ustoz, savol bor
14:32:40  Siz: Marhamat
14:35:12  Dilnoza (shaxsiy): rahmat, tushundim
14:40:03  Ali Valiyev: 📎 masala.pdf
```
Ustozning o'z xabarlari **`Siz`**, shaxsiy xabar **`(shaxsiy)`**, fayl **`📎 nom`**
(izoh bo'lsa `📎 nom — izoh`). Chat bo'sh bo'lsa `(Chatda xabar bo'lmagan)`.
Ko'p qatorli xabarning davomi 10 bo'sh joyga suriladi — o'quvchi tanasiga soxta
`HH:MM:SS  Ustoz: …` qatori yozib transkriptni qalbakilashtira olmasin.

**HTML** — bir faylli, inline CSS, Jonly brendida (qora fon + `#19d3a2`).
Tashqi resurs **yo'q**: internetsiz, Telegram ilovasidan ochiladi. Ism, matn,
fayl nomi va dars sarlavhasi HTML-escape qilinadi (hujjat brauzerda ochiladi,
mazmuni esa o'quvchi yozgan). Fayllarga havola **qo'yilmaydi** — presigned
havola bir soatda o'ladi va faylni butunlay yo'qolgandek ko'rsatardi;
materiallar ilovada (`/archive`) qoladi.
