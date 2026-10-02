"""Chat — host yo'li (mentor JWT) + guest yo'li (room-token) + transcript + delete.

Kontrakt (api-contract «Chat»):
  POST   /lessons/:id/chat  {body,to?}            → 201 ChatMessage
  GET    /lessons/:id/chat                        → {data:[ChatMessage]}
  DELETE /lessons/:id/chat/:messageID            → 204 (2-marta → 404)
  GET    /lessons/:id/chat/transcript?format=…   → fayl (txt|html); boshqa format → 400
  POST   /rooms/:id/chat  {token,body,to?}        → 201 (guest, room-token)
Host yo'li LiveKit'siz ishlaydi (broadcast best-effort). Guest yo'li room-token → guard.
"""
import pytest

from lib import rooms
from lib.schemas import CHAT_FILE, CHAT_MESSAGE, data_array, validate

pytestmark = pytest.mark.destructive

NIL_UUID = "00000000-0000-0000-0000-000000000000"


# ── Host yuborish ───────────────────────────────────────────────
def test_host_send_chat(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    r = as_mentor.post(f"/api/v1/lessons/{lesson['id']}/chat", json={"body": "Salom sinf"})
    as_mentor.expect(r, 201)
    data = as_mentor.data(r)
    validate(data, CHAT_MESSAGE)
    assert data["body"] == "Salom sinf"


def test_host_send_empty_body_rejected(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    r = as_mentor.post(f"/api/v1/lessons/{lesson['id']}/chat", json={"body": ""})
    as_mentor.expect(r, 400)


def test_host_send_too_long_rejected(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    r = as_mentor.post(f"/api/v1/lessons/{lesson['id']}/chat", json={"body": "x" * 2001})
    as_mentor.expect(r, 400)


def test_host_send_requires_auth(client):
    client.expect(client.post(f"/api/v1/lessons/{NIL_UUID}/chat", json={"body": "x"}), 401)


def test_host_send_other_mentor_forbidden(factory):
    owner = factory.mentor_client()
    other = factory.mentor_client()
    lesson = factory.lesson(owner)
    r = other.post(f"/api/v1/lessons/{lesson['id']}/chat", json={"body": "hack"})
    assert r.status_code in (403, 404), r.status_code


# ── History ─────────────────────────────────────────────────────
def test_chat_history_shape(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    as_mentor.post(f"/api/v1/lessons/{lesson['id']}/chat", json={"body": "birinchi"})
    r = as_mentor.get(f"/api/v1/lessons/{lesson['id']}/chat")
    as_mentor.expect(r, 200)
    validate(r.json(), data_array(CHAT_MESSAGE))
    assert len(as_mentor.data(r)) >= 1


# ── Delete (moderatsiya) ────────────────────────────────────────
def test_delete_own_message(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    msg = as_mentor.data(as_mentor.post(f"/api/v1/lessons/{lesson['id']}/chat", json={"body": "o'chiriladi"}))
    r = as_mentor.delete(f"/api/v1/lessons/{lesson['id']}/chat/{msg['id']}")
    as_mentor.expect(r, 204)
    # Takroriy o'chirish → 404 (atomik WHERE deleted_at IS NULL).
    r2 = as_mentor.delete(f"/api/v1/lessons/{lesson['id']}/chat/{msg['id']}")
    as_mentor.expect(r2, 404)


# ── Transcript ──────────────────────────────────────────────────
def test_transcript_txt(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    as_mentor.post(f"/api/v1/lessons/{lesson['id']}/chat", json={"body": "yozuvga tushadi"})
    r = as_mentor.get(f"/api/v1/lessons/{lesson['id']}/chat/transcript")
    as_mentor.expect(r, 200)
    assert "text/plain" in r.headers.get("Content-Type", "")
    assert "attachment" in r.headers.get("Content-Disposition", "")


def test_transcript_html(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    r = as_mentor.get(f"/api/v1/lessons/{lesson['id']}/chat/transcript", params={"format": "html"})
    as_mentor.expect(r, 200)
    assert "text/html" in r.headers.get("Content-Type", "")


def test_transcript_invalid_format(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    r = as_mentor.get(f"/api/v1/lessons/{lesson['id']}/chat/transcript", params={"format": "pdf"})
    as_mentor.expect(r, 400)


# ── Fayl ulashish (host) — MinIO kerak ──────────────────────────
def test_host_upload_file(requires_minio, as_mentor, factory):
    """Host fayl yuklaydi → 201 ChatMessage.file {name,size,mime,url,expires_in_s} (presigned)."""
    lesson = factory.lesson(as_mentor)
    r = as_mentor.post(
        f"/api/v1/lessons/{lesson['id']}/chat/upload",
        files={"file": ("uy_ishi.txt", b"matematika uy ishi", "text/plain")},
        data={"body": "Uy ishi"},
    )
    as_mentor.expect(r, 201)
    msg = as_mentor.data(r)
    validate(msg, CHAT_MESSAGE)
    assert msg.get("file"), "upload javobida file bo'lishi kerak"
    validate(msg["file"], CHAT_FILE)
    assert msg["file"]["url"].startswith("http"), "file.url presigned havola bo'lishi kerak"


def test_upload_rejects_bad_extension(as_mentor, factory):
    """Ruxsatsiz kengaytma → 400 (MinIO'gача yetmasdan rad etiladi — guard shart emas)."""
    lesson = factory.lesson(as_mentor)
    r = as_mentor.post(
        f"/api/v1/lessons/{lesson['id']}/chat/upload",
        files={"file": ("zararli.exe", b"MZ\x90\x00binary", "application/octet-stream")},
    )
    as_mentor.expect(r, 400)


def test_upload_requires_auth(client):
    r = client.post(
        f"/api/v1/lessons/{NIL_UUID}/chat/upload",
        files={"file": ("x.txt", b"x", "text/plain")},
    )
    client.expect(r, 401)


# ── Guest yo'li (room-token) — LiveKit kerak ────────────────────
def test_guest_send_chat(requires_livekit, as_mentor, factory, client):
    lesson = factory.lesson(as_mentor)
    tok = rooms.host_token(as_mentor, lesson["id"])["token"]
    r = client.post(f"/api/v1/rooms/{lesson['id']}/chat", json={"token": tok, "body": "guest xabar"})
    client.expect(r, 201)
    validate(client.data(r), CHAT_MESSAGE)


def test_guest_send_requires_token(client):
    client.expect(client.post(f"/api/v1/rooms/{NIL_UUID}/chat", json={"body": "x"}), 401)
