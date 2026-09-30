"""Mobil kontrakt — backend mobil ilova (Android/Kotlin) KUTGAN shaklni qaytaradimi.

Mobil MENTOR ilovasi (join/guest oqimi yo'q). DTO'lar `DarslyApi.kt`/`Models.kt` dan.
Eng muhim mobil-spetsifik farq — ENVELOPE ikki xil:
  · `total` BOR (ListEnvelope): GET /lessons, /notifications, /lessons/:id/participants, /rooms/:id/chat
  · `total` YO'Q (Envelope<List>): GET /blocklist, /lessons/:id/polls, /lessons/:id/waitingroom
Mobil `total` yo'qni default 0 bilan o'qiydi — lekin paginatsiya buzilmasin uchun
paginatsiyalangan ro'yxatlarda `total` KELISHI shart. Bu yerda ikkala shakl ham tekshiriladi.
"""
import pytest

from lib import rooms
from lib.schemas import (
    APP_CONFIG,
    LESSON,
    RECORDING,
    ROOM_TOKEN,
    USER,
    data_array,
    validate,
)

pytestmark = pytest.mark.destructive


# ── Envelope: TokenPair / User (mobil access_token, full_name, role) ─
def test_login_returns_token_pair_fields(user, client):
    from lib.auth import login

    data = login(client, user.email, user.password)
    assert "access_token" in data and "refresh_token" in data, "mobil TokenPair kalitlari"


def test_me_user_fields(as_mentor):
    data = as_mentor.data(as_mentor.get("/api/v1/auth/me"))
    validate(data, USER)
    # Mobil User DTO aynan shu kalitlarni o'qiydi (@Json).
    for k in ("id", "email", "full_name", "role", "is_active", "created_at"):
        assert k in data, f"mobil User.{k} kutadi"


# ── Lesson DTO ──────────────────────────────────────────────────
def test_lesson_dto_fields(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    validate(lesson, LESSON)
    for k in ("id", "title", "join_slug", "status", "is_waiting_room_enabled",
              "is_recording_enabled", "has_passcode", "is_locked",
              "mute_on_entry", "allow_self_unmute", "duration_min", "created_at"):
        assert k in lesson, f"mobil Lesson.{k} kutadi"


# ── Paginatsiyalangan ro'yxatlar → `total` BOR (mobil ListEnvelope) ─
def test_lessons_list_has_total(as_mentor, factory):
    factory.lesson(as_mentor)
    body = as_mentor.get("/api/v1/lessons").json()
    assert isinstance(body.get("data"), list)
    assert "total" in body, "mobil ListEnvelope<Lesson>.total kutadi"


def test_notifications_list_has_total(as_mentor):
    body = as_mentor.get("/api/v1/notifications").json()
    assert isinstance(body.get("data"), list)
    assert "total" in body, "mobil ListEnvelope<Notification>.total kutadi"


# ── Oddiy ro'yxatlar → `{data:[...]}` (mobil Envelope<List>, total shart emas) ─
def test_blocklist_is_data_array(as_mentor):
    body = as_mentor.get("/api/v1/blocklist").json()
    assert isinstance(body.get("data"), list), "mobil Envelope<List<BlocklistEntry>> kutadi"


def test_polls_list_is_data_array(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    body = as_mentor.get(f"/api/v1/lessons/{lesson['id']}/polls").json()
    assert isinstance(body.get("data"), list), "mobil Envelope<List<Poll>> kutadi"


def test_waitingroom_list_is_data_array(as_mentor, factory, client):
    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=True)
    rooms.join(client, lesson["join_slug"], guest_name="M")
    body = as_mentor.get(f"/api/v1/lessons/{lesson['id']}/waitingroom").json()
    assert isinstance(body.get("data"), list), "mobil Envelope<List<WaitingRoomRequest>> kutadi"
    assert len(body["data"]) >= 1
    # Mobil REST DTO `id` kaliti bilan o'qiydi (WS esa request_id).
    assert "id" in body["data"][0], "mobil WaitingRoomRequest.id (REST) kutadi"


# ── unread-count / app-config ───────────────────────────────────
def test_unread_count_field(as_mentor):
    data = as_mentor.data(as_mentor.get("/api/v1/notifications/unread-count"))
    assert "count" in data, "mobil UnreadCount.count kutadi"


def test_app_config_android_fields(client):
    data = client.data(client.get("/api/v1/app-config"))
    validate(data, APP_CONFIG)
    android = data["android"]
    for k in ("min_version", "latest_version", "apk_url", "force_update", "release_notes"):
        assert k in android, f"mobil AndroidConfig.{k} kutadi (forced-update gate)"


# ── Recording DTO (list) ────────────────────────────────────────
def test_recording_list_data_array(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    body = as_mentor.get(f"/api/v1/lessons/{lesson['id']}/recordings").json()
    validate(body, data_array(RECORDING))


# ── RoomToken (mobil room_name/identity ham o'qiydi) — LiveKit kerak ─
def test_room_token_mobile_fields(requires_livekit, as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    data = rooms.host_token(as_mentor, lesson["id"])
    validate(data, ROOM_TOKEN)
    for k in ("token", "ws_url", "room_name", "identity", "role"):
        assert k in data, f"mobil RoomToken.{k} kutadi"
