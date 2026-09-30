"""Xona / room-token yordamchilari — test yozuvchi uchun qayta ishlatiladigan resurs.

Chat / poll / roomstate testlari guest'ning LiveKit **room-token**'iga tayanadi
(guest'da JWT yo'q). Bu yordamchi host va guest tokenlarini olishning takroriy
qadamlarini bir joyga yig'adi. TESTLARNING O'ZI keyin yoziladi — bu faqat asbob.

Kontrakt (docs/api-contract.md):
  POST /lessons/:id/token        (mentor JWT)   → RoomToken (host)
  GET  /joinlink/:slug           (ochiq)        → LessonPublic (preview)
  POST /joinlink/:slug           (ochiq)        → JoinLessonResp (join)
  POST /waitingroom/:id/admit    (mentor JWT)   → RoomToken (guest, admit)
"""
from __future__ import annotations

from .client import Client


def host_token(owner: Client, lesson_id: str) -> dict:
    """Mentor uchun host RoomToken (dars faol bo'lishi kerak)."""
    resp = owner.post(f"/api/v1/lessons/{lesson_id}/token")
    owner.expect(resp, 200)
    return owner.data(resp)


def preview_join(client: Client, slug: str) -> dict:
    """Joinlink preview — LessonPublic."""
    resp = client.get(f"/api/v1/joinlink/{slug}")
    client.expect(resp, 200)
    return client.data(resp)


def join(client: Client, slug: str, guest_name: str = "QA Guest", passcode: str | None = None) -> dict:
    """Joinlink join — JoinLessonResp (next_step: join | waiting_room | lesson_ended)."""
    body: dict = {"guest_name": guest_name}
    if passcode is not None:
        body["passcode"] = passcode
    resp = client.post(f"/api/v1/joinlink/{slug}", json=body)
    client.expect(resp, 200)
    return client.data(resp)


def admit(owner: Client, request_id: str) -> dict:
    """Kutish xonasidan guest'ni kiritish → guest RoomToken."""
    resp = owner.post(f"/api/v1/waitingroom/{request_id}/admit")
    owner.expect(resp, 200)
    return owner.data(resp)
