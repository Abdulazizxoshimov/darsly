"""WebSocket — /ws (mentor, authed) va /ws/waitingroom (guest, ochiq) ulanishi.

Kontrakt (api-contract «WebSocket» + CLAUDE.md): brauzer header yubora olmaydi →
token `?token=`, guest `?request_id=`. LiveKit KERAK EMAS (WS hub mustaqil).
Sinxron `websockets.sync.client` ishlatiladi (pytest-asyncio shart emas).
"""
import pytest

from lib import rooms
from lib.ws import mentor_ws_url, waitingroom_ws_url


# ── Auth (HTTP darajasida — upgrade shart emas) ─────────────────
def test_ws_requires_token(client):
    # Auth middleware token'siz so'rovni 401 bilan to'xtatadi (upgrade'dan oldin).
    client.expect(client.get("/api/v1/ws"), 401)


def test_ws_invalid_token(client):
    client.expect(client.get("/api/v1/ws", params={"token": "chala.token.qiymat"}), 401)


# ── Haqiqiy handshake ───────────────────────────────────────────
@pytest.mark.ws
@pytest.mark.destructive
def test_mentor_ws_connects(mentor, ws_url):
    from websockets.sync.client import connect

    with connect(mentor_ws_url(ws_url, mentor.access), open_timeout=5):
        pass  # ulanish ochildi — hub tirik (pending snapshot kelishi mumkin)


@pytest.mark.ws
@pytest.mark.destructive
def test_guest_waitingroom_ws_connects(as_mentor, factory, client, ws_url):
    from websockets.sync.client import connect

    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=True)
    data = rooms.join(client, lesson["join_slug"], guest_name="WS Guest")
    request_id = data["request_id"]
    with connect(waitingroom_ws_url(ws_url, request_id), open_timeout=5):
        pass  # guest kutish-xonasi soketi ochildi (admit push shu yerdan keladi)
