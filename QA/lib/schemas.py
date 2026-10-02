"""Javob shakli (kontrakt) validatorlari.

`docs/api-contract.md` «Model shakllari» bo'limidan olingan. Maqsad — backend
web/mobil klient KUTGAN maydonlarni qaytarishini kafolatlash. Backend qo'shimcha
maydon qo'shsa test SINMAYDI (`additionalProperties` ochiq): faqat kontrakt
kafolatlagan maydonlar `required`. Backend maydonni OLIB TASHLASA yoki tipini
o'zgartirsa — shu yerda ushlanadi (web/mobil sindan oldin).

Ishlatish:
    from lib.schemas import validate, USER, LESSON, list_of
    data = client.data(resp)
    validate(data, USER)                    # bitta obyekt
    validate(body, list_of(LESSON))         # list konverti {data:[...], total, ...}
"""
from __future__ import annotations

from typing import Any

import jsonschema

# ── past-daraja yordamchi ────────────────────────────────────────


def validate(instance: Any, schema: dict) -> None:
    """jsonschema tekshiruvi; xato bo'lsa tushunarli AssertionError."""
    try:
        jsonschema.validate(instance=instance, schema=schema)
    except jsonschema.ValidationError as e:
        path = "/".join(str(p) for p in e.absolute_path) or "<root>"
        raise AssertionError(
            f"kontrakt buzildi @ {path}: {e.message}\n  qiymat: {instance!r}"
        ) from None


def obj(required: list[str], props: dict[str, Any] | None = None) -> dict:
    """Ochiq obyekt sxemasi — faqat `required` majburiy, qo'shimcha maydon erkin."""
    return {
        "type": "object",
        "required": required,
        "properties": props or {},
        "additionalProperties": True,
    }


def list_of(item: dict) -> dict:
    """Paginatsiyalangan list konverti: {data:[item], total, page, limit, total_pages}.

    Bo'sh ro'yxat `[]` bo'lishi shart (`null` emas — kontrakt qoidasi).
    Ishlatiladi: lessons, users, notifications.
    """
    return obj(
        ["data", "total", "page", "limit", "total_pages"],
        {
            "data": {"type": "array", "items": item},
            "total": {"type": "integer"},
            "page": {"type": "integer"},
            "limit": {"type": "integer"},
            "total_pages": {"type": "integer"},
        },
    )


def data_array(item: dict) -> dict:
    """Oddiy (paginatsiyasiz) ro'yxat: {data:[item]}.

    Ishlatiladi: blocklist, participants, chat history, polls list, recordings.
    """
    return obj(["data"], {"data": {"type": "array", "items": item}})


_STR = {"type": "string"}
_INT = {"type": "integer"}
_BOOL = {"type": "boolean"}
_NUM = {"type": "number"}

# ── konvert ──────────────────────────────────────────────────────

ERROR = obj(["code", "message"], {"code": _STR, "message": _STR})

TOKEN_PAIR = obj(
    ["access_token", "refresh_token"],
    {"access_token": _STR, "refresh_token": _STR},
)

# ── entity'lar (api-contract «Model shakllari») ──────────────────

USER = obj(
    ["id", "email", "full_name", "role", "is_active", "created_at"],
    {
        "id": _STR,
        "email": _STR,
        "full_name": _STR,
        "role": {"enum": ["admin", "mentor", "student", "guest"]},
        "is_active": _BOOL,
    },
)

LESSON = obj(
    [
        "id",
        "mentor_id",
        "title",
        "duration_min",
        "join_slug",
        "has_passcode",
        "is_locked",
        "is_recording_enabled",
        "is_waiting_room_enabled",
        "status",
        "created_at",
    ],
    {
        "id": _STR,
        "mentor_id": _STR,
        "title": _STR,
        "duration_min": _INT,
        "join_slug": _STR,
        "has_passcode": _BOOL,
        "is_locked": _BOOL,
        "status": {"enum": ["scheduled", "live", "ended", "cancelled"]},
    },
)

ROOM_TOKEN = obj(
    ["token", "ws_url", "room_name", "identity", "role"],
    {
        "token": _STR,
        "ws_url": _STR,
        "room_name": _STR,
        "identity": _STR,
        "role": {"enum": ["host", "participant"]},
    },
)

LESSON_PUBLIC = obj(
    ["id", "title", "mentor_name", "status", "has_passcode", "is_waiting_room_enabled"],
    {
        "id": _STR,
        "title": _STR,
        "mentor_name": _STR,
        "status": _STR,
        "has_passcode": _BOOL,
        "is_waiting_room_enabled": _BOOL,
    },
)

