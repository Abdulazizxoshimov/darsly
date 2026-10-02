"""Room-state — qo'l ko'tarish / reaksiya / holat (room-token bilan, guest).

Kontrakt (api-contract «Xona holati»):
  POST /rooms/:id/hand     {token,raised}  → 204 (token yo'q → 401)
  POST /rooms/:id/reaction {token,emoji}   → 204 · ruxsatsiz emoji → 400
  GET  /rooms/:id/state?token=             → RoomState
Ijobiy yo'llar room-token talab qiladi (host_token orqali) → LiveKit guard.
Token'siz 401 negativlari guardsiz.
"""
import pytest

from lib import rooms
from lib.schemas import ROOM_STATE, validate

pytestmark = pytest.mark.destructive

NIL_UUID = "00000000-0000-0000-0000-000000000000"

ALLOWED_EMOJI = "👍"
FORBIDDEN_EMOJI = "🚀"


# ── Token'siz → 401 ─────────────────────────────────────────────
def test_hand_requires_token(client):
    client.expect(client.post(f"/api/v1/rooms/{NIL_UUID}/hand", json={"raised": True}), 401)


def test_reaction_requires_token(client):
    client.expect(client.post(f"/api/v1/rooms/{NIL_UUID}/reaction", json={"emoji": ALLOWED_EMOJI}), 401)


def test_state_requires_token(client):
    client.expect(client.get(f"/api/v1/rooms/{NIL_UUID}/state"), 401)


# ── Ijobiy (room-token) — LiveKit kerak ─────────────────────────
def test_raise_hand_and_read_state(requires_livekit, as_mentor, factory, client):
    lesson = factory.lesson(as_mentor)
    tok = rooms.host_token(as_mentor, lesson["id"])["token"]
    lid = lesson["id"]
    client.expect(client.post(f"/api/v1/rooms/{lid}/hand", json={"token": tok, "raised": True}), 204)
    r = client.get(f"/api/v1/rooms/{lid}/state", params={"token": tok})
    client.expect(r, 200)
    validate(client.data(r), ROOM_STATE)


def test_reaction_allowed_emoji(requires_livekit, as_mentor, factory, client):
    lesson = factory.lesson(as_mentor)
    tok = rooms.host_token(as_mentor, lesson["id"])["token"]
    r = client.post(f"/api/v1/rooms/{lesson['id']}/reaction", json={"token": tok, "emoji": ALLOWED_EMOJI})
    client.expect(r, 204)


def test_reaction_forbidden_emoji(requires_livekit, as_mentor, factory, client):
    lesson = factory.lesson(as_mentor)
    tok = rooms.host_token(as_mentor, lesson["id"])["token"]
    r = client.post(f"/api/v1/rooms/{lesson['id']}/reaction", json={"token": tok, "emoji": FORBIDDEN_EMOJI})
    client.expect(r, 400)
