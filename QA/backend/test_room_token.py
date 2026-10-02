"""Room / LiveKit token — host token + end (mentor).

Kontrakt (api-contract «Room/LiveKit»):
  POST /lessons/:id/token → RoomToken (host; egasi bo'lmasa 403; yo'q dars 404)
  POST /lessons/:id/end   → 204
Ijobiy token/end LiveKit SFU'ni talab qiladi → `requires_livekit` bilan guard.
Negativ (auth/RBAC/ownership) LiveKit'dan OLDIN qaytadi → guardsiz.
"""
import pytest

from lib.schemas import ROOM_TOKEN, validate

pytestmark = pytest.mark.destructive

NIL_UUID = "00000000-0000-0000-0000-000000000000"


# ── Host token — ijobiy (LiveKit kerak) ─────────────────────────
def test_host_token_success(requires_livekit, as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    r = as_mentor.post(f"/api/v1/lessons/{lesson['id']}/token")
    as_mentor.expect(r, 200)
    data = as_mentor.data(r)
    validate(data, ROOM_TOKEN)
    assert data["role"] == "host"
    assert data["token"] and data["ws_url"]


# ── Host token — negativ (LiveKit kerak emas) ───────────────────
def test_host_token_requires_auth(client):
    client.expect(client.post(f"/api/v1/lessons/{NIL_UUID}/token"), 401)


def test_host_token_student_forbidden(as_user):
    as_user.expect(as_user.post(f"/api/v1/lessons/{NIL_UUID}/token"), 403)


def test_host_token_other_mentor_forbidden(factory):
    owner = factory.mentor_client()
    other = factory.mentor_client()
    lesson = factory.lesson(owner)
    r = other.post(f"/api/v1/lessons/{lesson['id']}/token")
    assert r.status_code in (403, 404), r.status_code


def test_host_token_nonexistent_lesson(as_mentor):
    r = as_mentor.post(f"/api/v1/lessons/{NIL_UUID}/token")
    assert r.status_code == 404, r.status_code


# ── End — negativ ───────────────────────────────────────────────
def test_end_requires_auth(client):
    client.expect(client.post(f"/api/v1/lessons/{NIL_UUID}/end"), 401)


def test_end_student_forbidden(as_user):
    as_user.expect(as_user.post(f"/api/v1/lessons/{NIL_UUID}/end"), 403)


def test_end_other_mentor_forbidden(factory):
    owner = factory.mentor_client()
    other = factory.mentor_client()
    lesson = factory.lesson(owner)
    r = other.post(f"/api/v1/lessons/{lesson['id']}/end")
    assert r.status_code in (403, 404), r.status_code
