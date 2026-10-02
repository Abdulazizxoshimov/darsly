"""Blocklist — mentorning doimiy qora ro'yxati (list / unblock).

Kontrakt (api-contract «Room/LiveKit» oxiri):
  GET    /blocklist     → [BlocklistEntry]
  DELETE /blocklist/:id → 204 (unban)
Mentor huquqi; student → 403.
"""
from lib.schemas import BLOCKLIST_ENTRY, data_array, validate

NIL_UUID = "00000000-0000-0000-0000-000000000000"


def test_blocklist_list_shape(as_mentor):
    # Blocklist paginatsiyasiz — oddiy {data:[...]}.
    r = as_mentor.get("/api/v1/blocklist")
    as_mentor.expect(r, 200)
    validate(r.json(), data_array(BLOCKLIST_ENTRY))


def test_blocklist_requires_auth(client):
    client.expect(client.get("/api/v1/blocklist"), 401)


def test_student_cannot_access_blocklist(as_user):
    as_user.expect(as_user.get("/api/v1/blocklist"), 403)


def test_unblock_nonexistent(as_mentor):
    r = as_mentor.delete(f"/api/v1/blocklist/{NIL_UUID}")
    assert r.status_code in (204, 404), r.status_code
