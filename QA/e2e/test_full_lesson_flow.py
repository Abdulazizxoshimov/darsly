"""E2E — to'liq dars oqimi real-time WS bilan (mentor push + guest push).

Oqim (LiveKit'siz qism):
  mentor dars yaratadi (waiting room) → mentor /ws ga ulanadi → guest join →
  mentor WS orqali `waiting_room.request` oladi → guest /ws/waitingroom ga ulanadi →
  mentor reject → guest WS orqali `waiting_room.rejected` oladi → status rejected.

To'liq admit→token→chat qismi LiveKit ishtirokchisini talab qiladi (README GAP).
"""
import pytest

from lib import rooms
from lib.ws import mentor_ws_url, sync_connect, wait_for_type, waitingroom_ws_url

pytestmark = [pytest.mark.slow, pytest.mark.ws, pytest.mark.destructive]


def test_waiting_room_realtime_reject_flow(mentor, as_mentor, factory, client, ws_url):
    # as_mentor va mentor AYNI foydalanuvchi (as_mentor `mentor` fixture'iga tayanadi).
    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=True)

    # 1) Mentor real-time kanalga ulanadi (so'rovni WS orqali olish uchun).
    with sync_connect(mentor_ws_url(ws_url, mentor.access)) as mentor_ws:
        # 2) Guest kutish xonasiga so'rov yuboradi.
        joinresp = rooms.join(client, lesson["join_slug"], guest_name="Real-time Guest")
        rid = joinresp["request_id"]

        # 3) Mentor WS orqali `waiting_room.request` oladi (pending snapshot o'tkaziladi).
        req = wait_for_type(mentor_ws, "waiting_room.request", timeout=8)
        assert req.get("payload") is not None

        # 4) Guest o'z kutish-xonasi kanaliga ulanadi (qaror push'ini olish uchun).
        with sync_connect(waitingroom_ws_url(ws_url, rid)) as guest_ws:
            # 5) Mentor rad etadi.
            as_mentor.expect(as_mentor.post(f"/api/v1/waitingroom/{rid}/reject"), 204)
            # 6) Guest WS orqali `waiting_room.rejected` oladi.
            wait_for_type(guest_ws, "waiting_room.rejected", timeout=8)

    # 7) Public status ham rad etilganini tasdiqlaydi (polling fallback).
    status = client.data(client.get(f"/api/v1/waitingroom/{rid}/status"))
    assert status["status"] == "rejected"


def test_full_admit_flow_requires_livekit(requires_livekit, mentor, as_mentor, factory, client, ws_url):
    """Mentor WS so'rov oladi → admit → guest WS orqali `waiting_room.admitted` (payload=RoomToken)."""
    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=True)
    with sync_connect(mentor_ws_url(ws_url, mentor.access)) as mentor_ws:
        rid = rooms.join(client, lesson["join_slug"], guest_name="Admit RT")["request_id"]
        wait_for_type(mentor_ws, "waiting_room.request", timeout=8)
        with sync_connect(waitingroom_ws_url(ws_url, rid)) as guest_ws:
            token = rooms.admit(as_mentor, rid)
            assert token["role"] == "participant"
            admitted = wait_for_type(guest_ws, "waiting_room.admitted", timeout=8)
            assert admitted["payload"]["token"], "admitted payload RoomToken bo'lishi kerak"
