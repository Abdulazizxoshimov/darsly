"""Joinlink — preview / join / passcode / lock (ochiq endpoint).

Kontrakt (api-contract «Joinlink» + usecase/joinlink):
  GET  /joinlink/:slug  → LessonPublic (yo'q slug → 404)
  POST /joinlink/:slug  {guest_name?,passcode?} → JoinLessonResp
    · locked            → 403 "lesson is locked by the host"
    · passcode yo'q     → 401 "passcode required"
    · noto'g'ri passcode→ 401 "invalid passcode"
  next_step: join | waiting_room | lesson_ended | waiting_for_host
"""
import pytest

from lib import rooms
from lib.schemas import JOIN_LESSON_RESP, LESSON_PUBLIC, validate

pytestmark = pytest.mark.destructive


# ── Preview ─────────────────────────────────────────────────────
def test_preview_existing(as_mentor, factory):
    lesson = factory.lesson(as_mentor, title="Kimyo")
    data = rooms.preview_join(as_mentor, lesson["join_slug"])
    validate(data, LESSON_PUBLIC)
    assert data["title"] == "Kimyo"
    assert data["has_passcode"] is False


def test_preview_nonexistent_slug(client):
    r = client.get("/api/v1/joinlink/yoq-boaqin-slug-000")
    assert r.status_code == 404, r.status_code


# ── Join (parolsiz, kutish xonasisiz) ───────────────────────────
def test_join_scheduled_lesson_waits_for_host(as_mentor, factory, client):
    """Dars `scheduled` (host kirmagan), kutish xonasi o'chiq → waiting_for_host."""
    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=False)
    data = rooms.join(client, lesson["join_slug"], guest_name="Ali")
    validate(data, JOIN_LESSON_RESP)
    assert data["next_step"] == "waiting_for_host"
    assert "room" not in data or data.get("room") is None


def test_join_with_waiting_room_returns_request_id(as_mentor, factory, client):
    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=True)
    data = rooms.join(client, lesson["join_slug"], guest_name="Vali")
    validate(data, JOIN_LESSON_RESP)
    assert data["next_step"] == "waiting_room"
    assert data.get("request_id"), "waiting_room bo'lsa request_id kerak"


# ── Passcode ────────────────────────────────────────────────────
def test_join_missing_passcode(as_mentor, factory, client):
    lesson = factory.lesson(as_mentor, passcode="1234")
    assert rooms.preview_join(as_mentor, lesson["join_slug"])["has_passcode"] is True
    r = client.post(f"/api/v1/joinlink/{lesson['join_slug']}", json={"guest_name": "X"})
    client.expect(r, 401)  # "passcode required"


def test_join_wrong_passcode(as_mentor, factory, client):
    lesson = factory.lesson(as_mentor, passcode="1234")
    r = client.post(
        f"/api/v1/joinlink/{lesson['join_slug']}",
        json={"guest_name": "X", "passcode": "9999"},
    )
    client.expect(r, 401)  # "invalid passcode"


def test_join_correct_passcode(as_mentor, factory, client):
    lesson = factory.lesson(as_mentor, passcode="1234", is_waiting_room_enabled=False)
    data = rooms.join(client, lesson["join_slug"], guest_name="X", passcode="1234")
    validate(data, JOIN_LESSON_RESP)
    assert data["next_step"] in ("waiting_for_host", "join")


# ── Lock ────────────────────────────────────────────────────────
def test_join_locked_lesson_forbidden(as_mentor, factory, client):
    lesson = factory.lesson(as_mentor)
    as_mentor.expect(as_mentor.patch(f"/api/v1/lessons/{lesson['id']}", json={"is_locked": True}), 200)
    r = client.post(f"/api/v1/joinlink/{lesson['join_slug']}", json={"guest_name": "X"})
    client.expect(r, 403)  # "lesson is locked by the host"
