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

## Auth (himoyalangan)
| POST | `/auth/logout` | `{refresh_token}` | 204 |
| GET | `/auth/me` | — | `User` |

## Joinlink (ochiq, rate-limit 600/min/IP)
| GET | `/joinlink/:slug` | — | `LessonPublic` |
| POST | `/joinlink/:slug` | `{guest_name?,passcode?}` | `JoinLessonResp` |

## Waiting room
| GET | `/waitingroom/:id/status` | — (ochiq) | `WaitingRoomStatusResp` |
| GET | `/lessons/:id/waitingroom` | auth(mentor) | `[WaitingRoomRequest]` |
| POST | `/waitingroom/:id/admit` | auth(mentor) | `RoomToken` |
| POST | `/waitingroom/:id/reject` | auth(mentor) | 204 |

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
| POST | `/lessons/:id/mute-all` | — | 204 |
| POST | `/lessons/:id/participants/:identity/mute` | — | 204 |
| POST | `/lessons/:id/participants/:identity/remove` | — | 204 |
| POST | `/lessons/:id/participants/:identity/allow-speak` | — | 204 |
| POST | `/lessons/:id/participants/:identity/revoke-speak` | — | 204 |

## Chat (himoyalangan, mentor)
| GET | `/lessons/:id/chat` | — | `[ChatMessage]` |
| POST | `/lessons/:id/chat` | `{body}` | `ChatMessage` |
> Guest chat backend'da YO'Q — LiveKit data-channel orqali (client). Backend host chat'ni data-channel'ga ham broadcast qiladi (`ChatMessage` shakli).

## Polls
| GET | `/lessons/:id/polls` | auth(mentor) | `[Poll]` |
| POST | `/lessons/:id/polls` | `{question,options[2..10]}` | 201 `Poll` |
| POST | `/polls/:id/close` | auth(mentor) | `PollResults` |
| POST | `/polls/:id/vote` | ochiq `{token,option_index}` | 204 |
| GET | `/polls/:id/results` | ochiq | `PollResults` |
> Guest poll push backend'da YO'Q — host LiveKit data-channel orqali e'lon qiladi (client).

## Recording (himoyalangan, mentor)
| POST | `/lessons/:id/recording/start` | — | 201 `Recording` |
| GET | `/lessons/:id/recordings` | — | `[Recording]` |
| POST | `/recordings/:id/stop` | — | 204 |
| GET | `/recordings/:id/download` | — | `RecordingDownload` (presigned, 1h) |

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
- **Lesson**: `{id,mentor_id,title,description?,scheduled_at?,duration_min,recurrence_rule?,join_slug,has_passcode,is_locked,is_recording_enabled,is_waiting_room_enabled,status,started_at?,ended_at?,created_at,updated_at}`; status: `scheduled|live|ended|cancelled`
- **CreateLessonReq**: `{title,description?,scheduled_at?,duration_min?,recurrence_rule?,passcode?,is_recording_enabled,is_waiting_room_enabled}`
- **UpdateLessonReq**: `{title?,description?,scheduled_at?,duration_min?,passcode?,remove_passcode?,is_locked?,is_recording_enabled?,is_waiting_room_enabled?,status?}`
- **RoomToken**: `{token,ws_url,room_name,identity,role}`; role: `host|participant`
- **RoomParticipant**: `{identity,name,joined_at_ms,active,audio_muted,video_muted}`
- **LessonPublic**: `{id,title,mentor_name,scheduled_at?,status,has_passcode,is_waiting_room_enabled}`
- **JoinLessonResp**: `{lesson:LessonPublic,next_step:"waiting_room"|"join",room?:RoomToken,request_id?}`
- **WaitingRoomRequest**: `{id,lesson_id,requester_name,status,created_at,decided_at?}`
- **WaitingRoomStatusResp**: `{request_id,status:"pending"|"admitted"|"rejected",room?:RoomToken}`
- **ChatMessage**: `{id,lesson_id,sender_identity,sender_name,body,created_at}`
- **Poll**: `{id,lesson_id,question,options[],is_active,created_at,closed_at?}`
- **PollResults**: `{poll:Poll,counts[],total}`
- **Recording**: `{id,lesson_id,egress_id,status:"recording"|"processing"|"ready"|"failed",duration_sec,size_bytes,started_at,ended_at?,created_at}`
- **RecordingDownload**: `{url,expires_in_s,duration_sec,size_bytes}`
- **Notification**: `{id,user_id,type:"lesson_reminder"|"waiting_room"|"system",title,body,lesson_id?,read_at?,created_at}`
