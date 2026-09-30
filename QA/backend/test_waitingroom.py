"""Kutish xonasi — list / status / admit / reject / admit-all.

Kontrakt (api-contract «Waiting room»):
  GET  /lessons/:id/waitingroom       (mentor) → [WaitingRoomRequest]
  GET  /waitingroom/:id/status        (ochiq)  → WaitingRoomStatusResp
  POST /waitingroom/:id/admit         (mentor) → RoomToken   · 2-marta → 409
  POST /waitingroom/:id/reject        (mentor) → 204
  POST /lessons/:id/waitingroom/admit-all      → AdmitAllResp {total,admitted,failed}
Admit RoomToken mint qiladi → LiveKit kerak (`requires_livekit`). List/status/reject
va bo'sh admit-all guardsiz.
"""
import pytest

from lib import rooms
from lib.schemas import ROOM_TOKEN, WAITING_ROOM_REQUEST, WAITING_ROOM_STATUS, data_array, validate

pytestmark = pytest.mark.destructive

NIL_UUID = "00000000-0000-0000-0000-000000000000"


def _pending_request(as_mentor, factory, client):
    """Kutish-xonasi yoqilgan dars + guest join → (lesson, request_id)."""
    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=True)
    data = rooms.join(client, lesson["join_slug"], guest_name="Kutuvchi")
    assert data["next_step"] == "waiting_room"
    return lesson, data["request_id"]


# ── List (mentor) ───────────────────────────────────────────────
def test_list_waitingroom_own(as_mentor, factory, client):
    lesson, _ = _pending_request(as_mentor, factory, client)
    r = as_mentor.get(f"/api/v1/lessons/{lesson['id']}/waitingroom")
    as_mentor.expect(r, 200)
    validate(r.json(), data_array(WAITING_ROOM_REQUEST))
    assert len(as_mentor.data(r)) >= 1


def test_list_waitingroom_other_mentor_forbidden(factory, client):
    owner = factory.mentor_client()
    other = factory.mentor_client()
    lesson, _ = _pending_request(owner, factory, client)
    r = other.get(f"/api/v1/lessons/{lesson['id']}/waitingroom")
    assert r.status_code in (403, 404), r.status_code


def test_list_waitingroom_requires_auth(client):
    client.expect(client.get(f"/api/v1/lessons/{NIL_UUID}/waitingroom"), 401)


# ── Status (ochiq) ──────────────────────────────────────────────
def test_status_public_pending(as_mentor, factory, client):
    _, rid = _pending_request(as_mentor, factory, client)
    r = client.get(f"/api/v1/waitingroom/{rid}/status")
    client.expect(r, 200)
    data = client.data(r)
    validate(data, WAITING_ROOM_STATUS)
    assert data["status"] == "pending"


# ── Reject (mentor) ─────────────────────────────────────────────
def test_reject_request(as_mentor, factory, client):
    _, rid = _pending_request(as_mentor, factory, client)
    r = as_mentor.post(f"/api/v1/waitingroom/{rid}/reject")
    as_mentor.expect(r, 204)
    # Reject'dan keyin status rejected bo'lishi kerak.
    status = client.data(client.get(f"/api/v1/waitingroom/{rid}/status"))
    assert status["status"] == "rejected"


def test_reject_requires_auth(client):
    client.expect(client.post(f"/api/v1/waitingroom/{NIL_UUID}/reject"), 401)


def test_reject_student_forbidden(as_user):
    as_user.expect(as_user.post(f"/api/v1/waitingroom/{NIL_UUID}/reject"), 403)


# ── Admit (mentor) — LiveKit kerak ──────────────────────────────
def test_admit_returns_room_token(requires_livekit, as_mentor, factory, client):
    _, rid = _pending_request(as_mentor, factory, client)
    data = rooms.admit(as_mentor, rid)
    validate(data, ROOM_TOKEN)
    assert data["role"] == "participant"


def test_double_admit_conflict(requires_livekit, as_mentor, factory, client):
    """Atomik TransitionFromPending — ikkinchi admit 409 (TOCTOU yo'q)."""
    _, rid = _pending_request(as_mentor, factory, client)
    rooms.admit(as_mentor, rid)
    r2 = as_mentor.post(f"/api/v1/waitingroom/{rid}/admit")
    as_mentor.expect(r2, 409)


def test_admit_requires_auth(client):
    client.expect(client.post(f"/api/v1/waitingroom/{NIL_UUID}/admit"), 401)


def test_admit_student_forbidden(as_user):
    as_user.expect(as_user.post(f"/api/v1/waitingroom/{NIL_UUID}/admit"), 403)


# ── Admit-all ───────────────────────────────────────────────────
def test_admit_all_requires_live_lesson(as_mentor, factory):
    """Dars LIVE emas (scheduled) → 400 "lesson is not live".

    Empty-queue → 200 {total:0} holati LIVE dars talab qiladi; darsni «live» qilish
    real LiveKit ishtirokchisini talab qiladi (black-box HTTP chegarasi — README GAP).
    """
    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=True)
    r = as_mentor.post(f"/api/v1/lessons/{lesson['id']}/waitingroom/admit-all")
    as_mentor.expect(r, 400)


def test_admit_all_requires_auth(client):
    client.expect(client.post(f"/api/v1/lessons/{NIL_UUID}/waitingroom/admit-all"), 401)


def test_admit_all_student_forbidden(as_user):
    as_user.expect(as_user.post(f"/api/v1/lessons/{NIL_UUID}/waitingroom/admit-all"), 403)
