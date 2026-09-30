"""E2E — guest oqimi (joinlink → preview → join → waiting → reject).

Ko'p-endpointli oqim, HTTP darajasida. Admit → room-token qismi LiveKit talab
qiladi (`requires_livekit`). Preview/join/passcode/lock/reject LiveKit'siz.
"""
import pytest

from lib import rooms
from lib.schemas import JOIN_LESSON_RESP, LESSON_PUBLIC, ROOM_TOKEN, validate

pytestmark = pytest.mark.destructive


def test_guest_join_without_waiting_room(as_mentor, factory, client):
    """Kutish xonasi o'chiq → preview + join. next_step darsning live-holatiga bog'liq
    (host kirmagan → waiting_for_host; live → join) — ikkalasi ham davom etish holati."""
    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=False)
    preview = rooms.preview_join(client, lesson["join_slug"])
    validate(preview, LESSON_PUBLIC)
    assert preview["has_passcode"] is False

    joinresp = rooms.join(client, lesson["join_slug"], guest_name="Guest A")
    validate(joinresp, JOIN_LESSON_RESP)
    assert joinresp["next_step"] in ("waiting_for_host", "join"), joinresp["next_step"]


def test_guest_passcode_flow(as_mentor, factory, client):
    """Preview parolni ko'rsatadi → noto'g'ri parol 401 → to'g'ri parol o'tadi."""
    lesson = factory.lesson(as_mentor, passcode="4321", is_waiting_room_enabled=False)
    assert rooms.preview_join(client, lesson["join_slug"])["has_passcode"] is True

    wrong = client.post(f"/api/v1/joinlink/{lesson['join_slug']}", json={"guest_name": "G", "passcode": "0000"})
    client.expect(wrong, 401)

    ok = rooms.join(client, lesson["join_slug"], guest_name="G", passcode="4321")
    assert ok["next_step"] in ("waiting_for_host", "join")


def test_guest_waiting_room_reject_flow(as_mentor, factory, client):
    """Guest kutish xonasiga tushadi → mentor ro'yxatda ko'radi → reject → status rejected."""
    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=True)
    joinresp = rooms.join(client, lesson["join_slug"], guest_name="Kutuvchi")
    assert joinresp["next_step"] == "waiting_room"
    rid = joinresp["request_id"]

    queue = as_mentor.data(as_mentor.get(f"/api/v1/lessons/{lesson['id']}/waitingroom"))
    assert any(r["id"] == rid for r in queue), "mentor navbatda so'rovni ko'rishi kerak"

    as_mentor.expect(as_mentor.post(f"/api/v1/waitingroom/{rid}/reject"), 204)
    status = client.data(client.get(f"/api/v1/waitingroom/{rid}/status"))
    assert status["status"] == "rejected"


def test_guest_admit_flow(requires_livekit, as_mentor, factory, client):
    """To'liq admit oqimi — LiveKit kerak: guest kiritiladi → participant room-token."""
    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=True)
    rid = rooms.join(client, lesson["join_slug"], guest_name="Admit Guest")["request_id"]
    token = rooms.admit(as_mentor, rid)
    validate(token, ROOM_TOKEN)
    assert token["role"] == "participant"
    status = client.data(client.get(f"/api/v1/waitingroom/{rid}/status"))
    assert status["status"] == "admitted"