JOIN_LESSON_RESP = obj(
    ["lesson", "next_step"],
    {
        "lesson": LESSON_PUBLIC,
        # 4 qiymat (backend entity/joinlink.go bilan tasdiqlangan):
        "next_step": {"enum": ["join", "waiting_room", "lesson_ended", "waiting_for_host"]},
        "room": ROOM_TOKEN,
        "request_id": _STR,
    },
)

WAITING_ROOM_REQUEST = obj(
    ["id", "lesson_id", "requester_name", "status", "created_at"],
    {"id": _STR, "lesson_id": _STR, "requester_name": _STR, "status": _STR},
)

WAITING_ROOM_STATUS = obj(
    ["request_id", "status"],
    {
        "request_id": _STR,
        "status": {"enum": ["pending", "admitted", "rejected"]},
        "room": ROOM_TOKEN,
    },
)

ADMIT_ALL_RESP = obj(
    ["total", "admitted", "failed"],
    {"total": _INT, "admitted": _INT, "failed": _INT},
)

ROOM_PARTICIPANT = obj(
    ["identity", "name", "active", "audio_muted", "video_muted"],
    {"identity": _STR, "name": _STR, "active": _BOOL},
)

ROOM_STATE = obj(
    ["hands", "mute_on_entry", "allow_self_unmute"],
    {
        "hands": {
            "type": "array",
            "items": obj(["identity", "name", "raised_at"]),
        },
        "mute_on_entry": _BOOL,
        "allow_self_unmute": _BOOL,
    },
)

CHAT_FILE = obj(
    ["name", "size", "mime", "url", "expires_in_s"],
    {"name": _STR, "size": _INT, "mime": _STR, "url": _STR, "expires_in_s": _INT},
)

CHAT_MESSAGE = obj(
    ["id", "lesson_id", "sender_identity", "sender_name", "body", "created_at"],
    {
        "id": _STR,
        "lesson_id": _STR,
        "sender_identity": _STR,
        "sender_name": _STR,
        "body": _STR,
        "file": CHAT_FILE,
    },
)

POLL = obj(
    ["id", "lesson_id", "question", "options", "is_active", "results_visibility", "created_at"],
    {
        "id": _STR,
        "lesson_id": _STR,
        "question": _STR,
        "options": {"type": "array", "items": _STR},
        "is_active": _BOOL,
        "results_visibility": {"enum": ["mentor_only", "public"]},
    },
)

POLL_RESULTS = obj(
    ["poll", "counts", "total"],
    {"poll": POLL, "counts": {"type": "array", "items": _INT}, "total": _INT},
)

RECORDING = obj(
    ["id", "lesson_id", "status", "created_at"],
    {
        "id": _STR,
        "lesson_id": _STR,
        "status": {
            "enum": [
                "recording",
                "processing",
                "ready",
                "failed",
                "expired",
                "archived",
                "restoring",
            ]
        },
    },
)

RECORDING_DOWNLOAD = obj(
    ["url", "expires_in_s"],
    {"url": _STR, "expires_in_s": _INT},
)

RECORDING_RESTORE = obj(
    ["status", "poll_after_s"],
    {"status": {"enum": ["restoring", "ready"]}, "poll_after_s": _INT},
)

NOTIFICATION = obj(
    ["id", "user_id", "type", "title", "created_at"],
    {"id": _STR, "user_id": _STR, "type": _STR, "title": _STR},
)

UNREAD_COUNT = obj(["count"], {"count": _INT})

BLOCKLIST_ENTRY = obj(
    ["id", "identity", "display_name", "created_at"],
    {"id": _STR, "identity": _STR, "display_name": _STR},
)

TELEGRAM_LINK_STATUS = obj(
    ["enabled", "linked"],
    {"enabled": _BOOL, "linked": _BOOL},
)

TELEGRAM_LINK = obj(
    ["code", "expires_in_s"],
    {"code": _STR, "expires_in_s": _INT},
)

LESSON_ARCHIVE = obj(
    ["lesson", "chat", "materials"],
    {
        "lesson": LESSON,
        "recording": {"type": ["object", "null"]},
        "chat": {"type": "array"},
        "materials": {"type": "array"},
    },
)

APP_CONFIG = obj(["android"], {"android": {"type": "object"}})
