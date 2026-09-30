"""Web kontrakt — backend javobi web frontend (`src/api/*.jsx`) KUTGAN maydonlarni beradimi.

`src/api/api.jsx` konverti: success → `json.data` ochiladi; list (api.list) → TO'LIQ
`{data,total,page,limit,total_pages}`; xato → `{code,message}` (`code` web ERROR_UZ
map bilan AYNAN mos). Bu yerda web O'QIYDIGAN maydonlar (ayniqsa NOSTANDART bo'lganlar)
tekshiriladi. Backend shu shakllardan chetlashsa web sindan OLDIN ushlanadi.
"""
import pytest

from lib import rooms
from lib.auth import unique_email
from lib.schemas import ROOM_STATE, ROOM_TOKEN, validate

pytestmark = pytest.mark.destructive

NIL_UUID = "00000000-0000-0000-0000-000000000000"


# ── Xato konverti: {code, message} + AYNIQ kod qiymatlari ────────
# Web ERROR_UZ map (api.jsx) shu kodlarga tayanadi — drift bo'lsa foydalanuvchi
# noto'g'ri/ingliz xabar ko'radi.
def test_error_unauthorized_code(client):
    r = client.get("/api/v1/auth/me")
    assert r.status_code == 401
    assert r.json().get("code") == "UNAUTHORIZED", r.json()


def test_error_forbidden_code(as_user):
    r = as_user.post("/api/v1/lessons", json={"title": "x", "duration_min": 30,
                                              "is_recording_enabled": False, "is_waiting_room_enabled": False})
    assert r.status_code == 403
    assert r.json().get("code") == "FORBIDDEN", r.json()


def test_error_not_found_code(as_mentor):
    r = as_mentor.get(f"/api/v1/lessons/{NIL_UUID}")
    assert r.status_code == 404
    assert r.json().get("code") == "NOT_FOUND", r.json()


def test_error_bad_request_code(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    r = as_mentor.post(f"/api/v1/lessons/{lesson['id']}/polls", json={"question": "x?", "options": ["bitta"]})
    assert r.status_code == 400
    assert r.json().get("code") == "BAD_REQUEST", r.json()


def test_error_conflict_code(admin, factory):
    """Dublikat email bilan user yaratish → 409 CONFLICT."""
    u = factory.user(role="student")
    r = admin.post("/api/v1/users", json={"email": u.email, "password": "parol12345",
                                          "full_name": "Dublikat", "role": "student"})
    assert r.status_code == 409, r.status_code
    assert r.json().get("code") == "CONFLICT", r.json()


# ── List konverti — to'liq {data,total,page,limit,total_pages} ───
@pytest.mark.parametrize("path,fixture_name", [
    ("/api/v1/lessons", "as_mentor"),
    ("/api/v1/notifications", "as_mentor"),
    ("/api/v1/users", "admin"),
])
def test_list_envelope_full_shape(path, fixture_name, request):
    c = request.getfixturevalue(fixture_name)
    body = c.get(path).json()
    for k in ("data", "total", "page", "limit", "total_pages"):
        assert k in body, f"{path}: web api.list {k} kutadi, javob: {list(body.keys())}"
    assert isinstance(body["data"], list)


# ── unread-count → {count} (obyekt, oddiy son EMAS) ─────────────
def test_unread_count_is_object(as_mentor):
    data = as_mentor.data(as_mentor.get("/api/v1/notifications/unread-count"))
    assert isinstance(data, dict) and "count" in data, "web `.count` o'qiydi"


# ── UserShort ro'yxat elementi ──────────────────────────────────
def test_users_list_item_shape(admin, factory):
    factory.user(role="student")
    items = admin.data(admin.get("/api/v1/users"))
    assert len(items) >= 1
    item = items[0]
    for k in ("id", "full_name", "email"):
        assert k in item, f"web UserShort.{k} kutadi"


# ── Blocklist element ───────────────────────────────────────────
def test_blocklist_item_fields_documented(as_mentor):
    # Bo'sh bo'lishi mumkin — shakl BlocklistEntry {id,identity,display_name,created_at}
    # (data_array bo'lgani backend/test_blocklist.py da tekshirilgan; bu yerda web ochishi).
    data = as_mentor.data(as_mentor.get("/api/v1/blocklist"))
    assert isinstance(data, list)


# ── Waiting list item → requester_name (guest_name EMAS) ────────
def test_waiting_list_item_requester_name(as_mentor, factory, client):
    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=True)
    rooms.join(client, lesson["join_slug"], guest_name="Ism Familiya")
    items = as_mentor.data(as_mentor.get(f"/api/v1/lessons/{lesson['id']}/waitingroom"))
    assert len(items) >= 1
    assert "requester_name" in items[0], "web ParticipantsPanel `requester_name` o'qiydi"


# ── joinlink POST → next_step diskriminatori + shartli maydonlar ─
def test_joinlink_next_step_waiting_room(as_mentor, factory, client):
    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=True)
    data = rooms.join(client, lesson["join_slug"], guest_name="G")
    assert data["next_step"] == "waiting_room"
    assert "request_id" in data and "lesson" in data, "web waiting_room'da request_id+lesson o'qiydi"


def test_joinlink_status_shape(as_mentor, factory, client):
    lesson = factory.lesson(as_mentor, is_waiting_room_enabled=True)
    rid = rooms.join(client, lesson["join_slug"], guest_name="G")["request_id"]
    data = client.data(client.get(f"/api/v1/waitingroom/{rid}/status"))
    assert "status" in data, "web WaitingRoom `.status` o'qiydi"


# ── Telegram status — enabled/linked (+ linked bo'lsa telegram_username) ─
def test_telegram_status_web_fields(as_mentor):
    data = as_mentor.data(as_mentor.get("/api/v1/me/telegram"))
    assert "enabled" in data and "linked" in data, "web adaptTelegramStatus enabled/linked o'qiydi"


# ── LiveKit kerak: RoomToken ws_url / room-state / poll results ──
def test_room_token_ws_url(requires_livekit, as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    data = rooms.host_token(as_mentor, lesson["id"])
    validate(data, ROOM_TOKEN)
    assert data["ws_url"], "web LiveRoom `ws_url` (snake_case) o'qiydi"


def test_room_state_web_fields(requires_livekit, as_mentor, factory, client):
    lesson = factory.lesson(as_mentor)
    tok = rooms.host_token(as_mentor, lesson["id"])["token"]
    data = client.data(client.get(f"/api/v1/rooms/{lesson['id']}/state", params={"token": tok}))
    validate(data, ROOM_STATE)
    for k in ("hands", "allow_self_unmute", "recording"):
        assert k in data, f"web LiveRoom room-state `{k}` o'qiydi"
